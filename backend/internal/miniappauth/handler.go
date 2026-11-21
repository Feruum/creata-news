package miniappauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/telegram"
)

var opaquePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var verifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

type Handler struct {
	config   Config
	store    Store
	resolver Resolver
	signer   *signer
	origin   string
	now      func() time.Time
}

func NewHandler(config Config, store Store, resolver Resolver) (*Handler, error) {
	h := &Handler{config: config, store: store, resolver: resolver, now: time.Now}
	if config.PrivateKeyB64 == "" || config.ClientSecret == "" || config.BotToken == "" {
		return h, nil
	}
	if len(config.ClientSecret) < 32 {
		return nil, errors.New("Telegram Mini App client secret must contain at least 32 characters")
	}
	issuer, err := url.Parse(config.Issuer)
	if err != nil || !allowedURL(issuer) || issuer.RawQuery != "" || issuer.Fragment != "" || strings.HasSuffix(config.Issuer, "/") {
		return nil, errors.New("invalid Telegram Mini App issuer")
	}
	redirect, err := url.Parse(config.RedirectURI)
	if err != nil || !allowedURL(redirect) || redirect.RawQuery != "" || redirect.Fragment != "" {
		return nil, errors.New("invalid Telegram Mini App redirect URI")
	}
	if store == nil || resolver == nil {
		return nil, errors.New("Telegram Mini App storage and account resolver are required")
	}
	h.origin = issuer.Scheme + "://" + issuer.Host
	h.signer, err = newSigner(config.PrivateKeyB64)
	if err != nil {
		return nil, err
	}
	return h, nil
}
func allowedURL(u *url.URL) bool {
	if u == nil || u.Host == "" || u.User != nil {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))
}
func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func failure(w http.ResponseWriter, status int, code, message string) {
	reply(w, status, map[string]string{"error": code, "message": message})
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Vary", "Cookie, Origin")
	path := strings.TrimPrefix(r.URL.Path, "/telegram-miniapp/")
	if path == "status" && r.Method == "GET" {
		reply(w, 200, map[string]bool{"enabled": h.signer != nil})
		return
	}
	if h.signer == nil {
		failure(w, 503, "disabled", "Telegram automatic sign-in is not configured.")
		return
	}
	switch {
	case path == ".well-known/openid-configuration" && r.Method == "GET":
		w.Header().Set("Cache-Control", "public, max-age=300")
		reply(w, 200, map[string]any{"issuer": h.config.Issuer, "authorization_endpoint": h.config.Issuer + "/authorize", "token_endpoint": h.config.Issuer + "/token", "jwks_uri": h.config.Issuer + "/jwks", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}, "scopes_supported": []string{"openid"}, "token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"}, "code_challenge_methods_supported": []string{"S256"}})
	case path == "jwks" && r.Method == "GET":
		w.Header().Set("Cache-Control", "public, max-age=300")
		reply(w, 200, map[string]any{"keys": []map[string]string{h.signer.jwk}})
	case path == "prepare" && r.Method == "POST":
		h.prepare(w, r)
	case path == "proof" && r.Method == "GET":
		grant, err := h.readProof(r, false)
		if err != nil || h.resolver.Check(r.Context(), grant.Link) != nil {
			failure(w, 401, "expired", "Reopen the Mini App to sign in.")
			return
		}
		current, err := h.resolver.CurrentIdentity(r.Context(), r.Header.Get("Cookie"))
		if err != nil {
			failure(w, 503, "unavailable", "Could not verify your current session.")
			return
		}
		if current != "" && current != grant.Link.IdentityID {
			failure(w, 409, "account_conflict", "Telegram is linked to a different account. Sign out first.")
			return
		}
		reply(w, 200, map[string]bool{"valid": true, "authenticated": current == grant.Link.IdentityID})
	case path == "authorize" && r.Method == "GET":
		h.authorize(w, r)
	case path == "token" && r.Method == "POST":
		h.token(w, r)
	default:
		failure(w, 404, "not_found", "Not found.")
	}
}
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Origin") != h.origin {
		failure(w, 403, "forbidden", "This request is not allowed.")
		return
	}
	var body struct {
		InitData string `json:"initData"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		failure(w, 400, "invalid_request", "Invalid sign-in request.")
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		failure(w, 400, "invalid_request", "Invalid sign-in request.")
		return
	}
	identity, err := telegram.VerifyInitData(body.InitData, h.config.BotToken, h.now())
	if err != nil {
		failure(w, 401, "invalid_launch", "Telegram launch data is invalid or expired.")
		return
	}
	link, err := h.resolver.Resolve(r.Context(), identity.UserID)
	if errors.Is(err, ErrNotLinked) {
		failure(w, 404, "telegram_not_linked", "Sign in with Microsoft and link your Telegram account first.")
		return
	}
	if err != nil {
		failure(w, 503, "unavailable", "Could not verify your Telegram account.")
		return
	}
	current, err := h.resolver.CurrentIdentity(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		failure(w, 503, "unavailable", "Could not verify your current session.")
		return
	}
	if current != "" {
		if current != link.IdentityID {
			failure(w, 409, "account_conflict", "Telegram is linked to a different account. Sign out first.")
			return
		}
		reply(w, 200, map[string]any{"authenticated": true, "identityId": link.IdentityID})
		return
	}
	if err = h.resolver.EnsureCredential(r.Context(), *link); err != nil {
		failure(w, 503, "unavailable", "Could not prepare Telegram sign-in.")
		return
	}
	reserved, err := h.store.ReserveLaunch(r.Context(), identity.Fingerprint)
	if err != nil {
		failure(w, 503, "unavailable", "Could not prepare Telegram sign-in.")
		return
	}
	if !reserved {
		failure(w, 409, "launch_used", "Reopen the Mini App to start a new sign-in.")
		return
	}
	ticket, err := opaqueToken()
	if err != nil || h.store.Put(r.Context(), "launch", ticket, Grant{Link: *link, AuthenticatedAt: identity.AuthDate.Unix()}) != nil {
		failure(w, 503, "unavailable", "Could not prepare Telegram sign-in.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: launchCookie, Value: ticket, Path: "/api/auth/telegram", HttpOnly: true, Secure: strings.HasPrefix(h.origin, "https:"), SameSite: http.SameSiteLaxMode, MaxAge: 120})
	reply(w, 200, map[string]string{"startUrl": "/api/auth/telegram/start"})
}
func (h *Handler) readProof(r *http.Request, consume bool) (*Grant, error) {
	cookie, err := r.Cookie(launchCookie)
	if err != nil || !opaquePattern.MatchString(cookie.Value) {
		return nil, ErrExpired
	}
	if consume {
		return h.store.Consume(r.Context(), "launch", cookie.Value)
	}
	return h.store.Read(r.Context(), "launch", cookie.Value)
}
func unique(values url.Values) bool {
	for key, entries := range values {
		if key == "" || len(entries) != 1 {
			return false
		}
	}
	return true
}
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(r.URL.RawQuery) > 8192 || !unique(values) || values.Get("client_id") != ClientID || values.Get("redirect_uri") != h.config.RedirectURI || values.Get("response_type") != "code" || values.Get("scope") != "openid" || values.Get("state") == "" || len(values.Get("state")) > 512 || len(values.Get("nonce")) > 512 {
		failure(w, 400, "invalid_request", "Invalid authorization request.")
		return
	}
	challenge := values.Get("code_challenge")
	if (challenge != "" && (!opaquePattern.MatchString(challenge) || values.Get("code_challenge_method") != "S256")) || (challenge == "" && values.Get("code_challenge_method") != "") {
		failure(w, 400, "invalid_request", "Invalid PKCE challenge.")
		return
	}
	grant, err := h.readProof(r, true)
	if err != nil || h.resolver.Check(r.Context(), grant.Link) != nil {
		failure(w, 401, "expired", "Reopen the Mini App to sign in.")
		return
	}
	grant.Nonce = values.Get("nonce")
	current, err := h.resolver.CurrentIdentity(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		failure(w, 503, "unavailable", "Could not verify your current session.")
		return
	}
	if current != "" && current != grant.Link.IdentityID {
		failure(w, 409, "account_conflict", "Telegram is linked to a different account. Sign out first.")
		return
	}
	grant.CodeChallenge = challenge
	code, err := opaqueToken()
	if err != nil || h.store.Put(r.Context(), "code", code, *grant) != nil {
		failure(w, 503, "unavailable", "Could not authorize sign-in.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: launchCookie, Path: "/api/auth/telegram", HttpOnly: true, Secure: strings.HasPrefix(h.origin, "https:"), SameSite: http.SameSiteLaxMode, MaxAge: -1})
	redirect, _ := url.Parse(h.config.RedirectURI)
	query := url.Values{"code": {code}, "state": {values.Get("state")}}
	redirect.RawQuery = query.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
}
func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if r.ParseForm() != nil || !unique(r.PostForm) {
		failure(w, 400, "invalid_request", "Invalid token request.")
		return
	}
	id, secret, basic := r.BasicAuth()
	if !basic {
		id = r.PostForm.Get("client_id")
		secret = r.PostForm.Get("client_secret")
	}
	if id != ClientID || !hmac.Equal([]byte(secret), []byte(h.config.ClientSecret)) || (basic && r.PostForm.Get("client_id") != "" && r.PostForm.Get("client_id") != id) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Telegram Mini App"`)
		failure(w, 401, "invalid_client", "Client authentication required.")
		return
	}
	code := r.PostForm.Get("code")
	if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("redirect_uri") != h.config.RedirectURI || !opaquePattern.MatchString(code) {
		failure(w, 400, "invalid_grant", "Invalid authorization code.")
		return
	}
	grant, err := h.store.Consume(r.Context(), "code", code)
	if err != nil || h.resolver.Check(r.Context(), grant.Link) != nil {
		failure(w, 400, "invalid_grant", "Authorization code is expired or no longer valid.")
		return
	}
	if grant.CodeChallenge != "" {
		verifier := r.PostForm.Get("code_verifier")
		sum := sha256.Sum256([]byte(verifier))
		actual := base64.RawURLEncoding.EncodeToString(sum[:])
		if !verifierPattern.MatchString(verifier) || !hmac.Equal([]byte(actual), []byte(grant.CodeChallenge)) {
			failure(w, 400, "invalid_grant", "Invalid PKCE verifier.")
			return
		}
	}
	token, err := h.signer.sign(h.config, *grant, h.now())
	if err != nil {
		failure(w, 503, "unavailable", "Could not complete sign-in.")
		return
	}
	access, err := opaqueToken()
	if err != nil {
		failure(w, 503, "unavailable", "Could not complete sign-in.")
		return
	}
	reply(w, 200, map[string]any{"id_token": token, "access_token": access, "token_type": "Bearer", "expires_in": 120, "scope": "openid"})
}
