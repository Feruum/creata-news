package news

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

const base = "/api/creata/v1"

type Config struct {
	DatabaseURL, Environment, Address                                  string
	PublicURL, APIURL, CookieDomain                                    string
	Origins                                                            []string
	SessionSecret, WorkerToken, BotServiceToken, SSRToken              string
	TrustedProxies                                                     string
	DevAuth, AllowPrivateSources                                       bool
	GoogleClientID, GoogleClientSecret                                 string
	TelegramClientID, TelegramClientSecret                             string
	BotUsername, ChannelID                                             string
	TelegramChannelOnly                                                bool
	TelegramBotUnconfigured                                            bool
	MediaDir, S3Endpoint, S3Region, S3Bucket, S3AccessKey, S3SecretKey string
	PostizAPIKey, PostizAPIURL, PostizMode                             string
}

func ConfigFromEnv() Config {
	telegramBotConfigured, telegramBotKnown := os.LookupEnv("CREATA_TELEGRAM_BOT_CONFIGURED")
	return Config{
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		Environment:             env("CREATA_ENV", "development"),
		Address:                 env("CREATA_ADDRESS", ":5555"),
		PublicURL:               env("CREATA_PUBLIC_URL", "http://localhost:3002"),
		APIURL:                  env("CREATA_API_URL", "http://localhost:5555"),
		CookieDomain:            os.Getenv("CREATA_COOKIE_DOMAIN"),
		Origins:                 strings.Split(env("CREATA_ORIGINS", "http://localhost:3002,http://localhost:6868,http://localhost:5174"), ","),
		SessionSecret:           os.Getenv("CREATA_SESSION_SECRET"),
		WorkerToken:             os.Getenv("CREATA_WORKER_TOKEN"),
		BotServiceToken:         os.Getenv("CREATA_BOT_SERVICE_TOKEN"),
		SSRToken:                os.Getenv("CREATA_SSR_TOKEN"),
		TrustedProxies:          os.Getenv("CREATA_TRUSTED_PROXIES"),
		DevAuth:                 os.Getenv("CREATA_DEV_AUTH") == "1",
		AllowPrivateSources:     os.Getenv("CREATA_ALLOW_PRIVATE_SOURCES") == "1",
		GoogleClientID:          os.Getenv("CREATA_GOOGLE_CLIENT_ID"),
		GoogleClientSecret:      os.Getenv("CREATA_GOOGLE_CLIENT_SECRET"),
		TelegramClientID:        os.Getenv("CREATA_TELEGRAM_CLIENT_ID"),
		TelegramClientSecret:    os.Getenv("CREATA_TELEGRAM_CLIENT_SECRET"),
		BotUsername:             os.Getenv("CREATA_TELEGRAM_BOT_USERNAME"),
		ChannelID:               os.Getenv("CREATA_TELEGRAM_CHANNEL_ID"),
		TelegramChannelOnly:     os.Getenv("CREATA_TELEGRAM_CHANNEL_ONLY") == "1",
		TelegramBotUnconfigured: telegramBotKnown && telegramBotConfigured != "1",
		MediaDir:                env("CREATA_MEDIA_DIR", ".runtime/media"),
		S3Endpoint:              os.Getenv("CREATA_S3_ENDPOINT"),
		S3Region:                env("CREATA_S3_REGION", "us-east-1"),
		S3Bucket:                os.Getenv("CREATA_S3_BUCKET"),
		S3AccessKey:             os.Getenv("CREATA_S3_ACCESS_KEY"),
		S3SecretKey:             os.Getenv("CREATA_S3_SECRET_KEY"),
		PostizAPIKey:            os.Getenv("CREATA_POSTIZ_API_KEY"),
		PostizAPIURL:            os.Getenv("CREATA_POSTIZ_API_URL"),
		PostizMode:              os.Getenv("CREATA_POSTIZ_MODE"),
	}
}

func env(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}

type Server struct {
	postiz                                             *postizClient
	Store                                              *Store
	Config                                             Config
	mux                                                *http.ServeMux
	client                                             *http.Client
	trustedProxies                                     []netip.Prefix
	readLimiter, writeLimiter, authLimiter, ssrLimiter *burstLimiter
}

func (c *Config) Validate() error {
	if e := validateCookieDomain(c); e != nil {
		return e
	}
	secrets := []string{c.SessionSecret, c.WorkerToken, c.BotServiceToken}
	if c.SSRToken != "" {
		secrets = append(secrets, c.SSRToken)
	}
	seen := make(map[string]bool, len(secrets))
	for _, secret := range secrets {
		if len(secret) < 32 || seen[secret] {
			return errors.New("service and session credentials must each contain at least 32 characters and be distinct")
		}
		seen[secret] = true
	}
	if c.Environment == "production" {
		if c.DevAuth || c.AllowPrivateSources {
			return errors.New("development authentication and private source targets are forbidden in production")
		}
		if !productionHTTPS(c.PublicURL, false) {
			return errors.New("production requires a valid HTTPS public origin")
		}
		if !productionHTTPS(c.S3Endpoint, true) || strings.TrimSpace(c.S3Region) == "" ||
			strings.TrimSpace(c.S3Bucket) == "" || strings.TrimSpace(c.S3AccessKey) == "" ||
			strings.TrimSpace(c.S3SecretKey) == "" {
			return errors.New("production requires a valid HTTPS S3 endpoint, region, bucket and credentials")
		}
	}
	return nil
}

func productionHTTPS(raw string, allowPath bool) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil &&
		u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (allowPath || u.Path == "" || u.Path == "/")
}

func New(s *Store, c Config) (*Server, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	proxies, err := parseTrustedProxies(c.TrustedProxies)
	if err != nil {
		return nil, err
	}
	v := &Server{Store: s, Config: c, mux: http.NewServeMux(), trustedProxies: proxies,
		readLimiter: newBurstLimiter(240, time.Minute), writeLimiter: newBurstLimiter(120, time.Minute),
		authLimiter: newBurstLimiter(30, time.Minute), ssrLimiter: newBurstLimiter(6000, time.Minute),
		client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) > 5 {
				return errors.New("too many redirects")
			}
			return nil
		}}}
	v.postiz, err = newPostizClient(c)
	if err != nil {
		return nil, err
	}
	v.routes()
	return v, nil
}

type endpoint func(http.ResponseWriter, *http.Request) error

func (s *Server) handle(pattern string, fn endpoint) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		if e := fn(w, r); e != nil {
			var a *apiError
			if errors.As(e, &a) {
				respond(w, a.status, Doc{"error": a.message})
				return
			}
			slog.Error("api request failed", "path", r.URL.Path, "error", e.Error())
			respond(w, 500, Doc{"error": "Service unavailable"})
		}
	})
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request) (Doc, error) {
	defer r.Body.Close()
	var d Doc
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	if e := dec.Decode(&d); e != nil || d == nil {
		return nil, errStatus(400, "Invalid JSON request")
	}
	return d, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Cache-Control", "no-store")
	if s.Config.Environment == "production" && !strings.HasPrefix(r.URL.Path, base+"/internal/") && r.URL.Path != "/health" && r.Method != http.MethodOptions {
		if !s.allowRequest(r, time.Now()) {
			w.Header().Set("Retry-After", "60")
			respond(w, 429, Doc{"error": "Too many requests; retry shortly"})
			return
		}
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		allowed := false
		for _, o := range s.Config.Origins {
			if origin == strings.TrimSpace(o) {
				allowed = true
			}
		}
		if !allowed {
			respond(w, 403, Doc{"error": "Origin not allowed"})
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Vary", "Origin")
	}
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-CSRF-Token,Authorization")
		w.WriteHeader(204)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	s.mux.ServeHTTP(w, r)
}
func (s *Server) routes() {
	s.handle("GET /health", func(w http.ResponseWriter, r *http.Request) error {
		if e := s.Store.Pool.Ping(r.Context()); e != nil {
			return e
		}
		respond(w, 200, Doc{"status": "ok", "service": "creata-api"})
		return nil
	})
	s.handle("GET "+base+"/auth/me", s.me)
	s.handle("POST "+base+"/auth/dev-login", s.devLogin)
	s.handle("POST "+base+"/auth/logout", s.logout)
	s.handle("GET "+base+"/auth/{provider}/start", s.oauthStart)
	s.handle("GET "+base+"/auth/{provider}/callback", s.oauthCallback)
	s.handle("PATCH "+base+"/account", s.account)
	s.handle("GET "+base+"/account/saved", s.savedList)
	s.handle("POST "+base+"/account/saved/{kind}/{id}", s.savedMutation)
	s.handle("DELETE "+base+"/account/saved/{kind}/{id}", s.savedMutation)
	s.handle("POST "+base+"/account/telegram-link", s.telegramLink)
	s.followedStoryRoutes()
	s.storyFollowUpRoutes()
	s.publicRoutes()
	s.editorialRoutes()
	s.launchRoutes()
	s.newspaperRoutes()
	s.jobRoutes()
	s.digestRoutes()
	s.mediaRoutes()
	s.contentRoutes()
}
func (s *Server) user(r *http.Request) (Doc, error) {
	cookie, e := r.Cookie("creata_session")
	if e != nil {
		return nil, nil
	}
	var uid string
	e = s.Store.Pool.QueryRow(r.Context(), "SELECT user_id FROM creata_sessions WHERE token_hash=$1 AND expires_at>now()", hash(cookie.Value)).Scan(&uid)
	if e != nil {
		return nil, nil
	}
	return s.Store.Get(r.Context(), "user", uid)
}
func (s *Server) require(r *http.Request, admin bool) (Doc, error) {
	u, e := s.user(r)
	if e != nil {
		return nil, e
	}
	if u == nil {
		return nil, errStatus(401, "Sign in required")
	}
	if admin && !staff(u) {
		return nil, errStatus(403, "Editorial access required")
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		cookie, _ := r.Cookie("creata_session")
		if cookie == nil || !hmac.Equal([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.csrf(cookie.Value))) {
			return nil, errStatus(403, "Invalid CSRF token")
		}
	}
	return u, nil
}
func (s *Server) service(r *http.Request, kind string) error {
	want := s.Config.WorkerToken
	if kind == "bot" {
		want = s.Config.BotServiceToken
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !hmac.Equal([]byte(got), []byte(want)) {
		return errStatus(401, "Invalid service credential")
	}
	return nil
}
func (s *Server) csrf(token string) string {
	h := hmac.New(sha256.New, []byte(s.Config.SessionSecret))
	h.Write([]byte("csrf:" + token))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func (s *Server) session(w http.ResponseWriter, ctx context.Context, u Doc) error {
	token := id() + id()
	_, e := s.Store.Pool.Exec(ctx, "INSERT INTO creata_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hash(token), u["id"], time.Now().Add(30*24*time.Hour))
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "creata_session", Value: token, Domain: s.Config.CookieDomain, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(s.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 30 * 86400})
	return nil
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, e := s.user(r)
	if e != nil {
		return e
	}
	d := Doc{"user": safeUser(u), "providers": Doc{"google": s.Config.GoogleClientID != "" && s.Config.GoogleClientSecret != "", "telegram": s.Config.TelegramClientID != "" && s.Config.TelegramClientSecret != ""}, "devAuth": s.Config.DevAuth && s.localRequest(r)}
	if c, e := r.Cookie("creata_session"); e == nil && u != nil {
		d["csrfToken"] = s.csrf(c.Value)
	}
	respond(w, 200, d)
	return nil
}
func safeUser(u Doc) Doc {
	if u == nil {
		return nil
	}
	c := clone(u)
	delete(c, "telegramId")
	delete(c, "provider")
	delete(c, "subject")
	return c
}
func (s *Server) localRequest(r *http.Request) bool {
	host, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	if o := r.Header.Get("Origin"); o != "" {
		u, e := url.Parse(o)
		if e != nil || !isLoopback(u.Hostname()) {
			return false
		}
	}
	return true
}
func isLoopback(host string) bool {
	return host == "localhost" || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}
func (s *Server) devLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.Config.DevAuth || s.Config.Environment == "production" || !s.localRequest(r) {
		return errStatus(404, "Not found")
	}
	d, e := decode(r)
	if e != nil {
		return e
	}
	role := str(d["role"])
	if role == "" {
		role = "admin"
	}
	if !has([]string{"reader", "author", "editor", "chief_editor", "admin"}, role) {
		return errStatus(400, "Invalid role")
	}
	uid := "dev-" + role
	var u Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		u, _ = t.Get(r.Context(), "user", uid)
		if u == nil {
			roles := []any{role}
			if role == "admin" {
				roles = []any{"admin", "chief_editor", "editor", "author"}
			}
			cats, _ := t.List(r.Context(), "category")
			verticals := []any{}
			for _, c := range cats {
				verticals = append(verticals, c["id"])
			}
			u = newUser(uid, "Development "+role, "ru")
			u["roles"] = roles
			u["verticals"] = verticals
			if e := t.Put(r.Context(), "user", u); e != nil {
				return e
			}
			return t.Put(r.Context(), "author", Doc{"id": uid, "name": u["name"], "bio": localized("Creata News editorial team"), "role": "editor"})
		}
		return nil
	})
	if e != nil {
		return e
	}
	if e = s.session(w, r.Context(), u); e != nil {
		return e
	}
	respond(w, 200, Doc{"user": safeUser(u)})
	return nil
}
func newUser(uid, name, lang string) Doc {
	return Doc{"id": uid, "name": name, "locale": locale(lang), "stage": "exploring", "topics": []any{}, "frequency": "daily", "paused": false, "roles": []any{"reader"}, "verticals": []any{}, "telegramLinked": false, "createdAt": now()}
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.require(r, false); e != nil {
		return e
	}
	c, _ := r.Cookie("creata_session")
	_, e := s.Store.Pool.Exec(r.Context(), "DELETE FROM creata_sessions WHERE token_hash=$1", hash(c.Value))
	http.SetCookie(w, &http.Cookie{Name: "creata_session", Domain: s.Config.CookieDomain, Path: "/", HttpOnly: true, MaxAge: -1, Secure: strings.HasPrefix(s.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode})
	respond(w, 200, Doc{"ok": true})
	return e
}
func (s *Server) seal(value string) (string, error) {
	key := sha256.Sum256([]byte(s.Config.SessionSecret))
	b, e := aes.NewCipher(key[:])
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(value), nil)), nil
}
func (s *Server) unseal(value string) (string, error) {
	raw, e := base64.RawURLEncoding.DecodeString(value)
	if e != nil {
		return "", e
	}
	key := sha256.Sum256([]byte(s.Config.SessionSecret))
	b, e := aes.NewCipher(key[:])
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil || len(raw) < g.NonceSize() {
		return "", errors.New("invalid encrypted credential")
	}
	v, e := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	return string(v), e
}
func (s *Server) listResponse(w http.ResponseWriter, r *http.Request, kind string) error {
	items, e := s.Store.List(r.Context(), kind)
	if e != nil {
		return e
	}
	respond(w, 200, Doc{"items": items, "total": len(items)})
	return nil
}
func requireRole(u Doc, role string) error {
	if !has(u["roles"], role) {
		return errStatus(403, "Insufficient permissions")
	}
	return nil
}
func normalizeError(e error) error {
	if e == nil {
		return nil
	}
	var a *apiError
	if errors.As(e, &a) {
		return e
	}
	return fmt.Errorf("storage operation: %w", e)
}
