package news

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPublicArticleHidesEditorialNotes(t *testing.T) {
	a := Doc{"id": "one", "reviewNotes": []any{"Private reviewer comment"}, "publicationNote": "Private scheduling note", "title": localized("Public headline")}
	public := publicArticle(a)
	if len(array(public["reviewNotes"])) != 0 || public["publicationNote"] != nil {
		t.Fatalf("private content leaked: %v", public)
	}
	if len(array(a["reviewNotes"])) != 1 {
		t.Fatal("sanitization mutated editorial source")
	}
}

func TestRequestRateLimitResetsAndSeparatesClients(t *testing.T) {
	l := newBurstLimiter(2, time.Minute)
	start := time.Now()
	if !l.allow("one", start) || !l.allow("one", start) || l.allow("one", start) {
		t.Fatal("client burst was not bounded")
	}
	if !l.allow("two", start) {
		t.Fatal("clients share quota")
	}
	if !l.allow("one", start.Add(time.Minute)) {
		t.Fatal("window did not reset")
	}
}

func TestProductionRejectsDevelopmentAuthentication(t *testing.T) {
	_, e := New(nil, Config{SessionSecret: strings.Repeat("s", 40), WorkerToken: strings.Repeat("w", 40), BotServiceToken: strings.Repeat("b", 40), Environment: "production", DevAuth: true})
	if e == nil {
		t.Fatal("production accepted development authentication")
	}
}

func TestOIDCSignaturesAndClaims(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, Doc{"keys": []any{Doc{"kid": "key-1", "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
	}))
	defer keys.Close()
	s := &Server{client: keys.Client()}
	cfg := Doc{"issuer": "https://issuer.example", "jwks_uri": keys.URL}
	claims := Doc{"iss": "https://issuer.example", "sub": "subject-1", "aud": "client-1", "nonce": "nonce-1", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}
	sign := func(c Doc) string {
		h, _ := json.Marshal(Doc{"alg": "RS256", "kid": "key-1"})
		p, _ := json.Marshal(c)
		unsigned := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
		sum := sha256.Sum256([]byte(unsigned))
		signature, e := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if e != nil {
			t.Fatal(e)
		}
		return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
	}
	if _, e = s.verifyIDToken(sign(claims), cfg, "client-1", "nonce-1"); e != nil {
		t.Fatal(e)
	}
	for _, field := range []string{"aud", "nonce", "iss"} {
		bad := clone(claims)
		bad[field] = "wrong"
		if _, e = s.verifyIDToken(sign(bad), cfg, "client-1", "nonce-1"); e == nil {
			t.Fatalf("wrong %s accepted", field)
		}
	}
	expired := clone(claims)
	expired["exp"] = time.Now().Add(-time.Hour).Unix()
	if _, e = s.verifyIDToken(sign(expired), cfg, "client-1", "nonce-1"); e == nil {
		t.Fatal("expired token accepted")
	}
	token := sign(claims)
	parts := strings.Split(token, ".")
	tampered, _ := json.Marshal(Doc{"sub": "attacker", "aud": "client-1", "iss": cfg["issuer"], "nonce": "nonce-1", "exp": time.Now().Add(time.Hour).Unix()})
	parts[1] = base64.RawURLEncoding.EncodeToString(tampered)
	if _, e = s.verifyIDToken(strings.Join(parts, "."), cfg, "client-1", "nonce-1"); e == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestAPIPrivacyAndServiceSeparation(t *testing.T) {
	a := setupAPI(t)
	a.login("admin")
	draft := validArticle()
	draft["reviewNotes"] = []any{"PRIVATE REVIEW COMMENT"}
	art := a.request("POST", "/admin/articles", draft, 201, "")
	for _, action := range []string{"approve", "publish"} {
		a.request("POST", "/admin/articles/"+str(art["id"])+"/actions", Doc{"action": action, "expectedRevision": 1}, 200, "")
	}
	pub := a.request("GET", "/public/articles/"+str(art["slug"]), nil, 200, "")
	if len(array(pub["reviewNotes"])) != 0 {
		t.Fatal("private notes in public article")
	}
	admin := a.request("GET", "/admin/articles/"+str(art["id"]), nil, 200, "")
	if len(array(admin["reviewNotes"])) != 1 {
		t.Fatal("editorial notes removed from admin")
	}
	a.request("GET", "/internal/ai-config", nil, 401, a.bot)
	a.request("POST", "/internal/bot/users/upsert", Doc{"telegramId": "123"}, 401, a.worker)
	a.csrf = "bad"
	a.request("POST", "/admin/articles", validArticle(), 403, "")
	if e := a.store.Write(context.Background(), func(tx *Tx) error {
		u, e := tx.Get(context.Background(), "user", "dev-admin")
		if e != nil {
			return e
		}
		u["roles"] = []any{"admin"}
		return tx.Put(context.Background(), "user", u)
	}); e != nil {
		t.Fatal(e)
	}
	a.csrf = ""
	me := a.request("GET", "/auth/me", nil, 200, "")
	a.csrf = str(me["csrfToken"])
	a.request("POST", "/admin/articles", validArticle(), 403, "")
}
