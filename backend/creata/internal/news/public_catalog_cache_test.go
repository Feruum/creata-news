package news

import (
	"context"
	"net/http"
	"testing"
)

func TestPublicCatalogValidatorChangesOnlyAfterPublishedChanges(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	id := str(article["id"])
	actionPath := "/admin/articles/" + id + "/actions"
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", actionPath, Doc{"action": action, "expectedRevision": 1}, 200, "")
	}

	get := func(path, tag string, status int) string {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, a.server.URL+base+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if tag != "" {
			req.Header.Set("If-None-Match", tag)
		}
		out, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer out.Body.Close()
		if out.StatusCode != status {
			t.Fatalf("%s returned %d, want %d", path, out.StatusCode, status)
		}
		return out.Header.Get("ETag")
	}
	detail := "/public/articles/" + str(article["slug"])
	list := "/public/articles?limit=10"
	first := get(detail, "", 200)
	if first == "" {
		t.Fatal("published detail must include a validator")
	}
	get(detail, first, 304)
	firstList := get(list, first, 200)
	if firstList == "" || firstList == first {
		t.Fatal("a catalog validator must belong to one response URL")
	}
	a.request("PATCH", "/admin/articles/"+id, Doc{
		"expectedRevision": 1,
		"title":            localized("Updated title"),
	}, 200, "")
	get(detail, first, 304)
	get(list, firstList, 304)

	for _, action := range []string{"approve", "publish"} {
		a.request("POST", actionPath, Doc{"action": action, "expectedRevision": 2}, 200, "")
	}
	second := get(detail, first, 200)
	get(list, firstList, 200)
	get(detail, second, 304)
	authorID := str(article["authorId"])
	if err := a.store.Write(context.Background(), func(tx *Tx) error {
		return tx.Put(context.Background(), "author", Doc{"id": authorID, "name": "Updated author"})
	}); err != nil {
		t.Fatal(err)
	}
	third := get(detail, second, 200)
	a.request("POST", actionPath, Doc{"action": "unpublish", "expectedRevision": 2}, 200, "")
	get(detail, third, 404)
}
