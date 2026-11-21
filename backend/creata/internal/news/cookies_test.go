package news

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSharedCookieDomainValidation(t *testing.T) {
	baseConfig := func() Config {
		return Config{Environment: "production", APIURL: "https://api.creata.news", PublicURL: "https://creata.news", Origins: []string{"https://creata.news", "https://editor.creata.news", "https://hermes.creata.news"}, GoogleClientID: "client", GoogleClientSecret: "secret"}
	}
	for _, domain := range []string{"", "com", "co.uk", "evil.example", "api.creata.news", "127.0.0.1"} {
		c := baseConfig()
		c.CookieDomain = domain
		if e := validateCookieDomain(&c); e == nil {
			t.Fatalf("invalid domain accepted %q", domain)
		}
	}
	c := baseConfig()
	c.CookieDomain = ".creata.news"
	if e := validateCookieDomain(&c); e != nil || c.CookieDomain != "creata.news" {
		t.Fatalf("shared domain rejected: %v", e)
	}
	same := baseConfig()
	same.APIURL = "https://creata.news"
	same.Origins = []string{"https://creata.news"}
	if e := validateCookieDomain(&same); e != nil {
		t.Fatal(e)
	}
	local := Config{APIURL: "http://localhost:5555", PublicURL: "http://localhost:3002", Origins: []string{"http://localhost:6868"}, GoogleClientID: "local"}
	if e := validateCookieDomain(&local); e != nil {
		t.Fatal(e)
	}
}

func TestSessionAndOAuthStateUseSharedCookieDomain(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	a.app.Config.CookieDomain = "creata.news"
	a.app.Config.PublicURL = "https://creata.news"
	a.app.Config.APIURL = "https://api.creata.news"
	a.app.Config.GoogleClientID = "client"
	a.app.Config.GoogleClientSecret = "secret"
	a.app.Config.Origins = []string{"https://editor.creata.news"}
	u, e := a.store.Get(context.Background(), "user", "dev-admin")
	if e != nil {
		t.Fatal(e)
	}
	session := httptest.NewRecorder()
	if e = a.app.session(session, context.Background(), u); e != nil {
		t.Fatal(e)
	}
	cookies := session.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Domain != "creata.news" || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("session cookie domain/flags incorrect")
	}
	a.app.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"issuer":"https://accounts.google.com","authorization_endpoint":"https://accounts.google.com/o/oauth2/v2/auth","token_endpoint":"https://oauth2.googleapis.com/token","jwks_uri":"https://www.googleapis.com/oauth2/v3/certs"}`))}, nil
	})}
	req := httptest.NewRequest("GET", "https://editor.creata.news"+base+"/auth/google/start?returnTo=https%3A%2F%2Feditor.creata.news%2Fmaterials", nil)
	req.SetPathValue("provider", "google")
	response := httptest.NewRecorder()
	if e = a.app.oauthStart(response, req); e != nil {
		t.Fatal(e)
	}
	cookies = response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Domain != "creata.news" || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("OAuth state cookie domain/flags incorrect")
	}
}

func TestAuthorCanQueueManualLinkWithoutSourceAdministration(t *testing.T) {
	a := setupAPI(t)
	a.login("author")
	a.request("POST", "/admin/sources", Doc{"name": "not allowed", "url": "https://example.org/feed"}, 403, "")
	a.request("POST", "/admin/bank/collect", Doc{"url": "javascript:alert(1)", "locale": "ru"}, 400, "")
	result := a.request("POST", "/admin/bank/collect", Doc{"url": "https://example.org/new-story?utm_source=test", "locale": "kk", "topics": []any{"science"}}, 202, "")
	job := object(result["job"])
	source := object(object(job["payload"])["source"])
	if str(job["createdBy"]) != "dev-author" || !boolean(source["singlePage"]) || boolean(source["active"]) || str(source["url"]) != "https://example.org/new-story" {
		t.Fatalf("manual submission was not isolated/traced: %v", result)
	}
}

func TestExpiredApprovedOpportunityRemainsAddressable(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	o := a.request("POST", "/admin/opportunities", Doc{"title": localized("Expired opportunity"), "description": localized("Details retained for saved records"), "organization": "Creata", "type": "grant", "url": "https://example.org/grant", "country": "KZ", "deadline": "2020-01-01T00:00:00Z", "status": "approved"}, 200, "")
	a.request("GET", "/public/opportunities/"+str(o["id"]), nil, 200, "")
	feed := a.request("GET", "/public/opportunities", nil, 200, "")
	if len(array(feed["items"])) != 0 {
		t.Fatal("expired item leaked into current feed")
	}
}
