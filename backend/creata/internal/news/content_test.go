package news

import (
	"context"
	"testing"
	"time"
)

func aboutContentFixture(title string) Doc {
	languages := Doc{}
	for _, l := range []string{"ru", "kk", "en"} {
		languages[l] = Doc{"title": title, "body": "A publication for creators.", "second": "Sources and opportunities.", "values": []any{[]any{"Trust", "Verified sources"}}}
	}
	return languages
}

func TestSiteContentRevisionsPublicationAndPermissions(t *testing.T) {
	a := setupAPI(t)
	a.request("GET", "/public/pages/about", nil, 404, "")
	a.login("admin")
	// Import is explicit and idempotent; initialization never creates demo pages.
	bundle := Doc{"pages": []any{Doc{"slug": "about", "content": aboutContentFixture("Original")}}}
	if e := a.app.ImportContent(context.Background(), bundle, ""); e != nil {
		t.Fatal(e)
	}
	p := a.request("GET", "/public/pages/about", nil, 200, "")
	if str(object(object(p["content"])["ru"])["title"]) != "Original" {
		t.Fatal("missing imported page")
	}
	a.request("PATCH", "/admin/pages/about", Doc{"expectedRevision": 1, "content": aboutContentFixture("Draft")}, 200, "")
	a.request("PATCH", "/admin/pages/about", Doc{"expectedRevision": 1, "content": aboutContentFixture("Stale")}, 409, "")
	if e := a.app.ImportContent(context.Background(), bundle, ""); e != nil {
		t.Fatal(e)
	}
	p = a.request("GET", "/public/pages/about", nil, 200, "")
	if str(object(object(p["content"])["ru"])["title"]) != "Original" {
		t.Fatal("draft leaked to public")
	}
	a.login("editor")
	a.request("POST", "/admin/pages/about/publish", Doc{"expectedRevision": 2}, 403, "")
	a.login("admin")
	a.request("POST", "/admin/pages/about/publish", Doc{"expectedRevision": 1}, 409, "")
	a.request("POST", "/admin/pages/about/publish", Doc{"expectedRevision": 2}, 200, "")
	if len(array(a.request("GET", "/admin/pages/about/revisions", nil, 200, "")["items"])) != 2 {
		t.Fatal("revision history missing")
	}
	a.request("POST", "/admin/pages/about/unpublish", Doc{"expectedRevision": 2}, 200, "")
	a.request("GET", "/public/pages/about", nil, 404, "")
	a.login("reader")
	a.request("GET", "/admin/pages", nil, 403, "")
}

func TestSiteContentRejectsUnsafeAndIncompletePublication(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	if e := a.app.ImportContent(context.Background(), Doc{"pages": []any{Doc{"slug": "about", "content": aboutContentFixture("About")}}}, ""); e != nil {
		t.Fatal(e)
	}
	broken := aboutContentFixture("About")
	delete(broken, "kk")
	a.request("PATCH", "/admin/pages/about", Doc{"expectedRevision": 1, "content": broken}, 400, "")
	a.request("PATCH", "/admin/pages/about", Doc{"expectedRevision": 1, "content": aboutContentFixture("javascript:alert(1)")}, 200, "") // text remains escaped by React
	original := Doc{"slug": "01", "number": "01", "date": "2026-10", "locale": "ru", "title": "Original", "description": "Archive", "pages": 36, "cover": Doc{"url": "https://example.org/cover.jpg", "credit": "Archive", "rights": "Legacy import; review required"}, "html": "<article><script>alert(1)</script></article>"}
	if e := a.app.ImportContent(context.Background(), Doc{"pages": []any{Doc{"slug": "newspaper-01", "content": original}}}, ""); e == nil {
		t.Fatal("unsafe original HTML accepted")
	}
	original["html"] = "<div class=\"desk\"><article class=\"sheet\"><h1>Original</h1></article></div>"
	if e := a.app.ImportContent(context.Background(), Doc{"pages": []any{Doc{"slug": "newspaper-01", "content": original}}}, ""); e != nil {
		t.Fatal(e)
	}
	a.request("GET", "/public/newspapers/01", nil, 200, "")
	a.request("POST", "/admin/pages/newspaper-01/unpublish", Doc{"expectedRevision": 1}, 200, "")
	a.request("GET", "/public/newspapers/01", nil, 404, "")
	article := publishedNewspaperArticle(a)
	managed := newspaperFixture(str(article["id"]))
	managed["slug"] = "01"
	a.request("POST", "/admin/newspapers", managed, 409, "")
	a.request("POST", "/admin/pages/newspaper-01/publish", Doc{"expectedRevision": 1}, 200, "")
	p := a.request("GET", "/public/newspapers/01", nil, 200, "")
	if integer(p["revision"]) != 1 {
		t.Fatal("original publication revision is missing")
	}
}

func TestOriginalIssueValidationDoesNotWaitForWriteLock(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	const slug = "newspaper-01"
	const path = "/admin/pages/" + slug
	content := Doc{
		"slug": "01", "number": "01", "date": "2026-10", "locale": "ru", "title": "Original", "pages": 1,
		"cover": Doc{"url": "/cover.jpg"},
		"html":  "<article><form><button>Search</button></form></article>",
	}
	if e := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "site_page", Doc{"id": slug, "slug": slug, "content": content, "revision": 1, "status": "draft"})
	}); e != nil {
		t.Fatal(e)
	}
	lock, e := a.store.Pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Rollback(ctx)
	if _, e = lock.Exec(ctx, "SELECT pg_advisory_xact_lock(70801126)"); e != nil {
		t.Fatal(e)
	}
	a.client.Timeout = 3 * time.Second
	const want = "Active HTML is forbidden in an archived issue"
	if got := a.request("PATCH", path, Doc{"expectedRevision": 1, "content": content}, 400, "")["error"]; got != want {
		t.Fatalf("save error = %q, want %q", got, want)
	}
	if got := a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 400, "")["error"]; got != want {
		t.Fatalf("publish error = %q, want %q", got, want)
	}
}

func TestEcosystemMetadataMigrationPreservesDifferentDraftAndPublication(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	if e := a.store.Write(ctx, func(tx *Tx) error {
		for _, entry := range []struct {
			kind, id, marker string
			revision         int
		}{{"site_page", "ecosystem", "draft", 2}, {"site_page_publication", "ecosystem", "public", 1}, {"site_page_revision", "ecosystem:1", "history", 1}} {
			if e := tx.Put(ctx, entry.kind, Doc{"id": entry.id, "slug": "ecosystem", "revision": entry.revision, "content": Doc{"marker": entry.marker}}); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	bundle := Doc{"pages": []any{Doc{"slug": "ecosystem", "content": Doc{"issueSlug": "01", "issueNumber": "01"}}}}
	for i := 0; i < 2; i++ {
		if e := a.app.ImportContent(ctx, bundle, ""); e != nil {
			t.Fatal(e)
		}
	}
	for _, entry := range []struct {
		kind, id, marker string
		revision         int
	}{{"site_page", "ecosystem", "draft", 2}, {"site_page_publication", "ecosystem", "public", 1}, {"site_page_revision", "ecosystem:1", "history", 1}} {
		d, e := a.store.Get(ctx, entry.kind, entry.id)
		if e != nil || integer(d["revision"]) != entry.revision || str(object(d["content"])["marker"]) != entry.marker || str(object(d["content"])["issueSlug"]) != "01" {
			t.Fatal("migration overwrote a stored revision", entry, d, e)
		}
	}
}
