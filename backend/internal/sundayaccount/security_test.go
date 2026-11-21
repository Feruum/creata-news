package sundayaccount

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdentityValidation(t *testing.T) {
	email, err := normalizeEmail("  USER@Example.com ")
	if err != nil || email != "user@example.com" {
		t.Fatalf("normalization: %q %v", email, err)
	}
	for _, invalid := range []string{"", "name <u@example.com>", "bad", "u@example.com\r\nBcc: x@example.com"} {
		if _, err := normalizeEmail(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func TestOriginAndCSRF(t *testing.T) {
	h := New(nil, Config{Origin: "https://sunday.example", Secret: strings.Repeat("s", 32)}, nil)
	for _, origin := range []string{"", "null", "https://evil.example", "https://sunday.example.evil"} {
		r := httptest.NewRequest("POST", "/api/sundayopen/account/logout", nil)
		r.Header.Set("Origin", origin)
		if h.originAllowed(r) {
			t.Fatalf("accepted origin %q", origin)
		}
	}
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("Origin", "https://sunday.example")
	if !h.originAllowed(r) {
		t.Fatal("rejected configured origin")
	}
	if h.csrf("account-a", "token-a") == h.csrf("account-b", "token-a") || h.csrf("account-a", "token-a") == h.csrf("account-a", "token-b") {
		t.Fatal("CSRF token not bound to account and session")
	}
}

func TestTokensAndCookie(t *testing.T) {
	token, err := randomToken()
	if err != nil || len(token) < 40 {
		t.Fatalf("token: %v", err)
	}
	second, _ := randomToken()
	if token == second {
		t.Fatal("repeated token")
	}
	h := New(nil, Config{Origin: "https://sunday.example", Secret: strings.Repeat("s", 32)}, nil)
	w := httptest.NewRecorder()
	h.setCookie(w, token, 3600)
	cookie := w.Result().Cookies()[0]
	if cookie.Name != "sundayopen_session" || cookie.Path != "/api/sundayopen" || !cookie.HttpOnly || !cookie.Secure || cookie.Domain != "" {
		t.Fatalf("unsafe cookie: %+v", cookie)
	}
	if h.digest("code", "123456") == h.digest("session", "123456") {
		t.Fatal("hash domains overlap")
	}
}

func TestCookieSecureRequiresEveryOriginToBeLocalHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, origins string
		secure        bool
	}{
		{"production", "https://sunday.example", true},
		{"localhost first", "http://localhost:3002,https://sunday.example", true},
		{"production first", "https://sunday.example,http://localhost:3002", true},
		{"local only", " http://localhost:3002 ,http://127.0.0.1:3003,http://[::1]:3004", false},
		{"local HTTPS", "https://localhost:3002", true},
		{"remote HTTP", "http://localhost:3002,http://sunday.example", true},
		{"lookalike host", "http://localhost.evil.example", true},
		{"missing configuration", "", true},
		{"invalid configuration", "http://localhost:3002,%", true},
		{"empty configuration entry", "http://localhost:3002,", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(nil, Config{Origin: tc.origins}, nil)
			for _, age := range []int{3600, -1} {
				w := httptest.NewRecorder()
				h.setCookie(w, "token", age)
				cookie := w.Result().Cookies()[0]
				if cookie.Secure != tc.secure {
					t.Fatalf("origins %q, age %d: Secure=%v, want %v", tc.origins, age, cookie.Secure, tc.secure)
				}
			}
		})
	}
}

func TestForwardedIPTraversesOnlyConfiguredTrustedHops(t *testing.T) {
	for _, tc := range []struct {
		name, peer, trusted, forwarded, want string
	}{
		{"trusted chain", "172.20.0.2:1234", "172.20.0.0/16, 192.0.2.0/24", "203.0.113.5, 192.0.2.10", "203.0.113.5"},
		{"spoof before client", "172.20.0.2:1234", "172.20.0.0/16,192.0.2.0/24", "198.51.100.99, 203.0.113.5, 192.0.2.10", "203.0.113.5"},
		{"untrusted intermediate", "172.20.0.2:1234", "172.20.0.0/16", "198.51.100.99, 203.0.113.5", "203.0.113.5"},
		{"untrusted peer", "203.0.113.6:1234", "172.20.0.0/16", "198.51.100.99", "203.0.113.6"},
		{"no configured trust", "172.20.0.2:1234", "", "198.51.100.99", "172.20.0.2"},
		{"invalid trust", "172.20.0.2:1234", "172.20.0.2,invalid", "198.51.100.99", "172.20.0.2"},
		{"invalid final hop", "172.20.0.2:1234", "172.20.0.0/16", "198.51.100.99, invalid", "172.20.0.2"},
		{"invalid inner hop", "172.20.0.2:1234", "172.20.0.0/16,192.0.2.0/24", "198.51.100.99, invalid, 192.0.2.10", "192.0.2.10"},
		{"IPv6 chain", "[fd00::2]:1234", "fd00::/64,2001:db8:1::/64", "2001:db8:2::5, 2001:db8:1::10", "2001:db8:2::5"},
		{"missing forwarded header", "172.20.0.2:1234", "172.20.0.0/16", "", "172.20.0.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := New(nil, Config{TrustedProxyCIDRs: tc.trusted}, nil)
			r := httptest.NewRequest("POST", "/", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-For", tc.forwarded)
			r.Header.Set("True-Client-IP", "198.51.100.99")
			r.Header.Set("X-Real-IP", "198.51.100.99")
			if got := h.requestIP(r); got != tc.want {
				t.Fatalf("client IP=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestForwardedIPIncludesAllHeaderLines(t *testing.T) {
	h := New(nil, Config{TrustedProxyCIDRs: "172.20.0.0/16"}, nil)
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "172.20.0.2:1234"
	r.Header.Add("X-Forwarded-For", "198.51.100.99")
	r.Header.Add("X-Forwarded-For", "203.0.113.5")
	if got := h.requestIP(r); got != "203.0.113.5" {
		t.Fatalf("trusted spoofed earlier header line: %q", got)
	}
}

func TestForwardedIPTrustAndSpoofing(t *testing.T) {
	h := New(nil, Config{TrustedProxyCIDRs: "172.20.0.0/16"}, nil)
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "172.20.0.2:1234"
	r.Header.Set("X-Forwarded-For", "spoofed, 203.0.113.5")
	r.Header.Set("True-Client-IP", "192.0.2.1")
	r.Header.Set("X-Real-IP", "192.0.2.1")
	if ip := h.requestIP(r); ip != "203.0.113.5" {
		t.Fatalf("wrong client %s", ip)
	}
	r.RemoteAddr = "203.0.113.6:2345"
	if ip := h.requestIP(r); ip != "203.0.113.6" {
		t.Fatalf("trusted public spoof %s", ip)
	}
	r.RemoteAddr = "172.20.0.2:1234"
	r.Header.Set("X-Forwarded-For", "not-an-ip")
	if ip := h.requestIP(r); ip != "172.20.0.2" {
		t.Fatalf("invalid forwarded IP %s", ip)
	}
}

func TestPhoneNormalizationRejectsAmbiguousCharacters(t *testing.T) {
	for _, raw := range []string{"+7 (700) 123-45-67", "77001234567", " +77001234567 "} {
		if phone, err := normalizePhone(raw); err != nil || phone != "+77001234567" {
			t.Fatalf("phone %q: %s %v", raw, phone, err)
		}
	}
	for _, raw := range []string{"123", "+1234567890123456", "+7letters7001234567", "7+7001234567", "+7700\r\n1234567"} {
		if _, err := normalizePhone(raw); err == nil {
			t.Fatalf("invalid phone accepted %q", raw)
		}
	}
}
