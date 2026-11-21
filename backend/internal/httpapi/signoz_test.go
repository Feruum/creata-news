package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	ory "github.com/ory/kratos-client-go"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
)

type sigNozAdmin struct {
	err   error
	calls int
}

func (a *sigNozAdmin) RequirePlatformAdmin(context.Context) (int32, error) {
	a.calls++
	return 42, a.err
}

func sigNozSession() *ory.Session {
	session := ory.NewSession("browser-session")
	session.SetActive(true)
	session.SetExpiresAt(time.Now().Add(time.Hour))
	session.Identity = ory.NewIdentity("administrator-id", "default", "https://identity.test/schema", map[string]interface{}{})
	return session
}

func sigNozContext(session *ory.Session, userID string) context.Context {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	return context.WithValue(ctx, auth.SessionKey, session)
}

func assertSigNozPrivacy(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	for name, expected := range map[string]string{
		"Cache-Control": "no-store", "Pragma": "no-cache", "Referrer-Policy": "no-referrer",
	} {
		if got := response.Header().Get(name); got != expected {
			t.Errorf("%s = %q, want %q", name, got, expected)
		}
	}
}

func TestSigNozAccessRequiresCurrentSession(t *testing.T) {
	for _, test := range []struct {
		name   string
		userID string
		modify func(*ory.Session) *ory.Session
	}{
		{"anonymous", "", func(*ory.Session) *ory.Session { return nil }},
		{"user without session", "administrator-id", func(*ory.Session) *ory.Session { return nil }},
		{"inactive", "administrator-id", func(s *ory.Session) *ory.Session { s.SetActive(false); return s }},
		{"missing active", "administrator-id", func(s *ory.Session) *ory.Session { s.Active = nil; return s }},
		{"expired", "administrator-id", func(s *ory.Session) *ory.Session { s.SetExpiresAt(time.Now().Add(-time.Second)); return s }},
		{"missing expiry", "administrator-id", func(s *ory.Session) *ory.Session { s.ExpiresAt = nil; return s }},
		{"missing identity", "administrator-id", func(s *ory.Session) *ory.Session { s.Identity = nil; return s }},
		{"mismatched identity", "someone-else", func(s *ory.Session) *ory.Session { return s }},
		{"empty identity", "", func(s *ory.Session) *ory.Session { s.Identity.Id = ""; return s }},
	} {
		t.Run(test.name, func(t *testing.T) {
			admin := &sigNozAdmin{}
			request := httptest.NewRequest(http.MethodGet, "/admin/signoz/access", nil)
			request = request.WithContext(sigNozContext(test.modify(sigNozSession()), test.userID))
			response := httptest.NewRecorder()
			NewSigNozHandler(&config.Config{}, admin).ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", response.Code)
			}
			if admin.calls != 0 {
				t.Fatal("authorization ran before session verification")
			}
			assertSigNozPrivacy(t, response)
		})
	}
}

func TestSigNozAccessAdministratorReturnsNoContent(t *testing.T) {
	admin := &sigNozAdmin{}
	request := httptest.NewRequest(http.MethodGet, "/admin/signoz/access", nil)
	request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
	response := httptest.NewRecorder()
	NewSigNozHandler(&config.Config{SigNozUIURL: "https://signoz.example.test"}, admin).ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("response = %d %q, want empty 204", response.Code, response.Body.String())
	}
	if admin.calls != 1 {
		t.Fatalf("administrator check calls = %d, want 1", admin.calls)
	}
	assertSigNozPrivacy(t, response)
}

func TestSigNozAccessDisabledConfigurationFailsClosed(t *testing.T) {
	admin := &sigNozAdmin{}
	request := httptest.NewRequest(http.MethodGet, "/admin/signoz/access", nil)
	request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
	response := httptest.NewRecorder()
	NewSigNozHandler(&config.Config{
		SigNozUIURL: "", SigNozLoginEmail: "operator@example.test", SigNozLoginPassword: "private-password",
	}, admin).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || strings.TrimSpace(response.Body.String()) != `{"error":"Service Unavailable"}` {
		t.Fatalf("disabled access response = %d %q, want generic 503", response.Code, response.Body.String())
	}
	if admin.calls != 1 {
		t.Fatalf("administrator check calls = %d, want 1 before disabled configuration check", admin.calls)
	}
	assertSigNozPrivacy(t, response)
}

func TestSigNozAuthorizationErrorsStayGeneric(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"missing local identity", connect.NewError(connect.CodeUnauthenticated, errors.New("secret-diagnostic")), 401},
		{"staff", connect.NewError(connect.CodePermissionDenied, errors.New("secret-diagnostic")), 403},
		{"unavailable", connect.NewError(connect.CodeUnavailable, errors.New("secret-diagnostic")), 503},
		{"internal", connect.NewError(connect.CodeInternal, errors.New("secret-diagnostic")), 503},
		{"unknown", errors.New("secret-diagnostic"), 503},
	} {
		for _, path := range []string{"/admin/signoz/access", "/admin/signoz/info"} {
			t.Run(test.name+path, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, path, nil)
				request.Header.Set("X-Studyverse-SigNoz", "1")
				request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
				response := httptest.NewRecorder()
				NewSigNozHandler(&config.Config{}, &sigNozAdmin{err: test.err}).ServeHTTP(response, request)
				if response.Code != test.status {
					t.Fatalf("status = %d, want %d", response.Code, test.status)
				}
				if strings.Contains(response.Body.String(), "secret-diagnostic") {
					t.Fatal("authorization error leaked to caller")
				}
				assertSigNozPrivacy(t, response)
			})
		}
	}
}

func TestSigNozInfoConfigurationAndCredentials(t *testing.T) {
	for _, test := range []struct {
		name string
		cfg  config.Config
		want string
	}{
		{"disabled", config.Config{}, `{"configured":false}`},
		{"disabled credentials hidden", config.Config{SigNozLoginEmail: "operator@example.test", SigNozLoginPassword: "private-password"}, `{"configured":false}`},
		{"enabled", config.Config{SigNozUIURL: "https://signoz.example.test/"}, `{"configured":true,"url":"https://signoz.example.test"}`},
		{"IPv6 root origin", config.Config{SigNozUIURL: "https://[2001:db8::1]:3301/"}, `{"configured":true,"url":"https://[2001:db8::1]:3301"}`},
		{"email only", config.Config{SigNozUIURL: "https://signoz.example.test", SigNozLoginEmail: "operator@example.test"}, `{"configured":true,"url":"https://signoz.example.test"}`},
		{"password only", config.Config{SigNozUIURL: "https://signoz.example.test", SigNozLoginPassword: "private-password"}, `{"configured":true,"url":"https://signoz.example.test"}`},
		{"complete login", config.Config{SigNozUIURL: "https://signoz.example.test", SigNozLoginEmail: "operator@example.test", SigNozLoginPassword: "private-password"}, `{"configured":true,"url":"https://signoz.example.test","login":{"email":"operator@example.test","password":"private-password"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/admin/signoz/info", nil)
			request.Header.Set("X-Studyverse-SigNoz", "1")
			request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
			response := httptest.NewRecorder()
			NewSigNozHandler(&test.cfg, &sigNozAdmin{}).ServeHTTP(response, request)
			if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != test.want {
				t.Fatalf("response = %d %s, want 200 %s", response.Code, response.Body.String(), test.want)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("unexpected content type: %q", response.Header().Get("Content-Type"))
			}
			assertSigNozPrivacy(t, response)
		})
	}
}

func TestSigNozInfoRequiresExplicitHeader(t *testing.T) {
	for _, header := range []string{"", "true", "0"} {
		request := httptest.NewRequest(http.MethodGet, "/admin/signoz/info", nil)
		request.Header.Set("X-Studyverse-SigNoz", header)
		request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
		response := httptest.NewRecorder()
		NewSigNozHandler(&config.Config{SigNozUIURL: "https://signoz.example.test", SigNozLoginEmail: "operator@example.test", SigNozLoginPassword: "private-password"}, &sigNozAdmin{}).ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), "private-password") {
			t.Fatalf("response for header %q = %d %q", header, response.Code, response.Body.String())
		}
		assertSigNozPrivacy(t, response)
	}
}

func TestSigNozInvalidOriginFailsClosed(t *testing.T) {
	for _, origin := range []string{
		"http://signoz.example.test", "https://user:private-password@signoz.example.test", "https://signoz.example.test/path",
		"https://signoz.example.test/?query=private-password", "https://signoz.example.test/?", "https://signoz.example.test/#fragment",
		"https://signoz.example.test/#", "https://signoz.example.test//", "https://signoz.example.test/%2f", "https:///",
		"https://signoz.example.test:invalid", "https://signoz.example.test:65536", "https://signoz.example.test\\path",
		"https://example.test%25", "https://[fe80::1%25eth0]", "https://[fe80::1%25eth%30]",
	} {
		for _, path := range []string{"/admin/signoz/info", "/admin/signoz/access"} {
			t.Run(origin+path, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, path, nil)
				request.Header.Set("X-Studyverse-SigNoz", "1")
				request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
				response := httptest.NewRecorder()
				NewSigNozHandler(&config.Config{SigNozUIURL: origin}, &sigNozAdmin{}).ServeHTTP(response, request)
				if response.Code != http.StatusServiceUnavailable {
					t.Errorf("origin %q at %s = %d, want 503", origin, path, response.Code)
				}
				if strings.Contains(response.Body.String(), "private-password") || strings.Contains(response.Body.String(), origin) {
					t.Fatal("invalid configuration leaked to caller")
				}
				assertSigNozPrivacy(t, response)
			})
		}
	}
}

func TestSigNozRoutesFreshCookieValidation(t *testing.T) {
	for _, test := range []struct {
		name         string
		cookie       string
		kratosStatus int
		malformed    bool
		wantStatus   int
		wantCalls    int
	}{
		{"cookie with SigNoz tokens", "ory_kratos_session=valid", 200, false, 204, 1},
		{"bearer and session token cannot authenticate", "", 200, false, 401, 0},
		{"invalid cookie clears prior context", "ory_kratos_session=invalid", 401, false, 401, 1},
		{"forbidden cookie", "ory_kratos_session=invalid", 403, false, 401, 1},
		{"Kratos unavailable", "ory_kratos_session=valid", 503, false, 503, 1},
		{"malformed Kratos response", "ory_kratos_session=valid", 200, true, 503, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/sessions/whoami" || r.Header.Get("Cookie") != test.cookie {
					t.Error("cookie was not freshly validated by Kratos")
				}
				if r.Header.Get("Authorization") != "" || r.Header.Get("X-Session-Token") != "" {
					t.Error("SigNoz token forwarded to Kratos")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.kratosStatus)
				if test.kratosStatus != 200 {
					_, _ = w.Write([]byte(`{"error":{"message":"private-diagnostic"}}`))
				} else if test.malformed {
					_, _ = w.Write([]byte(`{"private-diagnostic":`))
				} else {
					_ = json.NewEncoder(w).Encode(sigNozSession())
				}
			}))
			defer identity.Close()
			client := auth.NewKratosClient(identity.URL)
			fallback := auth.NewMiddleware(client)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Error("SigNoz endpoint used global auth router")
				w.WriteHeader(500)
			}))
			admin := &sigNozAdmin{}
			handler := WithSigNozRoutes(fallback, &config.Config{SigNozUIURL: "https://signoz.example.test"}, admin, client)
			request := httptest.NewRequest(http.MethodGet, "/admin/signoz/access", nil)
			request.Header.Set("Cookie", test.cookie)
			request.Header.Set("Authorization", "Bearer arbitrary-signoz-token")
			request.Header.Set("X-Session-Token", "arbitrary-signoz-session-token")
			request.Header.Set("X-User-ID", "administrator-id")
			request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || int(calls.Load()) != test.wantCalls {
				t.Fatalf("status=%d Kratos calls=%d, want %d/%d", response.Code, calls.Load(), test.wantStatus, test.wantCalls)
			}
			if test.wantStatus != 204 && admin.calls != 0 {
				t.Fatal("failed fresh authentication reached authorizer")
			}
			if strings.Contains(response.Body.String(), "private-diagnostic") {
				t.Fatal("Kratos diagnostic leaked")
			}
			assertSigNozPrivacy(t, response)
		})
	}
}

func TestSigNozRoutesOnlySelectExactEndpoints(t *testing.T) {
	fallbackCalls := 0
	handler := WithSigNozRoutes(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fallbackCalls++
		w.WriteHeader(218)
	}), &config.Config{}, &sigNozAdmin{}, nil)
	for _, path := range []string{"/health", "/admin/signoz/info/", "/admin/signoz/access/", "/admin/promote"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 218 {
			t.Fatalf("unrelated path %s intercepted", path)
		}
	}
	if fallbackCalls != 4 {
		t.Fatalf("fallback calls = %d, want 4", fallbackCalls)
	}
}

func TestSigNozCookieDependencyTimeout(t *testing.T) {
	identity := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer identity.Close()
	client := auth.NewKratosClient(identity.URL)
	client.GetConfig().HTTPClient.Timeout = 25 * time.Millisecond
	admin := &sigNozAdmin{}
	handler := WithSigNozRoutes(http.NotFoundHandler(), &config.Config{}, admin, client)
	request := httptest.NewRequest(http.MethodGet, "/admin/signoz/info", nil)
	request.Header.Set("Cookie", "ory_kratos_session=valid")
	request.Header.Set("X-Studyverse-SigNoz", "1")
	request = request.WithContext(sigNozContext(sigNozSession(), "administrator-id"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || admin.calls != 0 {
		t.Fatalf("timeout response = %d, authorizer calls = %d", response.Code, admin.calls)
	}
	assertSigNozPrivacy(t, response)
}

func TestSigNozInfoAndMethodsRequireAuthorization(t *testing.T) {
	for _, path := range []string{"/admin/signoz/info", "/admin/signoz/access"} {
		response := httptest.NewRecorder()
		NewSigNozHandler(&config.Config{}, &sigNozAdmin{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != 401 {
			t.Fatalf("anonymous %s status = %d", path, response.Code)
		}
		assertSigNozPrivacy(t, response)
		response = httptest.NewRecorder()
		NewSigNozHandler(&config.Config{}, &sigNozAdmin{}).ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != 405 || response.Header().Get("Allow") != "GET" {
			t.Fatalf("POST %s status = %d", path, response.Code)
		}
		assertSigNozPrivacy(t, response)
	}
}
