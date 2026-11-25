package news

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type apiTest struct {
	t           *testing.T
	app         *Server
	store       *Store
	server      *httptest.Server
	client      *http.Client
	csrf        string
	worker, bot string
}

func TestTelegramChannelOnlyClaimsConfiguredChannel(t *testing.T) {
	a := setupAPI(t)
	a.app.Config.TelegramChannelOnly = true
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		if err := tx.Put(ctx, "delivery", Doc{"id": "channel", "chatId": a.app.Config.ChannelID, "status": "queued"}); err != nil {
			return err
		}
		return tx.Put(ctx, "delivery", Doc{"id": "personal", "chatId": "20", "status": "queued"})
	}); err != nil {
		t.Fatal(err)
	}
	claim := func() Doc {
		return object(a.request("POST", "/internal/deliveries/claim", Doc{"workerId": "pilot"}, 200, a.bot)["delivery"])
	}
	if got := str(claim()["id"]); got != "channel" {
		t.Fatalf("claimed %q, expected the configured channel delivery", got)
	}
	if got := str(claim()["id"]); got != "" {
		t.Fatalf("claimed %q outside the configured channel", got)
	}
}

func setupAPI(t *testing.T) *apiTest {
	t.Helper()
	dsn := os.Getenv("CREATA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CREATA_TEST_DATABASE_URL is required for PostgreSQL integration tests")
	}
	u, e := url.Parse(dsn)
	if e != nil || !isLoopback(u.Hostname()) || !strings.Contains(u.Path, "test") {
		t.Fatal("integration database must be loopback and have test in its name")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schemaName := "test_" + id()
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schemaName}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	q.Set("search_path", schemaName)
	u.RawQuery = q.Encode()
	store, e := Open(ctx, u.String())
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Initialize(ctx); e != nil {
		t.Fatal(e)
	}
	cfg := Config{SessionSecret: strings.Repeat("s", 40), WorkerToken: strings.Repeat("w", 40), BotServiceToken: strings.Repeat("b", 40), Environment: "development", DevAuth: true, PublicURL: "http://localhost:3002", APIURL: "http://localhost:5555", ChannelID: "-1001234", BotUsername: "creata_test_bot", MediaDir: t.TempDir(), AllowPrivateSources: true}
	app, e := New(store, cfg)
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	jar, _ := cookiejar.New(nil)
	a := &apiTest{t: t, app: app, store: store, server: server, client: &http.Client{Jar: jar}, worker: cfg.WorkerToken, bot: cfg.BotServiceToken}
	t.Cleanup(func() {
		server.Close()
		store.Pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schemaName}.Sanitize()+" CASCADE")
		admin.Close()
	})
	return a
}
func (a *apiTest) request(method, path string, body any, status int, service string) Doc {
	a.t.Helper()
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req, e := http.NewRequest(method, a.server.URL+base+path, bytes.NewReader(data))
	if e != nil {
		a.t.Fatal(e)
	}
	req.Header.Set("Content-Type", "application/json")
	if service != "" {
		req.Header.Set("Authorization", "Bearer "+service)
	} else if a.csrf != "" {
		req.Header.Set("X-CSRF-Token", a.csrf)
	}
	res, e := a.client.Do(req)
	if e != nil {
		a.t.Fatal(e)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != status {
		a.t.Fatalf("%s %s status %d wanted %d: %s", method, path, res.StatusCode, status, raw)
	}
	var d Doc
	if e = json.Unmarshal(raw, &d); e != nil {
		a.t.Fatalf("bad JSON %s: %v", raw, e)
	}
	return d
}
func (a *apiTest) login(role string) {
	a.request("POST", "/auth/dev-login", Doc{"role": role}, 200, "")
	me := a.request("GET", "/auth/me", nil, 200, "")
	a.csrf = str(me["csrfToken"])
	if a.csrf == "" {
		a.t.Fatal("CSRF token missing")
	}
}
func validArticle() Doc {
	return Doc{"title": localized("A tested story"), "lead": localized("A factual lead"), "body": localized("A factual body with sources."), "summary": localized("Summary"), "actions": localized("Read more"), "categoryId": "ai", "sources": []any{Doc{"url": "https://example.org/source", "title": "Primary source"}}}
}

func TestAPIEditorialRevisionWorkflow(t *testing.T) {
	a := setupAPI(t)
	a.request("GET", "/admin/articles", nil, 401, "")
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	aid := str(article["id"])
	if len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 0 {
		t.Fatal("draft leaked")
	}
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "publish", "expectedRevision": 1}, 409, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "approve", "expectedRevision": 1}, 200, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "publish", "expectedRevision": 1}, 200, "")
	public := a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	if localize(public["title"], "en") != "A tested story" {
		t.Fatal(public)
	}
	changed := a.request("PATCH", "/admin/articles/"+aid, Doc{"expectedRevision": 1, "title": localized("Changed headline")}, 200, "")
	if integer(changed["revision"]) != 2 || integer(changed["approvedRevision"]) != 0 {
		t.Fatal("approval survived edit")
	}
	public = a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	if localize(public["title"], "en") != "A tested story" {
		t.Fatal("unapproved edit leaked")
	}
	a.request("PATCH", "/admin/articles/"+aid, Doc{"expectedRevision": 1, "title": localized("Stale edit")}, 409, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "approve", "expectedRevision": 2}, 200, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "publish", "expectedRevision": 2, "note": "Corrected headline"}, 200, "")
	public = a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	if len(array(public["corrections"])) != 1 {
		t.Fatal("public correction missing")
	}
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "unpublish", "expectedRevision": 2}, 200, "")
	a.request("GET", "/public/articles/"+str(article["slug"]), nil, 404, "")
	a.login("reader")
	a.request("GET", "/admin/articles", nil, 403, "")
}

func TestAPIWorkerLeasesDedupAndStaleDraft(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	source := a.request("POST", "/admin/sources", Doc{"name": "Test RSS", "type": "rss", "url": "https://example.org/rss", "locale": "en", "intervalMinutes": 15, "active": false}, 200, "")
	path := "/admin/sources/" + str(source["id"]) + "/run"
	a.request("POST", path, nil, 202, "")
	a.request("POST", "/internal/jobs/claim", Doc{"workerId": "bad"}, 401, "")
	job := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "worker-a", "kinds": []any{"collect"}}, 200, a.worker)["job"])
	jid := str(job["id"])
	a.request("POST", "/internal/jobs/"+jid+"/complete", Doc{"leaseToken": "stolen", "result": Doc{}}, 409, a.worker)
	result := Doc{"items": []any{Doc{"url": "https://example.org/news?utm_source=x", "title": "A sufficiently long headline for source grouping of scientific news", "text": "Some source text.", "locale": "en"}}}
	a.request("POST", "/internal/jobs/"+jid+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 200, a.worker)
	a.request("POST", "/internal/jobs/"+jid+"/complete", Doc{"leaseToken": job["leaseToken"], "result": result}, 200, a.worker)
	a.request("POST", path, nil, 202, "")
	second := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "worker-a", "kinds": []any{"collect"}}, 200, a.worker)["job"])
	a.request("POST", "/internal/jobs/"+str(second["id"])+"/complete", Doc{"leaseToken": second["leaseToken"], "result": result}, 200, a.worker)
	bank := array(a.request("GET", "/admin/bank", nil, 200, "")["items"])
	if len(bank) != 1 {
		t.Fatalf("dedup failed: %d stories", len(bank))
	}
	story := object(bank[0])
	draft := a.request("POST", "/admin/bank/"+str(story["id"])+"/draft", nil, 201, "")
	draftJob := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "worker-a", "kinds": []any{"draft"}}, 200, a.worker)["job"])
	a.request("PATCH", "/admin/articles/"+str(draft["id"]), Doc{"expectedRevision": 1, "body": localized("Human edited body")}, 200, "")
	ai := validArticle()
	a.request("POST", "/internal/jobs/"+str(draftJob["id"])+"/complete", Doc{"leaseToken": draftJob["leaseToken"], "result": ai}, 409, a.worker)
	current := a.request("GET", "/admin/articles/"+str(draft["id"]), nil, 200, "")
	if localize(current["body"], "ru") != "Human edited body" {
		t.Fatal("AI overwrote human change")
	}
	if len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 0 {
		t.Fatal("worker published a draft")
	}
}

func TestAPIBudgetIsAtomicAndSecretsArePrivate(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	a.request("POST", "/internal/ai/reserve", Doc{"requestId": "unconfigured", "estimatedCostUsd": 0.1}, 503, a.worker)
	settings := a.request("PATCH", "/admin/settings", Doc{"ai": Doc{"endpoint": "https://example.org/v1", "model": "test-model", "apiKey": "test-private-key", "dailyBudgetUsd": 1, "inputPricePerMillion": 1, "outputPricePerMillion": 1}}, 200, "")
	raw, _ := json.Marshal(settings)
	if strings.Contains(string(raw), "test-private-key") || !boolean(object(settings["ai"])["keyConfigured"]) {
		t.Fatal("secret leaked or configuration missing")
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, key := range []string{"one", "two"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			data, _ := json.Marshal(Doc{"requestId": key, "estimatedCostUsd": 0.7})
			req, _ := http.NewRequest("POST", a.server.URL+base+"/internal/ai/reserve", bytes.NewReader(data))
			req.Header.Set("Authorization", "Bearer "+a.worker)
			res, e := http.DefaultClient.Do(req)
			if e != nil {
				statuses <- 0
				return
			}
			res.Body.Close()
			statuses <- res.StatusCode
		}(key)
	}
	wg.Wait()
	close(statuses)
	success, limited := 0, 0
	for code := range statuses {
		if code == 200 {
			success++
		}
		if code == 429 {
			limited++
		}
	}
	if success != 1 || limited != 1 {
		t.Fatalf("budget race: success %d limited %d", success, limited)
	}
}

func TestAPIBotDigestLinkAndDeliveryIdempotency(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	aid := str(article["id"])
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "12345", "locale": "kk"}, 200, a.bot)
	a.request("PATCH", "/internal/bot/users/12345", Doc{"locale": "en", "topics": []any{"ai"}}, 200, a.bot)
	u := a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "12345", "locale": "ru"}, 200, a.bot)
	if str(u["locale"]) != "en" {
		t.Fatal("upsert reset chosen language")
	}
	digest := a.request("POST", "/internal/bot/digest", Doc{"telegramId": "12345"}, 200, a.bot)
	if len(array(digest["articles"])) != 1 {
		t.Fatal("published story missing from digest")
	}
	channel := a.request("POST", "/admin/digests/preview", Doc{"locale": "ru", "kind": "channel"}, 201, "")
	did := str(channel["id"])
	a.request("POST", "/admin/digests/"+did+"/send", nil, 409, "")
	a.request("POST", "/admin/digests/"+did+"/approve", nil, 200, "")
	a.request("POST", "/admin/digests/"+did+"/send", nil, 200, "")
	a.request("POST", "/admin/digests/"+did+"/send", nil, 200, "")
	delivery := object(a.request("POST", "/internal/deliveries/claim", Doc{"workerId": "bot-a"}, 200, a.bot)["delivery"])
	a.request("POST", "/internal/deliveries/"+str(delivery["id"])+"/complete", Doc{"leaseToken": delivery["leaseToken"], "messageId": 100}, 200, a.bot)
	a.request("POST", "/internal/deliveries/"+str(delivery["id"])+"/complete", Doc{"leaseToken": delivery["leaseToken"], "messageId": 100}, 200, a.bot)
	next := a.request("POST", "/internal/deliveries/claim", Doc{"workerId": "bot-a"}, 200, a.bot)
	if next["delivery"] != nil {
		t.Fatal("confirmed delivery was repeated")
	}
	link := a.request("POST", "/account/telegram-link", Doc{}, 200, "")
	parts := strings.Split(str(link["url"]), "link_")
	if len(parts) != 2 {
		t.Fatal(link)
	}
	a.request("POST", "/internal/bot/link", Doc{"token": parts[1], "telegramId": "12345"}, 200, a.bot)
	a.request("POST", "/internal/bot/link", Doc{"token": parts[1], "telegramId": "12345"}, 400, a.bot)
}

func TestScheduledPublicationAndLeaseRecovery(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	aid := str(article["id"])
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "approve", "expectedRevision": 1}, 200, "")
	scheduled := a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "schedule", "expectedRevision": 1, "scheduledAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, 200, "")
	if str(scheduled["status"]) != "scheduled" || integer(scheduled["approvedRevision"]) != 1 || str(scheduled["scheduledAt"]) == "" {
		t.Fatal("scheduled article was not retained as an approved scheduled revision", scheduled)
	}
	rescheduled := a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "schedule", "expectedRevision": 1, "scheduledAt": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)}, 200, "")
	if str(rescheduled["status"]) != "scheduled" || str(rescheduled["scheduledAt"]) == str(scheduled["scheduledAt"]) {
		t.Fatal("rescheduling did not update publication time", rescheduled)
	}
	if len(array(a.request("GET", "/admin/articles?status=scheduled", nil, 200, "")["items"])) != 1 || len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 0 {
		t.Fatal("scheduled article not present in the correct collections")
	}
	if e := a.store.Write(context.Background(), func(tx *Tx) error {
		v, e := tx.Get(context.Background(), "article", aid)
		if e != nil {
			return e
		}
		v["scheduledAt"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
		return tx.Put(context.Background(), "article", v)
	}); e != nil {
		t.Fatal(e)
	}
	if e := a.app.Tick(context.Background()); e != nil {
		t.Fatal(e)
	}
	if len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 1 {
		t.Fatal("scheduled publication not visible")
	}
	withdrawn := a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "unpublish", "expectedRevision": 1}, 200, "")
	if str(withdrawn["status"]) != "unpublished" || withdrawn["approvedRevision"] != nil || withdrawn["publishedRevision"] != nil || withdrawn["scheduledAt"] != nil {
		t.Fatal("withdrawal did not clear publication and approval state", withdrawn)
	}
	if len(array(a.request("GET", "/admin/articles?status=unpublished", nil, 200, "")["items"])) != 1 || len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 0 {
		t.Fatal("withdrawn article not present in the correct collections")
	}
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "publish", "expectedRevision": 1}, 409, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "approve", "expectedRevision": 1}, 200, "")
	a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "schedule", "expectedRevision": 1, "scheduledAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, 200, "")
	republished := a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": "publish", "expectedRevision": 1}, 200, "")
	if str(republished["status"]) != "published" || republished["scheduledAt"] != nil {
		t.Fatal("an approved scheduled article could not be immediately published", republished)
	}

	j := newJob("check", Doc{})
	j["status"] = "running"
	j["leaseToken"] = "old-token"
	j["leaseUntil"] = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	j["attempts"] = 1
	if e := a.store.Write(context.Background(), func(tx *Tx) error { return tx.Put(context.Background(), "job", j) }); e != nil {
		t.Fatal(e)
	}
	claimed := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "replacement", "kinds": []any{"check"}}, 200, a.worker)["job"])
	if str(claimed["id"]) != str(j["id"]) || str(claimed["leaseToken"]) == "old-token" {
		t.Fatal("abandoned lease was not recovered")
	}
}
