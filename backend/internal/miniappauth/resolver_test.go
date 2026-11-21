package miniappauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolverUsesConfirmedLinkAndImportsOnlyAdditionalProvider(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	var imported map[string]json.RawMessage
	existingSubject := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/identities/"+id {
			t.Errorf("invalid identity request: %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if r.Method == "PUT" {
			if err := json.NewDecoder(r.Body).Decode(&imported); err != nil {
				t.Error(err)
			}
			var credentials struct {
				OIDC struct {
					Config struct {
						Providers []struct {
							Provider string
							Subject  string
						}
					}
				}
			}
			if err := json.Unmarshal(imported["credentials"], &credentials); err != nil {
				t.Error(err)
			}
			if len(credentials.OIDC.Config.Providers) != 1 || credentials.OIDC.Config.Providers[0].Provider != ProviderID {
				t.Error("other credentials would be re-imported")
			}
			existingSubject = credentials.OIDC.Config.Providers[0].Subject
			w.WriteHeader(200)
			w.Write([]byte(`{}`))
			return
		}
		// Other credential metadata must never be imported, exposed or deleted.
		providers := []map[string]string{{"provider": "microsoft", "subject": "original-microsoft-subject", "initial_access_token": "synthetic-private-microsoft-token"}}
		if existingSubject != "" {
			providers = append(providers, map[string]string{"provider": ProviderID, "subject": existingSubject})
		}
		json.NewEncoder(w).Encode(map[string]any{"id": id, "schema_id": "default", "state": "active", "traits": map[string]any{"email": "synthetic@test.local", "name": map[string]string{"first": "Original"}}, "metadata_admin": map[string]string{"keep": "original-admin-metadata"}, "metadata_public": map[string]string{"keep": "original-public-metadata"}, "credentials": map[string]any{"oidc": map[string]any{"config": map[string]any{"providers": providers}}, "password": map[string]any{"type": "password"}, "totp": map[string]any{"type": "totp"}}})
	}))
	defer server.Close()
	resolver := NewResolver(nil, server.URL, server.URL, strings.Repeat("synthetic-", 4))
	record := linkRecord{IdentityID: id, TelegramID: 101, LinkedAt: time.Unix(100, 0)}
	resolver.lookup = func(context.Context, int64) (*linkRecord, error) { copy := record; return &copy, nil }
	link, err := resolver.Resolve(context.Background(), 101)
	if err != nil {
		t.Fatal(err)
	}
	if link.IdentityID != id || link.TelegramID != 101 || len(link.Subject) != 64 {
		t.Fatal("invalid link")
	}
	if err := resolver.EnsureCredential(context.Background(), *link); err != nil {
		t.Fatal(err)
	}
	if imported == nil {
		t.Fatal("provider not imported")
	}
	for _, key := range []string{"state", "schema_id", "password", "totp"} {
		if _, ok := imported[key]; ok {
			t.Fatalf("unrelated identity field overwritten: %s", key)
		}
	}
	if strings.Contains(string(imported["credentials"]), "synthetic-private-microsoft-token") || strings.Contains(string(imported["credentials"]), "microsoft") {
		t.Fatal("existing Microsoft credential rewritten")
	}
	if !strings.Contains(string(imported["traits"]), "Original") || !strings.Contains(string(imported["metadata_admin"]), "original-admin-metadata") {
		t.Fatal("existing traits or metadata lost")
	}
	imported = nil
	if err := resolver.EnsureCredential(context.Background(), *link); err != nil {
		t.Fatal(err)
	}
	if imported != nil {
		t.Fatal("matching credential imported again")
	}
	record.LinkedAt = record.LinkedAt.Add(time.Second)
	if resolver.Check(context.Background(), *link) == nil {
		t.Fatal("re-linked Telegram accepted old credential")
	}
}

func TestResolverRejectsInactiveOrMissingKratosIdentity(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	for _, status := range []string{"inactive", "missing"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == "missing" {
					w.WriteHeader(404)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"id": id, "state": "inactive"})
			}))
			defer server.Close()
			resolver := NewResolver(nil, server.URL, server.URL, strings.Repeat("synthetic-", 4))
			resolver.lookup = func(context.Context, int64) (*linkRecord, error) {
				return &linkRecord{IdentityID: id, TelegramID: 101, LinkedAt: time.Unix(100, 0)}, nil
			}
			if _, err := resolver.Resolve(context.Background(), 101); err == nil {
				t.Fatal("inactive identity can sign in")
			}
		})
	}
}

func TestResolverSessionValidationDoesNotFollowRedirectsOrAcceptInactiveSessions(t *testing.T) {
	var cookieForwarded bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { cookieForwarded = true; w.WriteHeader(200) }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") == "ory_kratos_session=mfa-required" {
			w.WriteHeader(403)
			return
		}
		if r.Header.Get("Cookie") == "ory_kratos_session=redirect" {
			http.Redirect(w, r, destination.URL, 302)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"active": false, "identity": map[string]string{"id": "old-user"}})
	}))
	defer server.Close()
	resolver := NewResolver(nil, server.URL, server.URL, strings.Repeat("synthetic-", 4))
	if _, err := resolver.CurrentIdentity(context.Background(), "ory_kratos_session=redirect"); err == nil || cookieForwarded {
		t.Fatal("session credentials followed a redirect")
	}
	if id, err := resolver.CurrentIdentity(context.Background(), "ory_kratos_session=expired"); err != nil || id != "" {
		t.Fatal("inactive session accepted")
	}
	if _, err := resolver.CurrentIdentity(context.Background(), "ory_kratos_session=mfa-required"); err == nil {
		t.Fatal("additional verification was treated as an anonymous browser")
	}
}

func TestResolverUsesOnlyConfiguredSessionCookie(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		cookie, err := r.Cookie("ory_kratos_session_dev")
		if err != nil || cookie.Value != "synthetic-dev-session" {
			t.Error("configured session cookie was not forwarded")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"active": true, "identity": map[string]string{"id": id}})
	}))
	defer server.Close()
	resolver := NewResolver(nil, server.URL, server.URL, "synthetic-secret", WithSessionCookieName("ory_kratos_session_dev"))
	if actual, err := resolver.CurrentIdentity(context.Background(), "ory_kratos_session=synthetic-prod-session"); err != nil || actual != "" || calls.Load() != 0 {
		t.Fatal("a production cookie alone must be ignored by a development resolver")
	}
	for _, cookie := range []string{
		"ory_kratos_session_dev=synthetic-dev-session",
		"ory_kratos_session=synthetic-prod-session; ory_kratos_session_dev=synthetic-dev-session",
	} {
		actual, err := resolver.CurrentIdentity(context.Background(), cookie)
		if err != nil || actual != id {
			t.Fatalf("configured development cookie did not resolve the identity: %q, %v", actual, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("only configured session cookies should reach Kratos")
	}
}

func TestResolverKeepsDefaultSessionCookieWhenOptionIsEmpty(t *testing.T) {
	id := "00000000-0000-4000-8000-000000000001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"active": true, "identity": map[string]string{"id": id}})
	}))
	defer server.Close()
	resolver := NewResolver(nil, server.URL, server.URL, "synthetic-secret", WithSessionCookieName(""))
	actual, err := resolver.CurrentIdentity(context.Background(), "ory_kratos_session=synthetic-prod-session")
	if err != nil || actual != id {
		t.Fatalf("default production cookie changed: %q, %v", actual, err)
	}
}
