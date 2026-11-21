package news

import "testing"

func newspaperFixture(articleID string) Doc {
	return Doc{"slug": "02", "number": "02", "date": "2026-11", "title": "Creata News: November", "description": "People and ideas", "locale": "ru", "cover": Doc{"url": "https://example.org/cover.jpg", "credit": "Creata", "rights": "Original photography", "caption": "Cover"}, "items": []any{Doc{"id": "lead", "articleId": articleID}}}
}
func publishedNewspaperArticle(a *apiTest) Doc {
	d := a.request("POST", "/admin/articles", validArticle(), 201, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+str(d["id"])+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	return d
}
func TestNewspaperDraftPublicationAndSnapshot(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	article := publishedNewspaperArticle(a)
	d := a.request("POST", "/admin/newspapers", newspaperFixture(str(article["id"])), 201, "")
	path := "/admin/newspapers/" + str(d["id"])
	a.request("GET", "/public/newspapers/02", nil, 404, "")
	if len(array(a.request("GET", "/public/newspapers", nil, 200, "")["items"])) != 0 {
		t.Fatal("draft in archive")
	}
	a.request("POST", path+"/publish", Doc{"expectedRevision": 0}, 409, "")
	a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 200, "")
	p := a.request("GET", "/public/newspapers/02", nil, 200, "")
	if localize(object(object(array(p["entries"])[0])["article"])["title"], "ru") != "A tested story" {
		t.Fatal("published article missing")
	}
	a.request("PATCH", path, Doc{"expectedRevision": 1, "title": "A new draft"}, 200, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "title": "Stale overwrite"}, 409, "")
	p = a.request("GET", "/public/newspapers/02", nil, 200, "")
	if str(p["title"]) != "Creata News: November" {
		t.Fatal("draft changed public snapshot")
	}
	a.request("POST", path+"/publish", Doc{"expectedRevision": 2}, 200, "")
	a.request("POST", path+"/publish", Doc{"expectedRevision": 2}, 200, "")
	if len(array(a.request("GET", "/public/newspapers", nil, 200, "")["items"])) != 1 {
		t.Fatal("duplicate publication")
	}
	a.request("POST", path+"/unpublish", Doc{"expectedRevision": 2}, 200, "")
	a.request("GET", "/public/newspapers/02", nil, 404, "")
}
func TestNewspaperPermissionsAndUnpublishedArticles(t *testing.T) {
	a := setupAPI(t)
	a.request("POST", "/admin/newspapers", Doc{}, 401, "")
	a.login("admin")
	article := a.request("POST", "/admin/articles", validArticle(), 201, "")
	d := a.request("POST", "/admin/newspapers", newspaperFixture(str(article["id"])), 201, "")
	path := "/admin/newspapers/" + str(d["id"])
	a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 400, "")
	a.login("editor")
	a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 403, "")
	a.login("author")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "title": "Other author's edit"}, 403, "")
	a.login("reader")
	a.request("GET", path, nil, 403, "")
	a.request("POST", "/auth/logout", nil, 200, "")
	a.csrf = ""
	a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 401, a.worker)
}
func TestNewspaperValidationAndStableOrdering(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	first := publishedNewspaperArticle(a)
	input := validArticle()
	input["title"] = localized("Second article")
	second := a.request("POST", "/admin/articles", input, 201, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+str(second["id"])+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	doc := newspaperFixture(str(first["id"]))
	doc["items"] = []any{Doc{"id": "second", "articleId": second["id"]}, Doc{"id": "first", "articleId": first["id"]}}
	d := a.request("POST", "/admin/newspapers", doc, 201, "")
	path := "/admin/newspapers/" + str(d["id"])
	a.request("POST", "/admin/newspapers", doc, 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "cover": Doc{"url": "javascript:alert(1)", "credit": "C", "rights": "Own"}}, 400, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "date": "2026-13"}, 400, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "slug": "03"}, 400, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "items": []any{Doc{"id": "a", "articleId": first["id"]}, Doc{"id": "b", "articleId": first["id"]}}}, 400, "")
	a.request("POST", path+"/publish", Doc{"expectedRevision": 1}, 200, "")
	p := a.request("GET", "/public/newspapers/02", nil, 200, "")
	if str(object(array(p["entries"])[0])["articleId"]) != str(second["id"]) {
		t.Fatal("article order lost")
	}
	a.request("PATCH", "/admin/articles/"+str(second["id"]), Doc{"expectedRevision": 1, "title": localized("Changed article")}, 200, "")
	p = a.request("GET", "/public/newspapers/02", nil, 200, "")
	if localize(object(object(array(p["entries"])[0])["article"])["title"], "ru") != "Second article" {
		t.Fatal("article edits mutated issue snapshot")
	}
}
