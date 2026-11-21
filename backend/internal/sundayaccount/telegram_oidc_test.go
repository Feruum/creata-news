package sundayaccount

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type telegramTestTransport func(*http.Request) (*http.Response, error)

func (f telegramTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func oidcFixture(t *testing.T) (*Handler, *rsa.PrivateKey, *atomic.Int32) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	h := widgetTestHandler()
	calls := new(atomic.Int32)
	h.client = &http.Client{Transport: telegramTestTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.String() != "https://oauth.telegram.org/.well-known/jwks.json" {
			t.Errorf("unexpected key URL %s", r.URL)
		}
		body, _ := json.Marshal(map[string]any{"keys": []any{map[string]string{
			"kid": "telegram-test-key", "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}
	return h, key, calls
}

func oidcClaims(nonce string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": "https://oauth.telegram.org", "aud": "12345", "sub": "telegram-subject",
		"id": "123456789", "name": "Sunday Visitor", "preferred_username": "sunday_visitor",
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(), "nonce": nonce,
	}
}

func signOIDC(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "telegram-test-key"
	raw, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestTelegramOIDCVerifiesSignedIdentityAndCachesOnlyTelegramKeys(t *testing.T) {
	h, key, calls := oidcFixture(t)
	nonce := strings.Repeat("n", 43)
	raw := signOIDC(t, key, oidcClaims(nonce))
	for range 2 {
		identity, err := h.verifyTelegramIDToken(context.Background(), raw, nonce, time.Now())
		if err != nil || identity.ID != 123456789 || identity.Username == nil || *identity.Username != "sunday_visitor" {
			t.Fatalf("verified identity %+v: %v", identity, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expected cached key, fetched %d times", calls.Load())
	}
}

func TestTelegramOIDCVerifiesExactIntegerIDs(t *testing.T) {
	h, key, _ := oidcFixture(t)
	nonce := strings.Repeat("n", 43)
	for _, tc := range []struct {
		name string
		id   any
		want int64
	}{
		{"string", "123456789", 123456789},
		{"number", int64(123456789), 123456789},
		{"minimum string", "1", 1},
		{"minimum number", int64(1), 1},
		{"maximum string", "9007199254740991", 1<<53 - 1},
		{"maximum number", int64(1<<53 - 1), 1<<53 - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := oidcClaims(nonce)
			claims["id"], claims["sub"] = tc.id, "987654321"
			identity, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, key, claims), nonce, time.Now())
			if err != nil || identity.ID != tc.want {
				t.Fatalf("verified ID %d, want %d: %v", identity.ID, tc.want, err)
			}
		})
	}
}

func TestTelegramOIDCRejectsInvalidIDs(t *testing.T) {
	h, key, _ := oidcFixture(t)
	nonce := strings.Repeat("n", 43)
	for _, tc := range []struct {
		name string
		id   any
	}{
		{"null", nil},
		{"empty string", ""},
		{"nonnumeric string", "telegram-user"},
		{"whitespace string", " 123456789 "},
		{"fractional string", "123.5"},
		{"fractional number", json.Number("123.5")},
		{"decimal string", "123.0"},
		{"decimal number", json.Number("123.0")},
		{"exponent string", "123e3"},
		{"exponent number", json.Number("123e3")},
		{"overflow string", "9223372036854775808"},
		{"overflow number", json.Number("9223372036854775808")},
		{"unsafe string", "9007199254740992"},
		{"unsafe number", int64(1 << 53)},
		{"zero string", "0"},
		{"zero number", int64(0)},
		{"negative string", "-1"},
		{"negative number", int64(-1)},
		{"boolean", true},
		{"array", []int64{123456789}},
		{"object", map[string]int64{"id": 123456789}},
		{"missing", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := oidcClaims(nonce)
			claims["id"] = tc.id
			if tc.name == "missing" {
				delete(claims, "id")
			}
			if _, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, key, claims), nonce, time.Now()); !errors.Is(err, errTelegramIDToken) {
				t.Fatalf("expected invalid ID token, got %v", err)
			}
		})
	}
}

func TestTelegramOIDCRejectsUntrustedClaims(t *testing.T) {
	h, key, _ := oidcFixture(t)
	nonce := strings.Repeat("n", 43)
	for _, tc := range []struct {
		name   string
		mutate func(jwt.MapClaims)
	}{
		{"issuer", func(c jwt.MapClaims) { c["iss"] = "https://attacker.example" }},
		{"audience", func(c jwt.MapClaims) { c["aud"] = "99999" }},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }},
		{"missing expiry", func(c jwt.MapClaims) { delete(c, "exp") }},
		{"missing issue time", func(c jwt.MapClaims) { delete(c, "iat") }},
		{"future", func(c jwt.MapClaims) { c["iat"] = time.Now().Add(time.Hour).Unix() }},
		{"old", func(c jwt.MapClaims) { c["iat"] = time.Now().Add(-time.Hour).Unix() }},
		{"not yet valid", func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Hour).Unix() }},
		{"nonce", func(c jwt.MapClaims) { c["nonce"] = "another-browser" }},
		{"missing nonce", func(c jwt.MapClaims) { delete(c, "nonce") }},
		{"missing subject", func(c jwt.MapClaims) { delete(c, "sub") }},
		{"invalid username", func(c jwt.MapClaims) { c["preferred_username"] = "bad/name" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := oidcClaims(nonce)
			tc.mutate(claims)
			if _, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, key, claims), nonce, time.Now()); err == nil {
				t.Fatal("accepted invalid claims")
			}
		})
	}
	if _, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, key, oidcClaims(nonce)), "", time.Now()); err == nil {
		t.Fatal("accepted missing browser nonce")
	}
	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, oidcClaims(nonce))
	bad.Header["kid"] = "telegram-test-key"
	forged, _ := bad.SignedString([]byte("attacker"))
	if _, err := h.verifyTelegramIDToken(context.Background(), forged, nonce, time.Now()); err == nil {
		t.Fatal("accepted symmetric signature")
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	if _, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, other, oidcClaims(nonce)), nonce, time.Now()); err == nil {
		t.Fatal("accepted forged RSA signature")
	}
}

func TestTelegramOIDCChallengeBindsTokenToBrowserAndExpires(t *testing.T) {
	h := widgetTestHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", prefix+"telegram-challenge", nil)
	r.Header.Set("Origin", "https://sunday.example")
	h.ServeHTTP(w, r)
	var result struct {
		ClientID string `json:"clientId"`
		Nonce    string `json:"nonce"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.ClientID != "12345" || len(result.Nonce) != 43 {
		t.Fatalf("challenge %d %s: %v", w.Code, w.Body, err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].MaxAge != 300 {
		t.Fatalf("unsafe nonce cookie %+v", cookies)
	}
	proof := httptest.NewRequest("POST", prefix+"telegram-login", nil)
	proof.AddCookie(cookies[0])
	if h.telegramBrowserNonce(proof, result.Nonce, time.Now()) != result.Nonce {
		t.Fatal("browser nonce did not verify")
	}
	if h.telegramBrowserNonce(proof, result.Nonce, time.Now().Add(6*time.Minute)) != "" {
		t.Fatal("expired browser nonce verified")
	}
	cookies[0].Value += "tampered"
	proof = httptest.NewRequest("POST", prefix+"telegram-login", nil)
	proof.AddCookie(cookies[0])
	if h.telegramBrowserNonce(proof, result.Nonce, time.Now()) != "" {
		t.Fatal("tampered nonce verified")
	}
}

func TestTelegramOIDCChallengesInDifferentTabsStayIndependent(t *testing.T) {
	h := widgetTestHandler()
	r := httptest.NewRequest("POST", prefix+"telegram-challenge", nil)
	r.Header.Set("Origin", "https://sunday.example")
	proof := httptest.NewRequest("POST", prefix+"telegram-login", nil)
	var nonces []string
	for range 2 {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var result struct {
			Nonce string `json:"nonce"`
		}
		if json.Unmarshal(w.Body.Bytes(), &result) != nil || w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		nonces = append(nonces, result.Nonce)
		proof.AddCookie(w.Result().Cookies()[0])
	}
	if nonces[0] == nonces[1] {
		t.Fatal("reused challenge")
	}
	for _, nonce := range nonces {
		if h.telegramBrowserNonce(proof, nonce, time.Now()) != nonce {
			t.Fatal("another tab replaced the browser challenge")
		}
	}
}

func TestTelegramOIDCSigningKeyOutageFailsClosed(t *testing.T) {
	h, key, _ := oidcFixture(t)
	h.client = &http.Client{Transport: telegramTestTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header)}, nil
	})}
	nonce := strings.Repeat("n", 43)
	_, err := h.verifyTelegramIDToken(context.Background(), signOIDC(t, key, oidcClaims(nonce)), nonce, time.Now())
	if !errors.Is(err, errTelegramKeys) {
		t.Fatalf("expected unavailable signing keys, got %v", err)
	}
}
