package news

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func productionSecurityConfig() Config {
	return Config{
		Environment: "production", PublicURL: "https://news.example.com",
		S3Endpoint: "https://s3.example.com", S3Region: "us-east-1",
		S3Bucket: "creata", S3AccessKey: "test-access", S3SecretKey: "test-secret",
		SessionSecret: strings.Repeat("s", 32), WorkerToken: strings.Repeat("w", 32),
		BotServiceToken: strings.Repeat("b", 32),
	}
}

func TestProductionRequiresCompleteS3AndIndependentSecrets(t *testing.T) {
	for name, change := range map[string]func(*Config){
		"missing bucket": func(c *Config) { c.S3Bucket = "" },
		"insecure S3":    func(c *Config) { c.S3Endpoint = "http://s3.example.com" },
		"reused secret":  func(c *Config) { c.WorkerToken = c.BotServiceToken },
		"invalid public": func(c *Config) { c.PublicURL = "https://news.example.com/not-an-origin" },
	} {
		t.Run(name, func(t *testing.T) {
			c := productionSecurityConfig()
			change(&c)
			if _, err := New(nil, c); err == nil {
				t.Fatal("invalid production configuration was accepted")
			}
		})
	}
}

func TestPublicRateLimitsBehindTrustedProxy(t *testing.T) {
	t.Setenv("CREATA_TRUSTED_PROXIES", "10.20.0.0/24,10.40.0.2/32")
	t.Setenv("CREATA_SSR_TOKEN", strings.Repeat("x", 32))
	env := ConfigFromEnv()
	cfg := productionSecurityConfig()
	cfg.TrustedProxies, cfg.SSRToken = env.TrustedProxies, env.SSRToken
	server, err := New(nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	server.mux = mux
	call := func(peer, chain, method, path string, ssr bool) int {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = peer
		req.Header.Set("X-Forwarded-For", chain)
		if ssr {
			req.Header.Set("X-Creata-SSR-Token", env.SSRToken)
		}
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		return res.Code
	}
	public := base + "/public/articles"
	for i := 0; i < 240; i++ {
		if status := call("10.20.0.7:99", "203.0.113.10, 10.40.0.2", "GET", public, false); status != 204 {
			t.Fatalf("trusted client exhausted early at %d: %d", i, status)
		}
	}
	for _, tc := range []struct {
		name, peer, chain, method, path string
		ssr                             bool
		want                            int
	}{
		{"same reader", "10.20.0.8:99", "203.0.113.10, 10.40.0.2", "GET", public, false, 429},
		{"spoofed left hand value", "10.20.0.7:99", "203.0.113.99, 203.0.113.10, 10.40.0.2", "GET", public, false, 429},
		{"second reader", "10.20.0.7:99", "203.0.113.11, 10.40.0.2", "GET", public, false, 204},
		{"separate write quota", "10.20.0.7:99", "203.0.113.10, 10.40.0.2", "POST", base + "/public/events", false, 204},
		{"authenticated SSR", "10.20.0.7:99", "203.0.113.10, 10.40.0.2", "GET", public, true, 204},
		{"SSR token does not extend account quota", "10.20.0.7:99", "203.0.113.10, 10.40.0.2", "GET", base + "/account/saved", true, 429},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := call(tc.peer, tc.chain, tc.method, tc.path, tc.ssr); got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
	for i := 0; i < 240; i++ {
		if got := call("198.51.100.4:99", "203.0.113.21", "GET", public, false); got != 204 {
			t.Fatalf("direct client exhausted early: %d", got)
		}
	}
	if got := call("198.51.100.4:99", "203.0.113.22", "GET", public, false); got != 429 {
		t.Fatalf("untrusted forwarded address changed the quota: %d", got)
	}
}
