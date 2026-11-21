package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestPrivateStorageEndpointIsOptionalAndIndependent(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	t.Setenv("STORAGE_ENDPOINT", "http://public-storage:9000")
	t.Setenv("STORAGE_PRIVATE_ENDPOINT", "")
	if cfg := Load(); cfg.StoragePrivateEndpoint != "" || cfg.StorageEndpoint != "http://public-storage:9000" {
		t.Fatal("private endpoint default changed the public provider")
	}
	t.Setenv("STORAGE_PRIVATE_ENDPOINT", "http://registration-storage:9000")
	if cfg := Load(); cfg.StoragePrivateEndpoint != "http://registration-storage:9000" || cfg.StorageEndpoint != "http://public-storage:9000" {
		t.Fatal("private endpoint override was not isolated")
	}
}

func TestHermesConfigDefaultsAndRuntimeOverrides(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "absent.env"))
	for _, tt := range []struct{ key, field, fallback, override string }{
		{"HERMES_OPS_URL", "HermesOpsURL", "", "http://ops-diagnostics:8090"},
		{"HERMES_OPS_TOKEN_FILE", "HermesOpsTokenFile", "", "/run/secrets/hermes-ops-token"},
		{"HERMES_DASHBOARD_URL", "HermesDashboardURL", "https://auth.webspace.cat/hermes/", "http://localhost:8080/hermes/"},
		{"HERMES_ALLOWED_ORIGIN", "HermesAllowedOrigin", "https://map.webspace.cat", "http://localhost:3000"},
	} {
		t.Run(tt.key, func(t *testing.T) {
			t.Setenv(tt.key, "")
			if err := os.Unsetenv(tt.key); err != nil {
				t.Fatal(err)
			}
			value := reflect.ValueOf(Load()).Elem().FieldByName(tt.field)
			if !value.IsValid() || value.String() != tt.fallback {
				t.Fatalf("missing default for %s", tt.key)
			}
			t.Setenv(tt.key, tt.override)
			if value = reflect.ValueOf(Load()).Elem().FieldByName(tt.field); !value.IsValid() || value.String() != tt.override {
				t.Fatalf("ignored override for %s", tt.key)
			}
		})
	}
}

func TestHermesRoutingURLsArePinnedOrExplicitLoopback(t *testing.T) {
	for _, value := range []string{
		"https://evil.example/hermes/", "http://auth.webspace.cat/hermes/", "https://auth.webspace.cat/",
		"https://auth.webspace.cat/hermes/?secret=1", "https://auth.webspace.cat/hermes/#", "http://localhost.evil.example/hermes/",
		"http://localhost/hermes/../", "http://localhost:65536/hermes/", "http://user:private-password@localhost/hermes/",
		"http://[fe80::1%25eth0]/hermes/", "http://localhost/%68ermes/", "http://localhost\\@evil.example/hermes/",
	} {
		if _, _, err := (&Config{HermesDashboardURL: value}).HermesURLs(); err == nil || strings.Contains(err.Error(), "private-password") {
			t.Errorf("unsafe dashboard accepted or error leaked for %q", value)
		}
	}
	for _, value := range []string{"*", "null", "https://evil.example", "https://map.webspace.cat/", "http://localhost/path"} {
		if _, _, err := (&Config{HermesAllowedOrigin: value}).HermesURLs(); err == nil {
			t.Errorf("unsafe browser origin accepted: %q", value)
		}
	}
	for _, value := range []string{"http://localhost:8080/hermes/", "http://127.0.0.1:8080/hermes/", "http://[::1]:8080/hermes/", "https://auth.webspace.cat/hermes/"} {
		if dashboard, _, err := (&Config{HermesDashboardURL: value}).HermesURLs(); err != nil || dashboard != value {
			t.Errorf("valid dashboard rejected: %q", value)
		}
	}
	for _, value := range []string{"ftp://ops", "http://user:password@ops", "http://ops/path", "http://ops/?", "http://ops/#", "http://ops:65536"} {
		if _, err := (&Config{HermesOpsURL: value}).HermesOpsOrigin(); err == nil {
			t.Errorf("unsafe ops URL accepted: %q", value)
		}
	}
	if origin, err := (&Config{HermesOpsURL: "http://ops-diagnostics:8090/"}).HermesOpsOrigin(); err != nil || origin != "http://ops-diagnostics:8090" {
		t.Fatal("valid internal service URL rejected")
	}
}

func TestSundayOpenProductionAliasAndDevelopmentIsolation(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	for _, tt := range []struct {
		name, configured string
		wantAlias        bool
	}{
		{"existing production", "https://sunday-open.vercel.app", true},
		{"existing production with local QA", "https://sunday-open.vercel.app,http://localhost:3003", true},
		{"already configured", "https://sundayopen.social,https://sunday-open.vercel.app", true},
		{"development", "http://localhost:3003", false},
		{"separate deployment", "https://private.example.test", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SUNDAYOPEN_ORIGINS", tt.configured)
			actual := Load().SundayOpenOrigins
			count := 0
			for _, origin := range strings.Split(actual, ",") {
				if origin == "https://sundayopen.social" {
					count++
				}
			}
			if (count == 1) != tt.wantAlias || count > 1 {
				t.Fatalf("production domain alias mismatch: %q", actual)
			}
			if !tt.wantAlias && actual != tt.configured {
				t.Fatalf("separate origin list changed: %q", actual)
			}
		})
	}
}

func TestSundayOpenLocalCORSAndExplicitProductionOrigins(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	t.Setenv("CORS_ORIGINS", "")
	if err := os.Unsetenv("CORS_ORIGINS"); err != nil {
		t.Fatal(err)
	}
	if origins := Load().CORSOrigins; !slices.Contains(origins, "http://localhost:3002") || !slices.Contains(origins, "http://localhost:5173") {
		t.Fatalf("local Sunday Open/admin origins missing: %v", origins)
	}
	t.Setenv("CORS_ORIGINS", "https://sundayopen.com,https://admin.example.test")
	if origins := Load().CORSOrigins; !slices.Equal(origins, []string{"https://sundayopen.com", "https://admin.example.test"}) {
		t.Fatalf("explicit production origins were not respected: %v", origins)
	}
}

func TestKratosSessionCookieNameUsesEnvironmentAndProductionDefault(t *testing.T) {
	// Do not load developer or production settings into this configuration test.
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	t.Setenv("KRATOS_SESSION_COOKIE_NAME", "ory_kratos_session_dev")
	if actual := Load().KratosSessionCookieName; actual != "ory_kratos_session_dev" {
		t.Fatalf("cookie namespace override was ignored: %q", actual)
	}
	if err := os.Unsetenv("KRATOS_SESSION_COOKIE_NAME"); err != nil {
		t.Fatal(err)
	}
	if actual := Load().KratosSessionCookieName; actual != "ory_kratos_session" {
		t.Fatalf("default production cookie changed: %q", actual)
	}
}

func TestSigNozConfigUsesOptionalRuntimeEnvironment(t *testing.T) {
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	for _, key := range []string{"SIGNOZ_UI_URL", "SIGNOZ_LOGIN_EMAIL", "SIGNOZ_LOGIN_PASSWORD"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Load()
	if cfg.SigNozUIURL != "" || cfg.SigNozLoginEmail != "" || cfg.SigNozLoginPassword != "" {
		t.Fatal("SigNoz fields must default to empty strings")
	}
	t.Setenv("SIGNOZ_UI_URL", "https://signoz.example.test")
	t.Setenv("SIGNOZ_LOGIN_EMAIL", "operator@example.test")
	t.Setenv("SIGNOZ_LOGIN_PASSWORD", "private-password")
	cfg = Load()
	if cfg.SigNozUIURL != "https://signoz.example.test" || cfg.SigNozLoginEmail != "operator@example.test" || cfg.SigNozLoginPassword != "private-password" {
		t.Fatal("SigNoz runtime environment override was ignored")
	}
}
