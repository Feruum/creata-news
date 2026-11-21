package news

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadCompensatesFailedMediaPersistence(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	_, e := a.store.Pool.Exec(ctx, `CREATE FUNCTION fail_media_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.kind='media' THEN RAISE EXCEPTION 'media write fixture'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_media BEFORE INSERT ON creata_records FOR EACH ROW EXECUTE FUNCTION fail_media_insert()`)
	if e != nil {
		t.Fatal(e)
	}
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	_ = form.WriteField("credit", "Fixture photographer")
	_ = form.WriteField("rights", "Test-only original")
	file, e := form.CreateFormFile("file", "fixture.png")
	if e != nil {
		t.Fatal(e)
	}
	_, _ = file.Write([]byte("\x89PNG\r\n\x1a\nfixture"))
	_ = form.Close()
	req, e := http.NewRequest(http.MethodPost, a.server.URL+base+"/admin/media", body)
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("X-CSRF-Token", a.csrf)
	res, e := a.client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 500 {
		t.Fatal("failed persistence became success", res.StatusCode)
	}
	files, e := os.ReadDir(filepath.Join(a.app.Config.MediaDir, "images"))
	if e != nil {
		t.Fatal(e)
	}
	if len(files) != 0 {
		t.Fatal("orphan object survived failed persistence")
	}
	pending, e := a.store.List(ctx, "media_cleanup")
	if e != nil || len(pending) != 0 {
		t.Fatal("compensation left unnecessary cleanup", pending, e)
	}
}

func TestMediaLibraryRightsReferencesAndDeletion(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	key := "images/library.png"
	target := filepath.Join(a.app.Config.MediaDir, "images", "library.png")
	if e := os.MkdirAll(filepath.Dir(target), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(target, []byte("fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	d := Doc{"id": "library", "key": key, "url": "https://media.example.org/library.png", "legacyPath": "/news/photos/library.png", "credit": "Photographer", "rights": "Review", "needsRightsReview": true, "revision": 1}
	if e := a.store.Write(ctx, func(tx *Tx) error { return tx.Put(ctx, "media", d) }); e != nil {
		t.Fatal(e)
	}
	a.request("PATCH", "/admin/media/library", Doc{"expectedRevision": 1, "credit": "Photographer", "rights": "Permission on file", "confirmRights": true}, 200, "")
	a.request("PATCH", "/admin/media/library", Doc{"expectedRevision": 1, "rights": "stale"}, 409, "")
	records := array(a.request("GET", "/admin/media?q=Photographer", nil, 200, "")["items"])
	if len(records) != 1 || boolean(object(records[0])["needsRightsReview"]) {
		t.Fatal("rights update missing")
	}
	if e := a.store.Write(ctx, func(tx *Tx) error { return tx.Put(ctx, "site_page_revision", Doc{"id": "old", "photo": d["url"]}) }); e != nil {
		t.Fatal(e)
	}
	a.request("DELETE", "/admin/media/library", Doc{"expectedRevision": 2}, 409, "")
	if e := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "site_page_revision", Doc{"id": "old", "photo": "/news/photos/library.png"})
	}); e != nil {
		t.Fatal(e)
	}
	a.request("DELETE", "/admin/media/library", Doc{"expectedRevision": 2}, 409, "")
	if e := a.store.Write(ctx, func(tx *Tx) error { return tx.Delete(ctx, "site_page_revision", "old") }); e != nil {
		t.Fatal(e)
	}
	a.login("author")
	a.request("DELETE", "/admin/media/library", Doc{"expectedRevision": 2}, 403, "")
	a.login("admin")
	a.request("DELETE", "/admin/media/library", Doc{"expectedRevision": 2}, 200, "")
	if _, e := os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("unreferenced file survived deletion")
	}
}

func TestAssetImportIsIdempotentAndRewritesBackendDocuments(t *testing.T) {
	a := setupAPI(t)
	ctx := context.Background()
	assets := t.TempDir()
	if e := os.MkdirAll(filepath.Join(assets, "news"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(assets, "news", "photo.png"), []byte("\x89PNG\r\n\x1a\nfixture"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := a.store.Write(ctx, func(tx *Tx) error {
		return tx.Put(ctx, "opportunity", Doc{"id": "imported", "photo": "/news/photo.png", "originalSource": "https://example.org/news/photo.png"})
	}); e != nil {
		t.Fatal(e)
	}
	bundle := Doc{"pages": []any{Doc{"slug": "about", "content": aboutContentFixture("Original")}}}
	for i := 0; i < 2; i++ {
		if e := a.app.ImportContent(ctx, bundle, assets); e != nil {
			t.Fatal(e)
		}
	}
	records, e := a.store.List(ctx, "media")
	if e != nil || len(records) != 1 {
		t.Fatal("duplicate import", records, e)
	}
	photo, e := a.store.Get(ctx, "opportunity", "imported")
	if e != nil || str(photo["photo"]) != str(records[0]["url"]) {
		t.Fatal("legacy path not rewritten")
	}
	if !boolean(records[0]["needsRightsReview"]) {
		t.Fatal("legacy rights invented")
	}
	if str(photo["originalSource"]) != "https://example.org/news/photo.png" {
		t.Fatal("external source URL was corrupted")
	}
	oldURL := str(records[0]["url"])
	a.app.Config.APIURL = "http://localhost:5981"
	if e = a.app.ImportContent(ctx, bundle, filepath.Join(assets, "missing")); e == nil {
		t.Fatal("invalid import directory accepted")
	}
	unchanged, e := a.store.Get(ctx, "media", str(records[0]["id"]))
	if e != nil || str(unchanged["url"]) != oldURL {
		t.Fatal("failed import partially migrated media URL")
	}
	if e = a.app.ImportContent(ctx, bundle, assets); e != nil {
		t.Fatal(e)
	}
	updated, e := a.store.Get(ctx, "media", str(records[0]["id"]))
	if e != nil {
		t.Fatal(e)
	}
	photo, e = a.store.Get(ctx, "opportunity", "imported")
	if e != nil || str(updated["url"]) == oldURL || str(photo["photo"]) != str(updated["url"]) {
		t.Fatal("port migration did not atomically update document references")
	}
}

func TestDeletedImportedAssetStaysDeletedOnRestart(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	ctx := context.Background()
	assets := t.TempDir()
	if e := os.MkdirAll(filepath.Join(assets, "news", "photos"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(assets, "news", "photos", "unused.png"), []byte("\x89PNG\r\n\x1a\nfixture"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := a.app.ImportContent(ctx, Doc{}, assets); e != nil {
		t.Fatal(e)
	}
	all, e := a.store.List(ctx, "media")
	if e != nil || len(all) != 1 {
		t.Fatal("import failed", e)
	}
	a.request("DELETE", "/admin/media/"+str(all[0]["id"]), Doc{"expectedRevision": 1}, 200, "")
	if e := a.app.ImportContent(ctx, Doc{}, assets); e != nil {
		t.Fatal(e)
	}
	all, e = a.store.List(ctx, "media")
	if e != nil || len(all) != 0 {
		t.Fatal("startup resurrected intentionally deleted media", e)
	}
}
