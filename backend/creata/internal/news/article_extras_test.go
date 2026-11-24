package news

import (
	"context"
	"strings"
	"testing"
	"time"
)

func storyUpdate(id, occurredAt string) Doc {
	return Doc{"id": id, "occurredAt": occurredAt, "title": localized("The update"), "body": localized("What changed"), "sourceUrl": "https://example.org/announcement"}
}

func TestArticleExtrasDraftValidation(t *testing.T) {
	const occurredAt = "2026-03-01T14:00:00Z"
	for _, tc := range []struct {
		name string
		data Doc
	}{
		{"wrong guide type", Doc{"actionGuide": []any{}}},
		{"wrong timeline type", Doc{"storyUpdates": Doc{}}},
		{"null timeline", Doc{"storyUpdates": nil}},
		{"null update", Doc{"storyUpdates": []any{nil}}},
		{"duplicate update id", Doc{"storyUpdates": []any{storyUpdate("notice", occurredAt), storyUpdate("notice", occurredAt)}}},
		{"malformed date", Doc{"storyUpdates": []any{storyUpdate("notice", "2026-02-30T14:00:00Z")}}},
		{"invalid offset", Doc{"storyUpdates": []any{storyUpdate("notice", "2026-03-01T14:00:00+24:00")}}},
		{"unsafe id", Doc{"storyUpdates": []any{storyUpdate("#notice/", occurredAt)}}},
		{"credentialed URL", Doc{"actionGuide": Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "https://user:password@example.org/"}}},
		{"unsafe URL", Doc{"actionGuide": Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "javascript:alert(1)"}}},
		{"invalid source host", Doc{"actionGuide": Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "https://[example.org]/rules"}}},
		{"invalid source port", Doc{"actionGuide": Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "https://example.org:65536/rules"}}},
		{"unknown guide field", Doc{"actionGuide": Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "", "internalNote": "private"}}},
		{"missing locale", Doc{"actionGuide": Doc{"audience": Doc{"en": "Students"}, "steps": localized("Read the rules"), "sourceUrl": ""}}},
		{"oversized audience", Doc{"actionGuide": Doc{"audience": localized(strings.Repeat("a", 301)), "steps": localized("Read the rules"), "sourceUrl": ""}}},
		{"unsafe related slug", Doc{"storyUpdates": []any{Doc{"id": "notice", "occurredAt": occurredAt, "title": localized("The update"), "body": localized("What changed"), "sourceUrl": "", "articleSlug": "../private"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newArticle(Doc{"id": "editor"}, tc.data)
			if err := validateArticle(a); err == nil {
				t.Fatal("accepted invalid article extras")
			}
		})
	}
}

func TestArticleExtrasNeedCompleteCopyAndSourceForPublication(t *testing.T) {
	guide := Doc{"audience": Doc{"ru": "", "kk": "", "en": "Students"}, "steps": Doc{"ru": "Read the rules", "kk": "", "en": ""}, "sourceUrl": "https://example.org/rules"}
	a := newArticle(Doc{"id": "editor"}, validArticle())
	applyArticle(a, Doc{"actionGuide": guide})
	if err := validateArticle(a); err != nil {
		t.Fatalf("incomplete draft rejected: %v", err)
	}
	if err := publishable(a); err == nil {
		t.Fatal("accepted text split across locales for publication")
	}
	guide["steps"] = Doc{"ru": "", "kk": "", "en": "Read the rules"}
	guide["sourceUrl"] = ""
	if err := publishable(a); err == nil {
		t.Fatal("accepted action guide without a source")
	}
	guide["sourceUrl"] = "https://example.org/rules"
	if err := publishable(a); err != nil {
		t.Fatalf("a guide with one complete locale should publish: %v", err)
	}
}

func TestStoryUpdateAcceptsGeneratedRelatedArticleSlug(t *testing.T) {
	update := storyUpdate("notice", "2026-03-01T14:00:00Z")
	related := newArticle(Doc{"id": "related-editor"}, Doc{"title": localized("Ручная проверка · акселератор «Орбита»")})
	update["articleSlug"] = related["slug"]
	article := newArticle(Doc{"id": "editor"}, Doc{"storyUpdates": []any{update}})
	if err := validateArticle(article); err != nil {
		t.Fatalf("related article slug %q was rejected: %v", related["slug"], err)
	}
	for _, invalid := range []string{".", "..", "article/other", "article\\other", "article#heading", "article?query", "article\x00"} {
		update["articleSlug"] = invalid
		if err := validateArticle(article); err == nil {
			t.Errorf("accepted unsafe article slug %q", invalid)
		}
	}
}

func TestArticleExtrasPublicProjectionOrdersAndFiltersFields(t *testing.T) {
	first := storyUpdate("first", "2026-03-01T09:00:00Z")
	newer := storyUpdate("newer", "2026-03-01T11:30:00+01:00")
	tie := storyUpdate("tie", "2026-03-01T10:30:00Z")
	newer["reviewNotes"] = "private update notes"
	guide := Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "https://example.org/rules", "internalNote": "private guide notes"}
	a := Doc{"storyUpdates": []any{first, newer, tie}, "actionGuide": guide, "storySlug": "stable-story"}
	public := publicArticle(a)
	for i, expected := range []string{"newer", "tie", "first"} {
		updates := array(public["storyUpdates"])
		if len(updates) != 3 || str(object(updates[i])["id"]) != expected {
			t.Fatalf("unexpected public timeline order: %v", updates)
		}
	}
	if object(array(public["storyUpdates"])[0])["reviewNotes"] != nil || object(public["actionGuide"])["internalNote"] != nil || public["storySlug"] != nil {
		t.Fatal("private fields appeared in public article", public)
	}
	if str(object(array(a["storyUpdates"])[0])["id"]) != "first" {
		t.Fatal("public projection changed editorial order")
	}
}

func TestAPIArticleExtrasFollowPublishedRevision(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	draft := validArticle()
	draft["reviewNotes"] = []any{"Editorial notes"}
	article := a.request("POST", "/admin/articles", draft, 201, "")
	path := "/admin/articles/" + str(article["id"])
	publicPath := "/public/articles/" + str(article["slug"])
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	patch := Doc{
		"expectedRevision": 1,
		"actionGuide":      Doc{"audience": localized("Students"), "steps": localized("Read the rules"), "sourceUrl": "https://example.org/rules"},
		"storyUpdates":     []any{storyUpdate("notice", "2026-03-01T14:00:00Z")},
	}
	a.login("reader")
	a.request("PATCH", path, patch, 403, "")
	a.login("admin")
	a.request("PATCH", path, patch, 200, "")
	a.request("PATCH", path, patch, 409, "")
	public := a.request("GET", publicPath, nil, 200, "")
	if public["actionGuide"] != nil || public["storyUpdates"] != nil {
		t.Fatal("public article exposed an unpublished guide or update", public)
	}
	a.request("POST", path+"/actions", Doc{"action": "approve", "expectedRevision": 2}, 200, "")
	a.request("POST", path+"/actions", Doc{"action": "schedule", "expectedRevision": 2, "scheduledAt": time.Now().Add(time.Hour).Format(time.RFC3339)}, 200, "")
	if public = a.request("GET", publicPath, nil, 200, ""); public["storyUpdates"] != nil {
		t.Fatal("scheduled update was exposed before its release", public)
	}
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		current, err := tx.Get(ctx, "article", str(article["id"]))
		if err != nil {
			return err
		}
		current["scheduledAt"] = time.Now().Add(-time.Minute).Format(time.RFC3339)
		return tx.Put(ctx, "article", current)
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.app.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	public = a.request("GET", publicPath, nil, 200, "")
	if len(array(public["storyUpdates"])) != 1 || str(object(public["actionGuide"])["sourceUrl"]) != "https://example.org/rules" || len(array(public["reviewNotes"])) != 0 {
		t.Fatal("published revision lacks its public extras or exposes private notes", public)
	}
	a.request("PATCH", path, Doc{"expectedRevision": 2, "slug": "relocated-story"}, 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 2, "actionGuide": nil, "storyUpdates": []any{}}, 200, "")
	if public = a.request("GET", publicPath, nil, 200, ""); public["actionGuide"] == nil || len(array(public["storyUpdates"])) != 1 {
		t.Fatal("draft removal changed the published snapshot", public)
	}
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 3}, 200, "")
	}
	if public = a.request("GET", publicPath, nil, 200, ""); public["actionGuide"] != nil || public["storyUpdates"] != nil {
		t.Fatal("public article retained cleared extras", public)
	}
	a.request("POST", path+"/actions", Doc{"action": "unpublish", "expectedRevision": 3}, 200, "")
	a.request("PATCH", path, Doc{"expectedRevision": 3, "slug": "relocated-story"}, 409, "")
}

func TestAPIStoryConversionKeepsLiveArticlePermalink(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	path := "/admin/articles/" + str(article["id"])
	slug := str(article["slug"])
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	updates := []any{storyUpdate("notice", "2026-03-01T14:00:00Z")}
	a.request("PATCH", path, Doc{"expectedRevision": 1, "slug": "draft-slug", "storyUpdates": updates}, 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "slug": "draft-slug"}, 200, "")
	a.request("PATCH", path, Doc{"expectedRevision": 2, "storyUpdates": updates}, 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 2, "slug": slug, "storyUpdates": updates}, 200, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 3}, 200, "")
	}
	public := a.request("GET", "/public/articles/"+slug, nil, 200, "")
	if len(array(public["storyUpdates"])) != 1 {
		t.Fatal("story conversion lost the original public article URL", public)
	}
}

func TestAPIStoryConversionWithChangedPermalinkCannotPublish(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	articleID := str(article["id"])
	path := "/admin/articles/" + articleID
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	ctx := context.Background()
	if err := a.store.Write(ctx, func(tx *Tx) error {
		draft, err := tx.Get(ctx, "article", articleID)
		if err != nil {
			return err
		}
		revise(draft)
		draft["slug"] = "draft-slug"
		draft["storyUpdates"] = []any{storyUpdate("notice", "2026-03-01T14:00:00Z")}
		draft["status"] = "approved"
		draft["approvedRevision"] = draft["revision"]
		draft["scheduledAt"] = time.Now().Add(-time.Minute).Format(time.RFC3339)
		return tx.Put(ctx, "article", draft)
	}); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"approve", "schedule", "publish"} {
		a.request("POST", path+"/actions", Doc{"action": action, "expectedRevision": 2, "scheduledAt": time.Now().Add(time.Hour).Format(time.RFC3339)}, 409, "")
	}
	if err := a.app.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	draft := a.request("GET", path, nil, 200, "")
	if str(draft["status"]) != "review" || draft["scheduledAt"] != nil {
		t.Fatal("scheduler kept the conflicting publication queued", draft)
	}
	public := a.request("GET", "/public/articles/"+str(article["slug"]), nil, 200, "")
	if public["storyUpdates"] != nil {
		t.Fatal("the conflicting story conversion was published", public)
	}
}

func TestAPIFollowedStoriesUseAccountAndPublicTimeline(t *testing.T) {
	a := setupAPI(t)
	const followed = "/account/followed-stories"
	a.request("GET", followed, nil, 401, "")
	a.login("admin")
	create := func(date string, publish bool) string {
		input := validArticle()
		if date != "" {
			input["storyUpdates"] = []any{storyUpdate("announcement", date)}
		}
		input["reviewNotes"] = []any{"Editor's note"}
		article := a.request("POST", "/admin/articles", input, 201, "")
		articleID := str(article["id"])
		if publish {
			for _, action := range []string{"approve", "publish"} {
				a.request("POST", "/admin/articles/"+articleID+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
			}
		}
		return articleID
	}
	newer := create("2026-03-02T14:00:00Z", true)
	older := create("2026-03-01T14:00:00Z", true)
	draft := create("2026-03-03T14:00:00Z", false)
	ordinary := create("", true)
	a.login("reader")
	csrf := a.csrf
	a.csrf = ""
	a.request("POST", followed+"/"+newer, nil, 403, "")
	a.request("DELETE", followed+"/"+newer, nil, 403, "")
	a.csrf = csrf
	a.request("POST", followed+"/"+draft, nil, 404, "")
	a.request("POST", followed+"/"+ordinary, nil, 404, "")
	for _, articleID := range []string{older, newer, older} {
		if !boolean(a.request("POST", followed+"/"+articleID, nil, 200, "")["ok"]) {
			t.Fatal("follow did not report success")
		}
	}
	items := array(a.request("GET", followed, nil, 200, "")["items"])
	if len(items) != 2 || str(object(items[0])["id"]) != newer || str(object(items[1])["id"]) != older || len(array(object(items[0])["reviewNotes"])) != 0 {
		t.Fatal("unexpected followed stories feed", items)
	}
	a.login("author")
	if items = array(a.request("GET", followed, nil, 200, "")["items"]); len(items) != 0 {
		t.Fatal("follows exposed to another account", items)
	}
	a.request("DELETE", followed+"/"+older, nil, 200, "")
	a.login("admin")
	a.request("POST", "/admin/articles/"+newer+"/actions", Doc{"action": "unpublish", "expectedRevision": 1}, 200, "")
	a.login("reader")
	items = array(a.request("GET", followed, nil, 200, "")["items"])
	if len(items) != 1 || str(object(items[0])["id"]) != older {
		t.Fatal("withdrawn story remained visible or another account removed a follow", items)
	}
	a.request("POST", followed+"/"+newer, nil, 404, "")
	for i := 0; i < 2; i++ {
		a.request("DELETE", followed+"/"+newer, nil, 200, "")
	}
	a.login("admin")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+newer+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	a.login("reader")
	if items = array(a.request("GET", followed, nil, 200, "")["items"]); len(items) != 1 || str(object(items[0])["id"]) != older {
		t.Fatal("deleted follow reappeared on republication", items)
	}
}
