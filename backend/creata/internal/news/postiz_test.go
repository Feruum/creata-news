package news

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type postizFixture struct {
	mu       sync.Mutex
	posts    []Doc
	uploads  []Doc
	failures map[string]int
	states   map[string]string
}

func attachPostiz(t *testing.T, a *apiTest, mode string) *postizFixture {
	t.Helper()
	f := &postizFixture{failures: map[string]int{}, states: map[string]string{}}
	handler := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "fixture-key" {
			http.Error(w, "unauthorized", 401)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /public/v1/integrations":
			respond(w, 200, []Doc{
				{"id": "one", "name": "News", "identifier": "threads", "disabled": false}, {"id": "two", "name": "Another", "identifier": "x"},
				{"id": "photo", "name": "News photos", "identifier": "instagram"}, {"id": "video", "name": "News video", "identifier": "youtube"},
				{"id": "off", "name": "Disabled", "identifier": "threads", "disabled": true}, {"id": "tg", "name": "Creata", "identifier": "telegram"},
			})
		case "POST /public/v1/upload-from-url":
			var d Doc
			if e := json.NewDecoder(r.Body).Decode(&d); e != nil {
				t.Error(e)
			}
			f.uploads = append(f.uploads, d)
			respond(w, 200, Doc{"id": "media-123", "path": "https://uploads.example.org/media"})
		case "POST /public/v1/posts":
			var d Doc
			if e := json.NewDecoder(r.Body).Decode(&d); e != nil {
				t.Error(e)
				return
			}
			post := object(array(d["posts"])[0])
			channel := str(object(post["integration"])["id"])
			code := f.failures[channel]
			delete(f.failures, channel)
			if code == 429 {
				w.Header().Set("Retry-After", "3")
				w.WriteHeader(code)
				return
			}
			f.posts = append(f.posts, d)
			if code == 503 {
				w.WriteHeader(code)
				return
			} // created upstream but response lost
			postID := fmt.Sprintf("post-%d", len(f.posts))
			respond(w, 200, []Doc{{"postId": postID, "integration": channel}})
		case "GET /public/v1/posts":
			if r.URL.Query().Get("startDate") == "" || r.URL.Query().Get("endDate") == "" {
				t.Error("calendar missing dates")
			}
			items := []Doc{}
			for i, req := range f.posts {
				post := object(array(req["posts"])[0])
				content := object(array(post["value"])[0])["content"]
				remoteID := fmt.Sprintf("post-%d", i+1)
				state := f.states[remoteID]
				if state == "" {
					state = "DRAFT"
					if req["type"] == "now" {
						state = "QUEUE"
					}
				}
				items = append(items, Doc{"id": remoteID, "integration": post["integration"], "content": content, "state": state, "releaseURL": "https://social.example.org/" + remoteID})
			}
			respond(w, 200, Doc{"posts": items})
		default:
			t.Errorf("unexpected Postiz request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(handler.Close)
	config := a.app.Config
	config.Environment, config.PostizAPIURL, config.PostizAPIKey, config.PostizMode = "test", handler.URL+"/public/v1", "fixture-key", mode
	var e error
	a.app.postiz, e = newPostizClient(config)
	if e != nil {
		t.Fatal(e)
	}
	return f
}

func postizAction(a *apiTest, article Doc, action string, extras Doc, status int) Doc {
	a.t.Helper()
	d := Doc{"action": action, "expectedRevision": article["revision"]}
	for k, v := range extras {
		d[k] = v
	}
	return a.request("POST", "/admin/articles/"+str(article["id"])+"/actions", d, status, "")
}
func postizItems(a *apiTest, article Doc) map[string]Doc {
	a.t.Helper()
	out := map[string]Doc{}
	for _, v := range array(a.request("GET", "/admin/articles/"+str(article["id"])+"/postiz", nil, 200, "")["items"]) {
		item := object(v)
		out[str(item["channelId"])+"/"+str(item["locale"])] = item
	}
	return out
}
func postizRun(t *testing.T, a *apiTest) {
	t.Helper()
	if e := a.app.postizTick(context.Background()); e != nil {
		t.Fatal(e)
	}
}
func makeDue(t *testing.T, a *apiTest) {
	t.Helper()
	if _, e := a.store.Pool.Exec(context.Background(), "UPDATE creata_postiz_throttle SET until_at=now()"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.store.Pool.Exec(context.Background(), "UPDATE creata_postiz_delivery SET next_at=now()"); e != nil {
		t.Fatal(e)
	}
}
func selections(ids ...string) Doc {
	list := []Doc{}
	for _, id := range ids {
		list = append(list, Doc{"channelId": id, "locale": "en"})
	}
	return Doc{"postizTargets": list}
}

func TestPostizApprovedFanoutRetriesAndReconciliation(t *testing.T) {
	a := setupAPI(t)
	a.request("GET", "/admin/postiz/channels", nil, 401, "")
	a.login("admin")
	if c := a.request("GET", "/admin/postiz/channels", nil, 200, ""); c["configured"] != false || len(array(c["items"])) != 0 || c["mode"] != "draft" {
		t.Fatal(c)
	}
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	postizAction(a, article, "publish", selections("one"), 409)
	postizAction(a, article, "approve", nil, 200)
	postizAction(a, article, "publish", selections("one"), 409)
	f := attachPostiz(t, a, "")
	c := a.request("GET", "/admin/postiz/channels", nil, 200, "")
	if len(array(c["items"])) != 6 || object(array(c["items"])[4])["supported"] != false || object(array(c["items"])[5])["supported"] != false {
		t.Fatal(c)
	}
	postizAction(a, article, "publish", selections("off"), 400)
	postizAction(a, article, "publish", selections("one", "one"), 400)
	preview := a.request("GET", "/admin/articles/"+str(article["id"])+"/postiz/preview?provider=threads&locale=en", nil, 200, "")
	cfg := selections("one", "two", "photo")
	cfg["postizTargets"] = append(cfg["postizTargets"].([]Doc), Doc{"channelId": "one", "locale": "kk"})
	postizAction(a, article, "publish", cfg, 200)
	postizAction(a, article, "publish", cfg, 200)
	items := postizItems(a, article)
	if len(items) != 4 || items["one/en"]["content"] != preview["content"] || items["photo/en"]["status"] != "missing_media" {
		t.Fatal(items)
	}
	public := a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	b, _ := json.Marshal(public)
	if strings.Contains(string(b), "postiz") || strings.Contains(string(b), "channelId") {
		t.Fatal("private account leaked")
	}
	f.failures["two"] = 429
	for i := 0; i < 4; i++ {
		postizRun(t, a)
	}
	if len(f.posts) > 2 || postizItems(a, article)["two/en"]["status"] != "queued" {
		t.Fatal("rate limited job should be delayed", f.posts)
	}
	makeDue(t, a)
	for i := 0; i < 3; i++ {
		postizRun(t, a)
	}
	if len(f.posts) != 3 {
		t.Fatal(f.posts)
	}
	items = postizItems(a, article)
	x := items["two/en"]
	if x["status"] != "draft" || integer(x["attempts"]) != 2 || x["postId"] == "" {
		t.Fatal(x)
	}
	foundX := false
	for _, req := range f.posts {
		post := object(array(req["posts"])[0])
		if req["type"] != "draft" || post["settings"] == nil || len(array(post["value"])) != 1 {
			t.Fatal(req)
		}
		if object(post["integration"])["id"] == "two" {
			foundX = true
			if object(post["settings"])["who_can_reply_post"] != "everyone" {
				t.Fatal(post)
			}
		}
	}
	if !foundX {
		t.Fatal("X settings not exercised")
	}
	f.states[str(x["postId"])] = "PUBLISHED"
	makeDue(t, a)
	postizRun(t, a)
	x = postizItems(a, article)["two/en"]
	if x["status"] != "published" || x["releaseUrl"] != "https://social.example.org/"+str(x["postId"]) {
		t.Fatal(x)
	}
	postizAction(a, article, "unpublish", nil, 200)
	if postizItems(a, article)["two/en"]["status"] != "published" {
		t.Fatal("withdrawal erased confirmed post")
	}
}

func TestPostizScheduleCrashWithdrawalAndSeed(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	f := attachPostiz(t, a, "now")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	postizAction(a, article, "approve", nil, 200)
	schedule := selections("one", "two")
	schedule["scheduledAt"] = time.Now().Add(time.Hour).Format(time.RFC3339)
	postizAction(a, article, "schedule", schedule, 200)
	postizRun(t, a)
	if len(f.posts) != 0 || postizItems(a, article)["one/en"]["status"] != "scheduled" {
		t.Fatal("posted before scheduled publication")
	}
	article = a.request("PATCH", "/admin/articles/"+str(article["id"]), Doc{"expectedRevision": 1, "title": localized("Next revision")}, 200, "")
	if postizItems(a, article)["one/en"]["status"] != "cancelled" {
		t.Fatal("editing did not cancel schedule")
	}
	postizAction(a, article, "approve", nil, 200)
	postizAction(a, article, "schedule", schedule, 200)
	ctx := context.Background()
	if _, e := a.store.Pool.Exec(ctx, `UPDATE creata_records SET data=jsonb_set(data,'{scheduledAt}',to_jsonb(now()-interval '1 hour')) WHERE kind='article' AND id=$1`, article["id"]); e != nil {
		t.Fatal(e)
	}
	if e := a.app.Tick(ctx); e != nil {
		t.Fatal(e)
	}
	f.failures["one"] = 503
	if _, e := a.store.Pool.Exec(ctx, "UPDATE creata_postiz_delivery SET next_at=now()+interval '5 hours' WHERE channel_id='two'"); e != nil {
		t.Fatal(e)
	}
	postizRun(t, a)
	if len(f.posts) != 1 {
		t.Fatal(f.posts)
	}
	var status string
	if e := a.store.Pool.QueryRow(ctx, "SELECT status FROM creata_postiz_delivery WHERE revision=2 AND channel_id='one'").Scan(&status); e != nil || status != "uncertain" {
		t.Fatal(status, e)
	}
	// Simulate death between sending and saving the Postiz response. A new
	// tick must reconcile by account and version-specific content, never POST.
	if _, e := a.store.Pool.Exec(ctx, "UPDATE creata_postiz_delivery SET status='sending',next_at=now() WHERE channel_id='one' AND revision=2"); e != nil {
		t.Fatal(e)
	}
	postizRun(t, a)
	if e := a.store.Pool.QueryRow(ctx, "SELECT status FROM creata_postiz_delivery WHERE revision=2 AND channel_id='one'").Scan(&status); e != nil || status != "accepted" {
		t.Fatal(status, e)
	}
	postizAction(a, article, "unpublish", nil, 200)
	makeDue(t, a)
	postizRun(t, a)
	if len(f.posts) != 1 {
		t.Fatal("ambiguous post duplicated or withdrawn post sent")
	}
	if e := a.store.Pool.QueryRow(ctx, "SELECT status FROM creata_postiz_delivery WHERE revision=2 AND channel_id='two'").Scan(&status); e != nil || status != "cancelled" {
		t.Fatal(status, e)
	}
	seed := validArticle()
	seed["id"] = "seed-postiz"
	seed["slug"] = "seed-only"
	seed["authorId"] = article["authorId"]
	seed["status"] = "published"
	seed["postizTargets"] = schedule["postizTargets"]
	data, _ := json.Marshal(Doc{"articles": []Doc{seed}})
	if e := a.store.Seed(ctx, data); e != nil {
		t.Fatal(e)
	}
	if len(postizItems(a, seed)) != 0 {
		t.Fatal("seed created delivery")
	}
}

func TestPostizMediaAndConcurrentWorkers(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	f := attachPostiz(t, a, "now")
	a.app.Config.APIURL = "https://cdn.example.org"
	var b bytes.Buffer
	if e := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 480, 480)), nil); e != nil {
		t.Fatal(e)
	}
	mp4 := []byte("\x00\x00\x00\x18ftypisom\x00\x00\x00\x00isommp42")
	for _, data := range [][]byte{b.Bytes(), mp4} {
		ctx := context.Background()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		_ = form.WriteField("credit", "Desk")
		_ = form.WriteField("rights", "Licensed")
		file, e := form.CreateFormFile("file", "media")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = file.Write(data); e != nil {
			t.Fatal(e)
		}
		if e = form.Close(); e != nil {
			t.Fatal(e)
		}
		req, e := http.NewRequest("POST", a.server.URL+base+"/admin/media", &body)
		if e != nil {
			t.Fatal(e)
		}
		req.Header.Set("Content-Type", form.FormDataContentType())
		req.Header.Set("X-CSRF-Token", a.csrf)
		res, e := a.client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		var m Doc
		e = json.NewDecoder(res.Body).Decode(&m)
		res.Body.Close()
		if res.StatusCode != http.StatusCreated || e != nil {
			t.Fatal(m, res.StatusCode, e)
		}
		aData := validArticle()
		config := selections("photo", "video")
		if m["mime"] == "image/jpeg" {
			aData["media"] = m
		} else {
			// Dedicated social video, independent of article cover.
			config["postizTargets"].([]Doc)[1]["mediaId"] = m["id"]
		}
		article := a.request("POST", "/admin/articles", aData, 201, "")
		postizAction(a, article, "approve", nil, 200)
		if m["mime"] == "video/mp4" {
			config["scheduledAt"] = time.Now().Add(time.Hour).Format(time.RFC3339)
			postizAction(a, article, "schedule", config, 200)
			a.request("DELETE", "/admin/media/"+str(m["id"]), Doc{"expectedRevision": m["revision"]}, 409, "")
		}
		postizAction(a, article, "publish", config, 200)
		a.request("DELETE", "/admin/media/"+str(m["id"]), Doc{"expectedRevision": m["revision"]}, 409, "")
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if e := a.app.postizTick(ctx); e != nil {
					t.Error(e)
				}
			}()
		}
		wg.Wait()
		want, blocked := "photo", "video"
		if m["mime"] == "video/mp4" {
			want, blocked = blocked, want
		}
		items := postizItems(a, article)
		if items[want+"/en"]["status"] != "accepted" || items[blocked+"/en"]["status"] != "missing_media" {
			t.Fatal(items)
		}
	}
	if len(f.posts) != 2 || len(f.uploads) != 2 {
		t.Fatal(f.posts, f.uploads)
	}
	for _, req := range f.posts {
		post := object(array(req["posts"])[0])
		settings := object(post["settings"])
		media := object(array(object(array(post["value"])[0])["image"])[0])
		if media["id"] != "media-123" || media["path"] != "https://uploads.example.org/media" {
			t.Fatal(post)
		}
		if settings["__type"] == "youtube" && (settings["type"] != "private" || settings["title"] != "A tested story") {
			t.Fatal(settings)
		}
		if settings["__type"] == "instagram" && settings["post_type"] != "post" {
			t.Fatal(settings)
		}
	}
}

func TestPostizConfigAndGraphicPreview(t *testing.T) {
	for _, c := range []Config{{PostizMode: "scheduled"}, {PostizAPIURL: "http://127.0.0.1", Environment: "production"}, {PostizAPIURL: "https://usr:pw@fixture.invalid"}} {
		if _, e := newPostizClient(c); e == nil {
			t.Fatal("accepted invalid config")
		}
	}
	a := Doc{"revision": 2, "slug": "graphic 1", "format": "graphic", "title": localized(strings.Repeat("界", 400))}
	server := Server{Config: Config{PublicURL: "https://news.example.org"}}
	content, link, e := server.postizText(a, "x", "kk")
	if e != nil || !strings.Contains(link, "/kk/charts/graphic%201?utm_content=creata-r2") || !strings.HasSuffix(content, link) || len([]rune(content)) >= 280 {
		t.Fatal(content, link, e)
	}
}

func TestPostizRollbackAndRetryBeforeConnect(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	f := attachPostiz(t, a, "draft")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	postizAction(a, article, "approve", nil, 200)
	ctx := context.Background()
	// Fail after publish() has written its snapshot. Neither publication nor
	// its outbox records may survive the aborted editorial transaction.
	_, e := a.store.Pool.Exec(ctx, `CREATE FUNCTION reject_queue() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='queued' THEN RAISE EXCEPTION 'fixture rollback'; END IF; RETURN NEW; END $$;
		CREATE TRIGGER reject_queue BEFORE UPDATE ON creata_postiz_delivery FOR EACH ROW EXECUTE FUNCTION reject_queue()`)
	if e != nil {
		t.Fatal(e)
	}
	postizAction(a, article, "publish", selections("one"), 500)
	a.request("GET", "/public/articles/"+str(article["slug"]), nil, 404, "")
	if len(postizItems(a, article)) != 0 {
		t.Fatal("outbox survived aborted publication")
	}
	if _, e = a.store.Pool.Exec(ctx, "DROP TRIGGER reject_queue ON creata_postiz_delivery"); e != nil {
		t.Fatal(e)
	}
	p := a.app.postiz
	transport := p.http.Transport
	p.catalog = nil
	p.expires = time.Time{}
	p.http.Transport = &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("fixture: unavailable before connect")
	}}
	// The last verified account selection survives a restart and upstream outage.
	postizAction(a, article, "publish", selections("one"), 200)
	a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	postizRun(t, a)
	if item := postizItems(a, article)["one/en"]; item["status"] != "queued" || integer(item["attempts"]) != 1 || len(f.posts) != 0 {
		t.Fatal(item)
	}
	p.http.Transport = transport
	makeDue(t, a)
	postizRun(t, a)
	if item := postizItems(a, article)["one/en"]; item["status"] != "draft" || integer(item["attempts"]) != 2 || len(f.posts) != 1 {
		t.Fatal(item)
	}
}
