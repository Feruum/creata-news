package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ory "github.com/ory/kratos-client-go"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/hermes"
)

type hermesLinks struct {
	identity string
	link     hermes.TelegramLink
	err      error
	calls    int
}

func (s *hermesLinks) UserCount(context.Context) (int, error) { return 42, s.err }

func TestHermesUserCountRequiresScopedServiceCredential(t *testing.T) {
	f := newHermesFixture(t)
	key, err := os.ReadFile(f.tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("studyverse/hermes/user-count/v1"))
	scoped := hex.EncodeToString(mac.Sum(nil))
	for _, tc := range []struct {
		credential, cookie string
		want               int
	}{
		{"", "", 403}, {string(key), "", 403}, {scoped, "ory_kratos_session=valid", 403}, {scoped, "", 200},
	} {
		r := httptest.NewRequest(http.MethodGet, "/internal/hermes/user-count", nil)
		r.Header.Set("Authorization", "Bearer "+tc.credential)
		r.Header.Set("Cookie", tc.cookie)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("counter status = %d, want %d", w.Code, tc.want)
		}
		if tc.want == 200 && strings.TrimSpace(w.Body.String()) != `{"users":42}` {
			t.Fatalf("counter returned %s", w.Body)
		}
	}
}

func (s *hermesLinks) VerifiedLink(_ context.Context, identity string) (hermes.TelegramLink, error) {
	s.calls++
	s.identity = identity
	return s.link, s.err
}

type hermesFixture struct {
	session      *ory.Session
	kratosStatus int
	kratosCalls  int
	opsCalls     int
	opsStatus    int
	opsBody      string
	bound        string
	lastPayload  map[string]interface{}
	lastPath     string
	tokenFile    string
	links        *hermesLinks
	handler      http.Handler
}

func ownerSession() *ory.Session {
	s := ory.NewSession("hermes-browser-session")
	s.SetActive(true)
	s.SetExpiresAt(time.Now().Add(time.Hour))
	s.Identity = ory.NewIdentity("owner-kratos-id", "default", "https://identity.test/schema", map[string]interface{}{"email": "owner@example.test"})
	s.Identity.SetState("active")
	s.Identity.VerifiableAddresses = []ory.VerifiableIdentityAddress{*ory.NewVerifiableIdentityAddress("completed", "owner@example.test", true, "email")}
	return s
}

func newHermesFixture(t *testing.T) *hermesFixture {
	t.Helper()
	f := &hermesFixture{session: ownerSession(), kratosStatus: 200, bound: "owner-kratos-id", links: &hermesLinks{
		link: hermes.TelegramLink{ID: "987654321", LinkedAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)},
	}}
	kratos := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.kratosCalls++
		if r.URL.Path != "/sessions/whoami" || r.Header.Get("Cookie") != "ory_kratos_session=valid" {
			t.Error("fresh browser cookie did not reach Kratos")
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Session-Token") != "" {
			t.Error("client supplied token reached Kratos")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(f.kratosStatus)
		if f.kratosStatus != 200 {
			_, _ = w.Write([]byte(`{"error":{"message":"private-kratos-diagnostic"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(f.session)
	}))
	t.Cleanup(kratos.Close)
	token := strings.Repeat("s", 48)
	ops := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.opsCalls++
		f.lastPath = r.URL.Path
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("internal request missing private bearer authentication")
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("X-Session-Token") != "" || r.Header.Get("Traceparent") != "" {
			t.Error("browser credentials or tracing reached ops service")
		}
		f.lastPayload = nil
		if err := json.NewDecoder(r.Body).Decode(&f.lastPayload); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if f.opsStatus != 0 {
			w.WriteHeader(f.opsStatus)
			_, _ = w.Write([]byte(`{"error":"private-ops-diagnostic","bot_token":"secret-token"}`))
			return
		}
		identity, ok := f.lastPayload["identity"].(map[string]interface{})
		if !ok || identity["kratos_id"] == "" || identity["kratos_id"] == nil {
			t.Error("trusted identity missing")
			w.WriteHeader(403)
			return
		}
		if _, supplied := identity["email"]; supplied {
			t.Error("email reached the owner policy")
		}
		if identity["kratos_id"] != "owner-kratos-id" || (f.bound != "" && identity["kratos_id"] != f.bound) {
			w.WriteHeader(403)
			return
		}
		f.bound = "owner-kratos-id"
		if r.URL.Path == "/internal/hermes/access" {
			w.WriteHeader(204)
			return
		}
		if f.opsBody != "" {
			_, _ = w.Write([]byte(f.opsBody))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"claimed": f.bound != "", "telegram_linked": false, "telegram_configured": false, "gitlab_configured": false,
			"chatgpt_connected": false, "ready": false,
			"bot_token": "secret-token", "gitlab_token": "secret-token", "owner_identity": "private-identity",
		})
	}))
	t.Cleanup(ops.Close)
	tokenFile := filepath.Join(t.TempDir(), "ops-token")
	f.tokenFile = tokenFile
	if err := os.WriteFile(tokenFile, []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	// Runtime configuration is loaded through the same server environment as production.
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "absent.env"))
	t.Setenv("HERMES_OPS_URL", ops.URL)
	t.Setenv("HERMES_OPS_TOKEN_FILE", tokenFile)
	t.Setenv("HERMES_ALLOWED_ORIGIN", "https://map.webspace.cat")
	t.Setenv("HERMES_DASHBOARD_URL", "https://auth.webspace.cat/hermes/")
	f.handler = WithHermesRoutes(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(218) }), config.Load(), auth.NewKratosClient(kratos.URL), f.links, nil)
	return f
}

func (f *hermesFixture) request(method, path, body string, change func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Cookie", "ory_kratos_session=valid")
	r.Header.Set("Origin", "https://map.webspace.cat")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Studyverse-Hermes", "1")
	r.Header.Set("Authorization", "Bearer forged-owner-token")
	r.Header.Set("X-Session-Token", "forged-session-token")
	r.Header.Set("X-User-ID", "owner-kratos-id")
	// A stale privileged context must never survive a failed fresh cookie check.
	r = r.WithContext(sigNozContext(ownerSession(), "owner-kratos-id"))
	if change != nil {
		change(r)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func TestHermesOwnerRoutesRequireFreshSessionAndPinnedIdentity(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*hermesFixture)
		request func(*http.Request)
		want    int
	}{
		{"verified owner", nil, nil, 204},
		{"normalized owner email", func(f *hermesFixture) {
			f.session.Identity.Traits = map[string]interface{}{"email": " 250875@ASTANAIT.EDU.KZ "}
		}, nil, 204},
		{"foreign admin email", func(f *hermesFixture) {
			f.session.Identity.Id = "other-admin-kratos-id"
			f.session.Identity.Traits = map[string]interface{}{"email": "admin@astanait.edu.kz", "role": "admin"}
		}, nil, 403},
		{"inactive identity", func(f *hermesFixture) { f.session.Identity.SetState("inactive") }, nil, 401},
		{"missing identity state", func(f *hermesFixture) { f.session.Identity.State = nil }, nil, 401},
		{"expired cookie", func(f *hermesFixture) { f.session.SetExpiresAt(time.Now().Add(-time.Minute)) }, nil, 401},
		{"missing expiry", func(f *hermesFixture) { f.session.ExpiresAt = nil }, nil, 401},
		{"inactive session", func(f *hermesFixture) { f.session.SetActive(false) }, nil, 401},
		{"missing identity", func(f *hermesFixture) { f.session.Identity = nil }, nil, 401},
		{"empty identity", func(f *hermesFixture) { f.session.Identity.Id = "" }, nil, 401},
		{"bearer only", nil, func(r *http.Request) { r.Header.Del("Cookie") }, 401},
		{"invalid fresh cookie", func(f *hermesFixture) { f.kratosStatus = 401 }, nil, 401},
		{"Kratos unavailable", func(f *hermesFixture) { f.kratosStatus = 503 }, nil, 503},
		{"alternate identity same verified email", func(f *hermesFixture) { f.session.Identity.Id = "replacement-identity" }, nil, 403},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newHermesFixture(t)
			if tt.change != nil {
				tt.change(f)
			}
			w := f.request("GET", "/admin/hermes/access", "", tt.request)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
			wantOpsCalls := 0
			if tt.want == 204 || tt.want == 403 {
				wantOpsCalls = 1
			}
			if f.opsCalls != wantOpsCalls {
				t.Fatalf("ops calls = %d, want %d", f.opsCalls, wantOpsCalls)
			}
			if tt.name == "bearer only" && f.kratosCalls != 0 {
				t.Fatal("bearer authenticated through Kratos")
			}
			if strings.Contains(w.Body.String(), "private-") {
				t.Fatal("dependency diagnostic leaked")
			}
			assertSigNozPrivacy(t, w)
		})
	}
}

func TestHermesFirstAndSubsequentAccessIgnoreEmailProof(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*ory.Identity)
	}{
		{"missing email", func(identity *ory.Identity) { identity.Traits = map[string]interface{}{} }},
		{"changed email", func(identity *ory.Identity) {
			identity.Traits = map[string]interface{}{"email": "changed@astanait.edu.kz"}
			identity.VerifiableAddresses[0].Value = "changed@astanait.edu.kz"
		}},
		{"unverified email", func(identity *ory.Identity) { identity.VerifiableAddresses[0].Verified = false }},
		{"pending verification", func(identity *ory.Identity) { identity.VerifiableAddresses[0].Status = "pending" }},
		{"missing verification", func(identity *ory.Identity) { identity.VerifiableAddresses = nil }},
		{"wrong verified email", func(identity *ory.Identity) { identity.VerifiableAddresses[0].Value = "other@astanait.edu.kz" }},
		{"wrong verified method", func(identity *ory.Identity) { identity.VerifiableAddresses[0].Via = "sms" }},
		{"display email only", func(identity *ory.Identity) {
			identity.Traits = map[string]interface{}{"display_email": "owner@example.test"}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, route := range []struct {
				name, method, path, body string
				claimed                  bool
				status                   int
			}{
				{"pinned access", "GET", "/admin/hermes/access", "", true, 204},
				{"pinned info", "GET", "/admin/hermes/info", "", true, 200},
				{"pinned settings", "POST", "/admin/hermes/settings", `{"action":"test"}`, true, 200},
				{"initial claim", "POST", "/admin/hermes/settings", `{"action":"claim"}`, false, 200},
				{"unclaimed info", "GET", "/admin/hermes/info", "", false, 200},
			} {
				t.Run(route.name, func(t *testing.T) {
					f := newHermesFixture(t)
					tt.change(f.session.Identity)
					if !route.claimed {
						f.bound = ""
					}
					w := f.request(route.method, route.path, route.body, nil)
					if w.Code != route.status || f.opsCalls != 1 {
						t.Fatalf("status = %d, want %d; ops calls = %d", w.Code, route.status, f.opsCalls)
					}
					identity, ok := f.lastPayload["identity"].(map[string]interface{})
					if !ok || len(identity) != 1 || identity["kratos_id"] != "owner-kratos-id" {
						t.Fatalf("expected only live UUID, got %#v", identity)
					}
					if f.bound != "owner-kratos-id" {
						t.Fatal("expected the configured owner to be bound")
					}
				})
			}
		})
	}
}

func TestHermesFirstDirectAccessUsesConfiguredIdentity(t *testing.T) {
	f := newHermesFixture(t)
	f.bound = ""
	f.session.Identity.Id = "different-account"
	if got := f.request("GET", "/admin/hermes/access", "", nil); got.Code != 403 || f.bound != "" {
		t.Fatal("a different identity with the same verified address must not acquire ownership")
	}
	f.session.Identity.Id = "owner-kratos-id"
	f.session.Identity.VerifiableAddresses = nil
	if got := f.request("GET", "/admin/hermes/access", "", nil); got.Code != 204 || f.bound != "owner-kratos-id" {
		t.Fatal("the configured owner could not open the panel directly")
	}
	if f.links.calls != 0 {
		t.Fatal("web access consulted a Telegram or local-account link")
	}
}

func TestHermesInfoOnlyReturnsPublicFlags(t *testing.T) {
	f := newHermesFixture(t)
	w := f.request("GET", "/admin/hermes/info", "", nil)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"claimed", "telegram_linked", "telegram_configured", "gitlab_configured", "chatgpt_connected", "ready"} {
		if _, ok := body[field].(bool); !ok {
			t.Errorf("missing public flag %s", field)
		}
	}
	if body["dashboard_url"] != "https://auth.webspace.cat/hermes/" {
		t.Errorf("dashboard URL = %v", body["dashboard_url"])
	}
	for _, secret := range []string{"secret-token", "bot_token", "gitlab_token", "owner_identity", "private-identity"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("private field leaked: %s", secret)
		}
	}
	assertSigNozPrivacy(t, w)
}

func TestHermesSettingsRejectCSRFAndSpoofedClaims(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		change     func(*http.Request)
		status     int
	}{
		{"missing origin", `{"action":"claim"}`, func(r *http.Request) { r.Header.Del("Origin") }, 403},
		{"foreign origin", `{"action":"claim"}`, func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, 403},
		{"suffix origin", `{"action":"claim"}`, func(r *http.Request) { r.Header.Set("Origin", "https://map.webspace.cat.evil.example") }, 403},
		{"null origin", `{"action":"claim"}`, func(r *http.Request) { r.Header.Set("Origin", "null") }, 403},
		{"missing header", `{"action":"claim"}`, func(r *http.Request) { r.Header.Del("X-Studyverse-Hermes") }, 403},
		{"simple content type", `{"action":"claim"}`, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 415},
		{"client telegram id", `{"action":"bind_telegram","telegram_id":"123"}`, nil, 400},
		{"client trusted telegram", `{"action":"bind_telegram","telegram":{"id":"123","linked_at":"2026-01-01T00:00:00Z"}}`, nil, 400},
		{"client identity", `{"action":"claim","identity":{"kratos_id":"x","email":"250875@astanait.edu.kz"}}`, nil, 400},
		{"unknown action", `{"action":"admin_override"}`, nil, 400},
		{"secret wrong action", `{"action":"claim","bot_token":"secret-token"}`, nil, 400},
		{"missing token", `{"action":"configure_telegram"}`, nil, 400},
		{"trailing json", `{"action":"claim"}{"action":"test"}`, nil, 400},
		{"oversized body", `{"action":"configure_gitlab","gitlab_token":"` + strings.Repeat("x", 9000) + `"}`, nil, 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newHermesFixture(t)
			w := f.request("POST", "/admin/hermes/settings", tt.body, tt.change)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.status, w.Body.String())
			}
			if f.opsCalls != 0 || f.links.calls != 0 {
				t.Fatal("invalid mutation reached dependency")
			}
			assertSigNozPrivacy(t, w)
		})
	}
}

func TestHermesTelegramBindingUsesVerifiedServerLink(t *testing.T) {
	f := newHermesFixture(t)
	w := f.request("POST", "/admin/hermes/settings", `{"action":"bind_telegram"}`, nil)
	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	if f.links.calls != 1 || f.links.identity != "owner-kratos-id" {
		t.Fatalf("lookup identity = %q, calls = %d", f.links.identity, f.links.calls)
	}
	if f.opsCalls != 2 || f.lastPath != "/internal/hermes/settings" {
		t.Fatalf("binding must check owner access before mutation: calls = %d, path = %s", f.opsCalls, f.lastPath)
	}
	link, ok := f.lastPayload["telegram"].(map[string]interface{})
	if !ok || link["id"] != "987654321" || link["linked_at"] != "2026-10-20T12:00:00Z" {
		t.Fatalf("trusted link = %#v", link)
	}
	for _, tt := range []struct {
		name   string
		link   hermes.TelegramLink
		err    error
		status int
	}{
		{"missing link", hermes.TelegramLink{}, nil, 403},
		{"unverified link", hermes.TelegramLink{ID: "987654321"}, nil, 403},
		{"negative id", hermes.TelegramLink{ID: "-100987", LinkedAt: time.Now()}, nil, 403},
		{"lookup unavailable", hermes.TelegramLink{}, hermes.ErrUnavailable, 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newHermesFixture(t)
			f.links.link = tt.link
			f.links.err = tt.err
			w := f.request("POST", "/admin/hermes/settings", `{"action":"bind_telegram"}`, nil)
			if w.Code != tt.status || f.opsCalls != 1 || f.lastPath != "/internal/hermes/access" {
				t.Fatalf("status=%d ops calls=%d", w.Code, f.opsCalls)
			}
		})
	}
}

func TestHermesTelegramBindingAuthorizesBeforeLinkLookup(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*hermesFixture)
		status int
	}{
		{"foreign administrator", func(f *hermesFixture) {
			f.session.Identity.Id = "other-admin-kratos-id"
			f.session.Identity.Traits = map[string]interface{}{"email": "admin@astanait.edu.kz", "role": "admin"}
		}, 403},
		{"alternate verified identity", func(f *hermesFixture) { f.session.Identity.Id = "replacement-identity" }, 403},
		{"conflicting binding", func(f *hermesFixture) { f.bound = "different-pinned-identity" }, 403},
		{"ops unavailable", func(f *hermesFixture) { f.opsStatus = 503 }, 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newHermesFixture(t)
			tt.change(f)
			w := f.request("POST", "/admin/hermes/settings", `{"action":"bind_telegram"}`, nil)
			if w.Code != tt.status || f.opsCalls != 1 || f.links.calls != 0 || f.lastPath != "/internal/hermes/access" {
				t.Fatalf("status = %d, ops calls = %d, link lookups = %d, path = %s", w.Code, f.opsCalls, f.links.calls, f.lastPath)
			}
		})
	}
}

func TestHermesOpsFailuresAreGenericAndInfoNeedsHeader(t *testing.T) {
	for _, status := range []int{401, 403, 500, 503} {
		f := newHermesFixture(t)
		f.opsStatus = status
		w := f.request("GET", "/admin/hermes/info", "", nil)
		want := 503
		if status == 403 {
			want = 403
		}
		if w.Code != want || strings.Contains(w.Body.String(), "private-") || strings.Contains(w.Body.String(), "secret-token") {
			t.Fatalf("runtime %d => %d %s", status, w.Code, w.Body.String())
		}
	}
	f := newHermesFixture(t)
	w := f.request("GET", "/admin/hermes/info", "", func(r *http.Request) { r.Header.Del("X-Studyverse-Hermes") })
	if w.Code != 403 || f.opsCalls != 0 {
		t.Fatalf("missing info header = %d", w.Code)
	}
	f.opsBody = `{"claimed":`
	if w = f.request("GET", "/admin/hermes/info", "", nil); w.Code != 503 {
		t.Fatalf("malformed runtime response = %d", w.Code)
	}
}

func TestHermesRoutesOnlySelectExactEndpoints(t *testing.T) {
	f := newHermesFixture(t)
	for _, path := range []string{"/health", "/admin/hermes/info/", "/admin/hermes/settings/", "/admin/hermes/access/", "/admin/promote"} {
		if w := f.request("GET", path, "", nil); w.Code != 218 {
			t.Fatalf("unrelated path %s intercepted: %d", path, w.Code)
		}
	}
	for _, tt := range []struct{ method, path, allow string }{
		{"POST", "/admin/hermes/info", "GET"}, {"POST", "/admin/hermes/access", "GET"}, {"GET", "/admin/hermes/settings", "POST"},
	} {
		w := f.request(tt.method, tt.path, "", nil)
		if w.Code != 405 || w.Header().Get("Allow") != tt.allow {
			t.Fatalf("method %s %s = %d", tt.method, tt.path, w.Code)
		}
	}
}

func TestHermesRuntimeRejectsIncompletePublicStatus(t *testing.T) {
	for _, body := range []string{"null", "{}", `{"claimed":true}`, strings.Repeat(" ", 65537)} {
		f := newHermesFixture(t)
		f.opsBody = body
		w := f.request("GET", "/admin/hermes/info", "", nil)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("incomplete runtime response %q => %d", body[:min(len(body), 30)], w.Code)
		}
	}
}

func TestHermesMissingOrInvalidSecretFileFailsClosed(t *testing.T) {
	for _, value := range []string{"", "short", strings.Repeat("x", 4097), "valid-looking-but-multiline-token-value\nsecond"} {
		f := newHermesFixture(t)
		if err := os.WriteFile(f.tokenFile, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		w := f.request("GET", "/admin/hermes/info", "", nil)
		if w.Code != 503 || f.opsCalls != 0 {
			t.Fatalf("unsafe token accepted: %d", w.Code)
		}
	}
	f := newHermesFixture(t)
	if err := os.Remove(f.tokenFile); err != nil {
		t.Fatal(err)
	}
	if w := f.request("GET", "/admin/hermes/access", "", nil); w.Code != 503 || f.opsCalls != 0 {
		t.Fatalf("missing secret accepted: %d", w.Code)
	}
}

func TestHermesDashboardSetupOriginsAndOAuthActions(t *testing.T) {
	for _, body := range []string{`{"action":"connect_chatgpt"}`, `{"action":"poll_chatgpt","flow_id":"flow-123"}`} {
		f := newHermesFixture(t)
		f.opsBody = `{"claimed":true,"telegram_linked":false,"telegram_configured":false,"gitlab_configured":false,"chatgpt_connected":false,"ready":false,"oauth":{"flow_id":"flow-123","verification_url":"https://auth.openai.com/codex/device","user_code":"ABC-123","status":"pending","access_token":"secret-token"}}`
		w := f.request("POST", "/admin/hermes/settings", body, func(r *http.Request) { r.Header.Set("Origin", "https://auth.webspace.cat") })
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"flow_id":"flow-123"`) || strings.Contains(w.Body.String(), "secret-token") {
			t.Fatalf("OAuth setup failed: %d %s", w.Code, w.Body.String())
		}
		if !strings.Contains(body, `"action":"`+f.lastPayload["action"].(string)+`"`) {
			t.Fatal("OAuth action changed in transit")
		}
	}
	for _, body := range []string{`{"action":"poll_chatgpt"}`, `{"action":"connect_chatgpt","flow_id":"unexpected"}`, `{"action":"test","gitlab_token":"secret-token"}`} {
		f := newHermesFixture(t)
		if w := f.request("POST", "/admin/hermes/settings", body, nil); w.Code != 400 || f.opsCalls != 0 {
			t.Fatalf("invalid action shape = %d", w.Code)
		}
	}
}

func TestHermesIncidentResultIsPublicButRawProviderDetailsAreNot(t *testing.T) {
	f := newHermesFixture(t)
	f.opsBody = `{"claimed":true,"telegram_linked":false,"telegram_configured":false,"gitlab_configured":false,"chatgpt_connected":false,"ready":false,"recent_incidents":[{"id":1,"source":"signoz","environment":"production","service":"backend","severity":2,"status":"complete","summary":"Database unavailable","result":"Check the database connection and retry.","created_at":1793655000,"raw_config":"private-configuration","logs":"private-log-blob"}]}`
	w := f.request("GET", "/admin/hermes/info", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"result":"Check the database connection and retry."`) {
		t.Fatalf("diagnosis result missing: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "private-") || strings.Contains(w.Body.String(), "raw_config") {
		t.Fatal("raw incident details leaked")
	}
}
