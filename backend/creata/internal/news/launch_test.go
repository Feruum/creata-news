package news

import (
	"strings"
	"testing"
)

func validLaunch() Doc {
	return Doc{"slug": "creata-studio", "name": "Creata Studio", "tagline": localized("Build your next idea"), "description": localized("Tools for early-stage teams."), "team": localized("A team of creators"), "url": "https://example.org/product", "launchDate": "2026-11-14", "country": "KZ", "categoryId": "startups", "photo": ""}
}

func TestLaunchValidation(t *testing.T) {
	d := validLaunch()
	d["status"] = "draft"
	if e := validateLaunch(d); e != nil {
		t.Fatal(e)
	}
	for key, value := range map[string]any{"slug": "../unsafe", "url": "javascript:alert(1)", "launchDate": "2026-02-30", "status": "approved", "photo": "//example.org/image.png", "tagline": Doc{"ru": "Missing locales"}} {
		bad := clone(d)
		bad[key] = value
		if validateLaunch(bad) == nil {
			t.Errorf("accepted invalid %s", key)
		}
	}
}

func TestLaunchAcceptsUnicodeWithinTextLimits(t *testing.T) {
	d := validLaunch()
	d["status"] = "draft"
	d["name"] = strings.Repeat("Ә", 200)
	d["country"] = strings.Repeat("Қ", 100)
	d["description"] = localized(strings.Repeat("Ө", 20000))
	if e := validateLaunch(d); e != nil {
		t.Fatal(e)
	}
	d["name"] = strings.Repeat("Ә", 201)
	if validateLaunch(d) == nil {
		t.Fatal("accepted a name longer than 200 characters")
	}
}

func TestLaunchCatalogPublicationAndPermissions(t *testing.T) {
	a := setupAPI(t)
	a.request("POST", "/admin/launches", validLaunch(), 401, "")
	a.login("author")
	a.request("POST", "/admin/launches", validLaunch(), 403, "")
	a.login("admin")
	d := a.request("POST", "/admin/launches", validLaunch(), 200, "")
	path := "/admin/launches/" + str(d["id"])
	a.request("GET", "/public/launches/creata-studio", nil, 404, "")
	if len(array(a.request("GET", "/public/launches", nil, 200, "")["items"])) != 0 {
		t.Fatal("draft leaked into catalog")
	}
	a.request("POST", "/admin/launches", validLaunch(), 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 0, "status": "published"}, 409, "")
	a.request("PATCH", path, Doc{"expectedRevision": 1, "slug": "new-slug"}, 400, "")
	published := a.request("PATCH", path, Doc{"expectedRevision": 1, "status": "published"}, 200, "")
	if integer(published["revision"]) != 2 {
		t.Fatal("revision not incremented")
	}
	a.request("GET", "/public/launches/creata-studio", nil, 200, "")
	if len(array(a.request("GET", "/public/launches?q=studio&category=startups&country=KZ", nil, 200, "")["items"])) != 1 {
		t.Fatal("matching launch missing")
	}
	if len(array(a.request("GET", "/public/launches?country=US", nil, 200, "")["items"])) != 0 {
		t.Fatal("country filter ignored")
	}
	a.request("PATCH", path, Doc{"expectedRevision": 2, "status": "archived"}, 200, "")
	a.request("GET", "/public/launches/creata-studio", nil, 404, "")
	a.request("POST", "/auth/logout", nil, 200, "")
	a.csrf = ""
	a.request("PATCH", path, Doc{"expectedRevision": 3, "status": "published"}, 401, a.worker)
}
