package sundayaccount

import (
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const telegramNonceCookie = "sundayopen_telegram_nonce"
const telegramIssuer = "https://oauth.telegram.org"

var errTelegramIDToken = errors.New("invalid Telegram ID token")
var errTelegramKeys = errors.New("Telegram signing keys unavailable")

// The public OIDC client ID is the numeric bot ID, never its API token.
func (h *Handler) telegramClientID() string {
	if h.widgetBotUsername() == "" {
		return ""
	}
	id, _, ok := strings.Cut(h.cfg.BotToken, ":")
	n, err := strconv.ParseInt(id, 10, 64)
	if !ok || err != nil || n <= 0 || n > 1<<53-1 || strconv.FormatInt(n, 10) != id {
		return ""
	}
	return id
}

func (h *Handler) telegramChallenge(w http.ResponseWriter, r *http.Request) {
	id := h.telegramClientID()
	if id == "" {
		fail(w, 503, "Telegram sign-in is not configured")
		return
	}
	if !h.loginLimiter.allow(h.digest("telegram-challenge-ip", h.requestIP(r)), time.Now()) {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "Please wait before trying Telegram sign-in again")
		return
	}
	p, err := h.authenticate(r)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 503, "Account service unavailable")
		return
	}
	if p != nil && !hmac.Equal([]byte(r.Header.Get("X-CSRF-Token")), []byte(h.csrf(p.ID, p.token))) {
		fail(w, 403, "Invalid CSRF token")
		return
	}
	nonce, err := randomToken()
	if err != nil {
		fail(w, 503, "Could not start Telegram sign-in")
		return
	}
	value := strconv.FormatInt(time.Now().Unix(), 10) + "." + nonce
	h.setTelegramNonce(w, nonce, value+"."+h.digest("telegram-oidc-nonce", value), 300)
	writeJSON(w, 200, map[string]string{"clientId": id, "nonce": nonce})
}

func (h *Handler) setTelegramNonce(w http.ResponseWriter, nonce, value string, age int) {
	http.SetCookie(w, &http.Cookie{Name: telegramNonceCookie + "_" + nonce, Value: value,
		Path: prefix, HttpOnly: true, Secure: h.cookieSecure(), SameSite: http.SameSiteLaxMode, MaxAge: age})
}

func (h *Handler) telegramBrowserNonce(r *http.Request, nonce string, now time.Time) string {
	if len(nonce) != 43 {
		return ""
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(nonce); err != nil || len(decoded) != 32 {
		return ""
	}
	c, err := r.Cookie(telegramNonceCookie + "_" + nonce)
	if err != nil || len(c.Value) > 128 {
		return ""
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 || len(parts[1]) != 43 {
		return ""
	}
	if !hmac.Equal([]byte(parts[1]), []byte(nonce)) {
		return ""
	}
	at, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || at <= now.Add(-5*time.Minute).Unix() || at > now.Add(30*time.Second).Unix() {
		return ""
	}
	if !hmac.Equal([]byte(parts[2]), []byte(h.digest("telegram-oidc-nonce", parts[0]+"."+parts[1]))) {
		return ""
	}
	return parts[1]
}

type telegramSigningKeys struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
	expires time.Time
}

// Fetch only Telegram's fixed HTTPS JWKS URL. Token-controlled jku/x5u URLs are
// never followed. Unknown kids can trigger at most one refresh per minute.
func (h *Handler) telegramSigningKey(ctx context.Context, kid string, now time.Time) (*rsa.PublicKey, error) {
	cache := &h.telegramKeys
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if now.Before(cache.expires) {
		if key := cache.keys[kid]; key != nil {
			return key, nil
		}
	}
	if now.Before(cache.fetched.Add(time.Minute)) {
		if now.Before(cache.expires) {
			return nil, errTelegramIDToken
		}
		return nil, errTelegramKeys
	}
	cache.fetched = now
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, telegramIssuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, errTelegramKeys
	}
	// Copying the client retains the transport but prevents HTTP redirects.
	client := *h.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return nil, errTelegramKeys
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errTelegramKeys
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(body) > 65536 {
		return nil, errTelegramKeys
	}
	var jwks struct {
		Keys []struct{ Kid, Kty, Alg, Use, N, E string } `json:"keys"`
	}
	if json.Unmarshal(body, &jwks) != nil || len(jwks.Keys) == 0 || len(jwks.Keys) > 64 {
		return nil, errTelegramKeys
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" || (jwk.Alg != "" && jwk.Alg != "RS256") || (jwk.Use != "" && jwk.Use != "sig") || jwk.Kid == "" || len(jwk.Kid) > 128 {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(jwk.N)
		e, errE := base64.RawURLEncoding.DecodeString(jwk.E)
		if errN != nil || errE != nil || len(n) < 256 || len(n) > 512 || len(e) == 0 || len(e) > 4 {
			continue
		}
		modulus := new(big.Int).SetBytes(n)
		exponent := new(big.Int).SetBytes(e).Int64()
		if modulus.BitLen() < 2048 || exponent < 3 || exponent > 1<<31-1 || exponent%2 == 0 {
			continue
		}
		if keys[jwk.Kid] != nil {
			return nil, errTelegramKeys
		}
		keys[jwk.Kid] = &rsa.PublicKey{N: modulus, E: int(exponent)}
	}
	if len(keys) == 0 {
		return nil, errTelegramKeys
	}
	cache.keys, cache.expires = keys, now.Add(time.Hour)
	if key := keys[kid]; key != nil {
		return key, nil
	}
	return nil, errTelegramIDToken
}

type telegramIDClaims struct {
	jwt.RegisteredClaims
	ID       json.Number `json:"id"`
	Name     string      `json:"name"`
	Username string      `json:"preferred_username"`
	Nonce    string      `json:"nonce"`
}

func (h *Handler) verifyTelegramIDToken(ctx context.Context, raw, nonce string, now time.Time) (telegramWidgetData, error) {
	var identity telegramWidgetData
	if len(raw) == 0 || len(raw) > 8192 || len(nonce) != 43 || h.telegramClientID() == "" {
		return identity, errTelegramIDToken
	}
	claims := new(telegramIDClaims)
	// Validate every required claim explicitly below with one consistent clock.
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}), jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 128 {
			return nil, errTelegramIDToken
		}
		return h.telegramSigningKey(ctx, kid, now)
	})
	if err != nil {
		if errors.Is(err, errTelegramKeys) {
			return identity, errTelegramKeys
		}
		return identity, errTelegramIDToken
	}
	id, err := claims.ID.Int64()
	if err != nil || !token.Valid || claims.Issuer != telegramIssuer || len(claims.Audience) != 1 || claims.Audience[0] != h.telegramClientID() ||
		claims.Subject == "" || len(claims.Subject) > 256 || claims.ExpiresAt == nil || claims.IssuedAt == nil ||
		!now.Before(claims.ExpiresAt.Time) || claims.ExpiresAt.Before(claims.IssuedAt.Time) ||
		claims.IssuedAt.Before(now.Add(-10*time.Minute)) || claims.IssuedAt.After(now.Add(30*time.Second)) ||
		(claims.NotBefore != nil && claims.NotBefore.After(now.Add(30*time.Second))) ||
		!hmac.Equal([]byte(claims.Nonce), []byte(nonce)) || id <= 0 || id > 1<<53-1 ||
		strings.TrimSpace(claims.Name) == "" || !validTelegramText(claims.Name, 256) || !telegramUsernameValid(claims.Username) {
		return identity, errTelegramIDToken
	}
	identity.ID, identity.FirstName, identity.Username = id, claims.Name, &claims.Username
	return identity, nil
}
