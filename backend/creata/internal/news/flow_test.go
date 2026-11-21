package news

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestOAuthReturnTargetDoesNotAllowOpenRedirect(t *testing.T) {
	s := &Server{Config: Config{Origins: []string{"http://localhost:6868", "https://editor.creata.news"}}}
	for _, raw := range []string{"https://evil.example/", "//evil.example/", "https://editor.creata.news.evil.example/", "https://editor.creata.news@evil.example/", "javascript:alert(1)", "/\\evil.example/"} {
		if _, e := s.returnTarget(raw); e == nil {
			t.Fatalf("unsafe redirect accepted %q", raw)
		}
	}
	for _, raw := range []string{"/ru/profile", "http://localhost:6868/articles/123", "https://editor.creata.news/materials"} {
		target, e := s.returnTarget(raw)
		if e != nil || target != raw {
			t.Fatalf("trusted return rejected %q: %v", raw, e)
		}
	}
}

func TestFullHermesToPublishedDigestFlow(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	source := a.request("POST", "/admin/sources", Doc{"name": "Science feed", "type": "rss", "url": "https://example.org/feed", "locale": "en", "intervalMinutes": 15, "active": false}, 200, "")
	a.request("POST", "/admin/sources/"+str(source["id"])+"/run", nil, 202, "")
	claim := func(kind string) Doc {
		j := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "pipeline", "kinds": []any{kind}}, 200, a.worker)["job"])
		if str(j["id"]) == "" {
			t.Fatalf("%s job missing", kind)
		}
		return j
	}
	complete := func(j, result Doc) {
		a.request("POST", "/internal/jobs/"+str(j["id"])+"/complete", Doc{"leaseToken": j["leaseToken"], "result": result}, 200, a.worker)
	}
	complete(claim("collect"), Doc{"items": []any{Doc{"url": "https://example.org/science", "title": "Science advance from a primary public source", "text": "A research paper describes a measured result.", "locale": "en"}}})
	complete(claim("classify"), Doc{"categoryId": "science", "region": "KZ", "tags": []any{"research"}, "score": 90, "reason": "Relevant original scientific research"})
	draftJob := claim("draft")
	aid := str(object(object(draftJob["payload"])["article"])["id"])
	complete(draftJob, validArticle())
	article := a.request("GET", "/admin/articles/"+aid, nil, 200, "")
	if str(article["status"]) != "review" || integer(article["revision"]) != 2 {
		t.Fatalf("AI draft must enter review: %v", article)
	}
	if len(array(a.request("GET", "/public/articles", nil, 200, "")["items"])) != 0 {
		t.Fatal("AI bypassed editorial approval")
	}
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+aid+"/actions", Doc{"action": action, "expectedRevision": 2}, 200, "")
	}
	a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "98765", "locale": "kk"}, 200, a.bot)
	digest := a.request("POST", "/internal/bot/digest", Doc{"telegramId": "98765"}, 200, a.bot)
	if len(array(digest["articles"])) != 1 {
		t.Fatal("published AI article missing from digest")
	}
	a.request("POST", "/internal/bot/digests/"+str(digest["id"])+"/complete", Doc{"telegramId": "98765", "messageId": 123}, 200, a.bot)
	next := a.request("POST", "/internal/bot/digest", Doc{"telegramId": "98765"}, 200, a.bot)
	if len(array(next["articles"])) != 0 {
		t.Fatal("confirmed manual digest content repeated")
	}
}

func TestUncertainManualDeliveryExcludesContentUntilOperatorResolves(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	art := a.request("POST", "/admin/articles", validArticle(), 201, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+str(art["id"])+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "44444", "locale": "ru"}, 200, a.bot)
	digest := a.request("POST", "/internal/bot/digest", Doc{"telegramId": "44444"}, 200, a.bot)
	a.request("POST", "/internal/bot/digests/"+str(digest["id"])+"/fail", Doc{"telegramId": "44444", "error": "network timeout after send", "unknown": true}, 200, a.bot)
	next := a.request("POST", "/internal/bot/digest", Doc{"telegramId": "44444"}, 200, a.bot)
	if len(array(next["articles"])) != 0 {
		t.Fatal("uncertain content was selected again")
	}
	deliveries := array(a.request("GET", "/admin/deliveries", nil, 200, "")["items"])
	unknownID := ""
	for _, raw := range deliveries {
		v := object(raw)
		if str(v["status"]) == "unknown" {
			unknownID = str(v["id"])
		}
		if v["leaseToken"] != nil {
			t.Fatal("worker lease token leaked to admin")
		}
	}
	if unknownID == "" {
		t.Fatal("uncertain delivery not visible")
	}
	a.request("POST", "/admin/deliveries/"+unknownID+"/resolve", Doc{"action": "mark_sent", "messageId": 42}, 200, "")
}

func TestMediaRequiresCreditAndImageData(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	req, _ := http.NewRequest("POST", a.server.URL+base+"/admin/media", strings.NewReader("not a multipart image"))
	req.Header.Set("X-CSRF-Token", a.csrf)
	res, e := a.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("unexpected media status %d", res.StatusCode)
	}
}

func TestCollectedBatchKeepsValidItemsAndAllSourceEvidence(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	src := a.request("POST", "/admin/sources", Doc{"name": "Evidence", "type": "rss", "url": "https://example.org/feed", "locale": "en", "intervalMinutes": 15, "active": false}, 200, "")
	run := func(items []any) {
		a.request("POST", "/admin/sources/"+str(src["id"])+"/run", nil, 202, "")
		j := object(a.request("POST", "/internal/jobs/claim", Doc{"workerId": "evidence", "kinds": []any{"collect"}}, 200, a.worker)["job"])
		a.request("POST", "/internal/jobs/"+str(j["id"])+"/complete", Doc{"leaseToken": j["leaseToken"], "result": Doc{"items": items}}, 200, a.worker)
	}
	title := "A shared long title about a primary source and an independent corroboration"
	run([]any{Doc{"url": "https://example.org/empty", "title": "Empty feed entry", "text": ""}, Doc{"url": "https://example.org/primary", "title": title, "text": "Primary factual evidence.", "locale": "en"}})
	run([]any{Doc{"url": "https://another.example/corroboration", "title": title, "text": "Independent factual evidence.", "locale": "en"}})
	bank := array(a.request("GET", "/admin/bank", nil, 200, "")["items"])
	if len(bank) != 2 {
		t.Fatalf("different evidence must await editor review: %d", len(bank))
	}
	left, right := object(bank[0]), object(bank[1])
	a.request("POST", "/admin/bank/"+str(left["id"])+"/merge", Doc{"otherId": right["id"], "expectedUpdatedAt": left["updatedAt"], "otherExpectedUpdatedAt": right["updatedAt"]}, 200, "")
	story, getErr := a.store.Get(context.Background(), "story", str(left["id"]))
	if getErr != nil {
		t.Fatal(getErr)
	}
	if len(array(story["evidence"])) != 2 || len(array(story["sources"])) != 2 {
		t.Fatalf("secondary evidence was lost: %v", story)
	}
	raw, e := a.store.List(context.Background(), "raw_item")
	if e != nil || len(raw) != 2 {
		t.Fatalf("immutable source originals missing: %v %d", e, len(raw))
	}
}
