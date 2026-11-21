package miniappauth

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryEntry struct {
	grant   Grant
	expires time.Time
}
type memoryStore struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
	used    map[string]bool
	now     func() time.Time
}

func (s *memoryStore) ReserveLaunch(_ context.Context, hash string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.used[hash] {
		return false, nil
	}
	s.used[hash] = true
	return true, nil
}
func (s *memoryStore) Put(_ context.Context, kind, token string, grant Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[kind+token] = memoryEntry{grant, s.now().Add(2 * time.Minute)}
	return nil
}
func (s *memoryStore) Read(_ context.Context, kind, token string) (*Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[kind+token]
	if !ok || !entry.expires.After(s.now()) {
		return nil, ErrExpired
	}
	copy := entry.grant
	return &copy, nil
}
func (s *memoryStore) Consume(_ context.Context, kind, token string) (*Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[kind+token]
	delete(s.entries, kind+token)
	if !ok || !entry.expires.After(s.now()) {
		return nil, ErrExpired
	}
	copy := entry.grant
	return &copy, nil
}

type fakeResolver struct {
	link    Link
	blocked bool
	current string
	ensured int
}

func (r *fakeResolver) Resolve(_ context.Context, id int64) (*Link, error) {
	if r.blocked || id != r.link.TelegramID {
		return nil, ErrNotLinked
	}
	link := r.link
	return &link, nil
}
func (r *fakeResolver) Check(_ context.Context, link Link) error {
	if r.blocked || link != r.link {
		return ErrNotLinked
	}
	return nil
}
func (r *fakeResolver) EnsureCredential(_ context.Context, _ Link) error { r.ensured++; return nil }
func (r *fakeResolver) CurrentIdentity(_ context.Context, _ string) (string, error) {
	return r.current, nil
}

var keyOnce sync.Once
var testKey *rsa.PrivateKey

func configuration(t *testing.T) Config {
	t.Helper()
	keyOnce.Do(func() {
		var err error
		testKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
	})
	der, err := x509.MarshalPKCS8PrivateKey(testKey)
	if err != nil {
		t.Fatal(err)
	}
	return Config{Issuer: "https://map.test/api/auth/telegram/oidc", RedirectURI: "https://auth.test/self-service/methods/oidc/callback/telegram-miniapp", ClientSecret: strings.Repeat("synthetic-", 4), PrivateKeyB64: base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), BotToken: "123456:synthetic-bot-token"}
}
func fixture(t *testing.T) (*Handler, *fakeResolver, *time.Time) {
	t.Helper()
	now := time.Unix(1791190800, 0)
	store := &memoryStore{entries: map[string]memoryEntry{}, used: map[string]bool{}, now: func() time.Time { return now }}
	resolver := &fakeResolver{link: Link{IdentityID: "00000000-0000-4000-8000-000000000001", TelegramID: 101, Subject: "synthetic-link-version"}}
	h, err := NewHandler(configuration(t), store, resolver)
	if err != nil {
		t.Fatal(err)
	}
	h.now = func() time.Time { return now }
	return h, resolver, &now
}
func launch(now time.Time) string {
	values := url.Values{"auth_date": {fmt.Sprint(now.Unix())}, "user": {`{"id":101,"username":"display_only"}`}, "query_id": {"synthetic-launch"}}
	keys := []string{}
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte("123456:synthetic-bot-token"))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}
func req(t *testing.T, h http.Handler, method, path, body string, cookie *http.Cookie, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "https://map.test/telegram-miniapp/"+path, strings.NewReader(body))
	if method == "POST" {
		if path == "prepare" {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Origin", "https://map.test")
		} else {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if auth {
		r.SetBasicAuth(ClientID, configuration(t).ClientSecret)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func prepare(t *testing.T, h *Handler, now time.Time) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"initData": launch(now)})
	w := req(t, h, "POST", "prepare", string(body), nil, false)
	if w.Code != 200 {
		t.Fatalf("prepare %d: %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/api/auth/telegram" {
		t.Fatalf("unsafe cookie: %+v", cookies)
	}
	if strings.Contains(w.Body.String(), "display_only") || strings.Contains(w.Body.String(), cookies[0].Value) {
		t.Fatal("private launch exposed")
	}
	return cookies[0]
}
func authURL(t *testing.T, extra url.Values) string {
	t.Helper()
	values := url.Values{"client_id": {ClientID}, "redirect_uri": {configuration(t).RedirectURI}, "response_type": {"code"}, "scope": {"openid"}, "state": {"synthetic-state"}, "nonce": {"synthetic-nonce"}}
	for k, v := range extra {
		values[k] = v
	}
	return "authorize?" + values.Encode()
}
func authorize(t *testing.T, h *Handler, cookie *http.Cookie, extra url.Values) string {
	t.Helper()
	w := req(t, h, "GET", authURL(t, extra), "", cookie, false)
	if w.Code != 303 {
		t.Fatalf("authorize %d: %s", w.Code, w.Body.String())
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Scheme+"://"+location.Host+location.Path != configuration(t).RedirectURI || location.Query().Get("state") != "synthetic-state" {
		t.Fatal("unsafe callback")
	}
	return location.Query().Get("code")
}
func exchange(t *testing.T, h *Handler, code, verifier string, auth bool) *httptest.ResponseRecorder {
	t.Helper()
	values := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {configuration(t).RedirectURI}}
	if verifier != "" {
		values.Set("code_verifier", verifier)
	}
	return req(t, h, "POST", "token", values.Encode(), nil, auth)
}

func TestBridgeLinkedAccountTokenSignatureAndSingleUse(t *testing.T) {
	h, r, now := fixture(t)
	code := authorize(t, h, prepare(t, h, *now), nil)
	w := exchange(t, h, code, "", true)
	if w.Code != 200 {
		t.Fatalf("exchange %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(body.IDToken, ".")
	if len(parts) != 3 {
		t.Fatal("invalid JWT")
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(&testKey.PublicKey, crypto.SHA256, digest[:], sig); err != nil {
		t.Fatal(err)
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	json.Unmarshal(payload, &claims)
	if claims["iss"] != configuration(t).Issuer || claims["aud"] != ClientID || claims["sub"] != r.link.Subject || claims["nonce"] != "synthetic-nonce" || claims["exp"] != float64(now.Unix()+120) {
		t.Fatalf("invalid claims: %v", claims)
	}
	if body.AccessToken == "" || r.ensured != 1 || strings.Contains(string(payload), "display_only") {
		t.Fatal("bad credential or private claims")
	}
	if exchange(t, h, code, "", true).Code != 400 {
		t.Fatal("code replay accepted")
	}
}
func TestBridgeLaunchValidationAndCanonicalReplay(t *testing.T) {
	h, r, now := fixture(t)
	body, _ := json.Marshal(map[string]string{"initData": launch(*now)})
	cross := httptest.NewRequest("POST", "https://map.test/telegram-miniapp/prepare", strings.NewReader(string(body)))
	cross.Header.Set("Origin", "https://attacker.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cross)
	if w.Code != 403 {
		t.Fatal("cross-origin login accepted")
	}
	r.blocked = true
	if req(t, h, "POST", "prepare", string(body), nil, false).Code != 404 {
		t.Fatal("unlinked identity accepted")
	}
	r.blocked = false
	prepare(t, h, *now)
	if req(t, h, "POST", "prepare", string(body), nil, false).Code != 409 {
		t.Fatal("launch replay")
	}
	parts := strings.Split(launch(*now), "&")
	sort.Sort(sort.Reverse(sort.StringSlice(parts)))
	body, _ = json.Marshal(map[string]string{"initData": strings.Join(parts, "&")})
	if req(t, h, "POST", "prepare", string(body), nil, false).Code != 409 {
		t.Fatal("canonical replay")
	}
	if req(t, h, "POST", "prepare", `{"initData":"invalid"}`, nil, false).Code != 401 {
		t.Fatal("invalid launch")
	}
	if req(t, h, "POST", "prepare", `{"initData":"invalid","identityId":"attacker-selected"}`, nil, false).Code != 400 {
		t.Fatal("caller-selected account")
	}
}
func TestBridgeClientRedirectPKCEAndAuthenticatedExchange(t *testing.T) {
	h, _, now := fixture(t)
	cookie := prepare(t, h, *now)
	for _, extra := range []url.Values{{"client_id": {"other"}}, {"redirect_uri": {"https://attacker.test"}}, {"response_type": {"token"}}, {"scope": {"profile"}}, {"code_challenge": {"bad"}, "code_challenge_method": {"plain"}}} {
		if req(t, h, "GET", authURL(t, extra), "", cookie, false).Code != 400 {
			t.Fatalf("unsafe authorization %v", extra)
		}
	}
	verifier := strings.Repeat("synthetic-verifier-", 3)
	digest := sha256.Sum256([]byte(verifier))
	challenge := url.Values{"code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "code_challenge_method": {"S256"}}
	code := authorize(t, h, cookie, challenge)
	if exchange(t, h, code, "", false).Code != 401 {
		t.Fatal("unauthenticated exchange")
	}
	if exchange(t, h, code, verifier, true).Code != 200 {
		t.Fatal("PKCE failed")
	}
	h, _, now = fixture(t)
	code = authorize(t, h, prepare(t, h, *now), challenge)
	if exchange(t, h, code, strings.Repeat("wrong", 10), true).Code != 400 {
		t.Fatal("wrong verifier")
	}
	if exchange(t, h, code, verifier, true).Code != 400 {
		t.Fatal("failed exchange not consumed")
	}
}
func TestBridgeExpirationRevokedLinkAndAccountConflict(t *testing.T) {
	h, r, now := fixture(t)
	cookie := prepare(t, h, *now)
	*now = now.Add(121 * time.Second)
	if req(t, h, "GET", authURL(t, nil), "", cookie, false).Code != 401 {
		t.Fatal("expired launch")
	}
	h, r, now = fixture(t)
	cookie = prepare(t, h, *now)
	r.blocked = true
	if req(t, h, "GET", authURL(t, nil), "", cookie, false).Code != 401 {
		t.Fatal("revoked link authorized")
	}
	h, r, now = fixture(t)
	code := authorize(t, h, prepare(t, h, *now), nil)
	r.blocked = true
	if exchange(t, h, code, "", true).Code != 400 {
		t.Fatal("revoked link exchanged")
	}
	h, r, now = fixture(t)
	code = authorize(t, h, prepare(t, h, *now), nil)
	*now = now.Add(121 * time.Second)
	if exchange(t, h, code, "", true).Code != 400 {
		t.Fatal("expired code")
	}
	h, r, now = fixture(t)
	r.current = "another-account"
	body, _ := json.Marshal(map[string]string{"initData": launch(*now)})
	if req(t, h, "POST", "prepare", string(body), nil, false).Code != 409 || r.ensured != 0 {
		t.Fatal("account silently replaced")
	}
	r.current = r.link.IdentityID
	w := req(t, h, "POST", "prepare", string(body), nil, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"authenticated":true`) || len(w.Result().Cookies()) != 0 {
		t.Fatal("matching session not reused")
	}
}
func TestBridgeAtomicTicketAndPublicMetadata(t *testing.T) {
	h, _, now := fixture(t)
	cookie := prepare(t, h, *now)
	var group sync.WaitGroup
	statuses := make(chan int, 2)
	for range 2 {
		group.Add(1)
		go func() { defer group.Done(); statuses <- req(t, h, "GET", authURL(t, nil), "", cookie, false).Code }()
	}
	group.Wait()
	close(statuses)
	wins := 0
	for status := range statuses {
		if status == 303 {
			wins++
		}
	}
	if wins != 1 {
		t.Fatal("multiple ticket winners")
	}
	for _, path := range []string{".well-known/openid-configuration", "jwks"} {
		w := req(t, h, "GET", path, "", nil, false)
		if w.Code != 200 || strings.Contains(w.Body.String(), "PRIVATE KEY") || strings.Contains(w.Body.String(), configuration(t).ClientSecret) || strings.Contains(w.Body.String(), "synthetic-bot-token") {
			t.Fatal("private metadata exposed")
		}
	}
	disabled, err := NewHandler(Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if req(t, disabled, "POST", "prepare", `{"initData":"invalid"}`, nil, false).Code != 503 {
		t.Fatal("unconfigured bridge active")
	}
}

func TestBridgeRejectsAccountChangedAfterPreparation(t *testing.T) {
	h, resolver, now := fixture(t)
	cookie := prepare(t, h, *now)
	resolver.current = "different-account"
	if req(t, h, "GET", "proof", "", cookie, false).Code != 409 {
		t.Fatal("proof accepted a different browser account")
	}
	if req(t, h, "GET", authURL(t, nil), "", cookie, false).Code != 409 {
		t.Fatal("authorization accepted a different browser account")
	}
}
