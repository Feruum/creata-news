// Package sundayaccount owns Sunday Open identities independently of campus SSO.
package sundayaccount

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/uptrace/bun"
)

const prefix = "/api/sundayopen/account/"
const cookieName = "sundayopen_session"

type Config struct {
	Origin             string
	MapAppURL          string
	Secret             string
	SMTPURL            string
	SMTPFrom           string
	BotToken           string
	BotUsername        string
	BotConfigEncrypted string
	WebhookSecret      string
	TrustedProxyCIDRs  string
}
type Mailer interface {
	SendCode(context.Context, string, string) error
}
type Handler struct {
	db            *bun.DB
	cfg           Config
	mailer        Mailer
	client        *http.Client
	plannerCursor int32
	organizer     organizerService
	loginLimiter  telegramLoginLimiter
	telegramKeys  telegramSigningKeys
	forms         *registrationforms.Service
}
type Account struct {
	ID               string  `json:"id"`
	FullName         string  `json:"fullName"`
	Email            string  `json:"email"`
	Phone            string  `json:"phone"`
	IsGuest          bool    `json:"isGuest"`
	TelegramUsername *string `json:"telegramUsername,omitempty"`
	TelegramLinked   bool    `json:"telegramLinked"`
}
type principal struct {
	Account
	token string
}

func New(db *bun.DB, cfg Config, mailer Mailer) *Handler {
	cfg = resolveTelegramConfig(cfg)
	if mailer == nil {
		mailer = &SMTPMailer{URL: cfg.SMTPURL, From: cfg.SMTPFrom}
	}
	return &Handler{db: db, cfg: cfg, mailer: mailer, client: &http.Client{Timeout: 15 * time.Second}}
}
func (h *Handler) WithForms(forms *registrationforms.Service) *Handler {
	h.forms = forms
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if len(h.cfg.Secret) < 32 {
		fail(w, 503, "Account sign-in is not configured")
		return
	}
	if r.URL.Path == "/api/sundayopen/telegram/webhook" {
		h.webhook(w, r)
		return
	}
	if (r.Method != "GET" || r.Header.Get("Origin") != "") && !h.originAllowed(r) {
		fail(w, 403, "Origin is not allowed")
		return
	}
	route := strings.TrimPrefix(r.URL.Path, prefix)
	if route == "telegram-setup-key" {
		h.telegramSetupKey(w, r)
		return
	}
	if route == "telegram-config" && r.Method == "GET" {
		writeJSON(w, 200, map[string]string{"botUsername": h.widgetBotUsername(), "clientId": h.telegramClientID()})
		return
	}
	if route == "telegram-challenge" && r.Method == "POST" {
		h.telegramChallenge(w, r)
		return
	}
	if route == "telegram-login" && r.Method == "POST" {
		h.telegramLogin(w, r)
		return
	}
	if route == "request-code" && r.Method == "POST" {
		h.requestCode(w, r)
		return
	}
	if route == "verify-code" && r.Method == "POST" {
		h.verifyCode(w, r)
		return
	}
	if route == "check-in/start" && r.Method == "GET" {
		h.startCheckIn(w, r)
		return
	}
	if route == "check-in/info" && r.Method == "GET" {
		h.checkInInfo(w, r)
		return
	}
	if route == "organizer/check-in" && r.Method == "GET" {
		h.organizerCheckIn(w, r)
		return
	}
	p, err := h.authenticate(r)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 503, "Account service unavailable")
		return
	}
	if route == "start" && r.Method == "POST" {
		if p != nil {
			h.writeSession(w, p)
			return
		}
		h.startGuest(w, r)
		return
	}
	if route == "session" && r.Method == "GET" {
		if p == nil {
			writeJSON(w, 200, map[string]any{"account": nil})
			return
		}
		h.writeSession(w, p)
		return
	}
	if p == nil {
		fail(w, 401, "Sign in to continue")
		return
	}
	if r.Method != "GET" && !hmac.Equal([]byte(r.Header.Get("X-CSRF-Token")), []byte(h.csrf(p.ID, p.token))) {
		fail(w, 403, "Invalid CSRF token")
		return
	}
	switch {
	case route == "forms/uploads" && r.Method == "POST":
		h.beginFormUpload(w, r, p)
	case strings.HasPrefix(route, "forms/uploads/"):
		h.formUpload(w, r, p, strings.TrimPrefix(route, "forms/uploads/"))
	case route == "onboarding" && r.Method == "GET":
		h.getOnboarding(w, r, p)
	case route == "onboarding" && r.Method == "PATCH":
		h.saveOnboarding(w, r, p)
	case route == "onboarding/prompted" && r.Method == "POST":
		h.promptOnboarding(w, r, p)
	case route == "check-in" && r.Method == "POST":
		h.completeCheckIn(w, r, p)
	case route == "logout" && r.Method == "POST":
		if _, err := h.db.ExecContext(r.Context(), `DELETE FROM sunday_open_sessions WHERE token_hash=?`, h.digest("session", p.token)); err != nil {
			fail(w, 503, "Could not sign out")
			return
		}
		h.setCookie(w, "", -1)
		success(w)
	case route == "profile" && r.Method == "PATCH":
		h.profile(w, r, p)
	case route == "registrations" && r.Method == "GET":
		h.history(w, r, p)
	case route == "registrations" && r.Method == "POST":
		h.register(w, r, p)
	case strings.HasPrefix(route, "registrations/") && r.Method == "DELETE":
		h.cancel(w, r, p, strings.TrimPrefix(route, "registrations/"))
	case route == "telegram-link" && r.Method == "POST":
		h.createTelegramLink(w, r, p)
	case route == "telegram-link" && r.Method == "DELETE":
		err := h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_accounts SET telegram_user_id=NULL,telegram_chat_id=NULL,telegram_username=NULL,telegram_blocked=false,updated_at=NOW() WHERE id=?`, p.ID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_telegram_links SET consumed_at=NOW() WHERE account_id=? AND consumed_at IS NULL`, p.ID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),last_error='unlinked' WHERE account_id=? AND sent_at IS NULL`, p.ID)
			return err
		})
		if err != nil {
			fail(w, 503, "Could not unlink Telegram")
			return
		}
		success(w)
	default:
		fail(w, 404, "Not found")
	}
}
func (h *Handler) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	for _, allowed := range strings.Split(h.cfg.Origin, ",") {
		if origin != "" && origin == strings.TrimSpace(allowed) {
			return true
		}
	}
	return false
}
func (h *Handler) digest(domain, value string) string {
	m := hmac.New(sha256.New, []byte(h.cfg.Secret))
	m.Write([]byte(domain + "\x00" + value))
	return hex.EncodeToString(m.Sum(nil))
}
func (h *Handler) csrf(id, token string) string { return h.digest("csrf", id+"\x00"+token) }
func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
func normalizeEmail(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	a, err := mail.ParseAddress(value)
	if err != nil || a.Address != value || len(value) > 254 || strings.ContainsAny(raw, "\r\n") {
		return "", errors.New("Enter a valid email address")
	}
	return value, nil
}
func validName(name string) bool {
	return len(name) > 0 && len(name) <= 200 && !strings.ContainsAny(name, "\r\n\x00")
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	return decodeLimit(w, r, target, 16384)
}
func decodeLimit(w http.ResponseWriter, r *http.Request, target any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(new(any)) != io.EOF {
		fail(w, 400, "Invalid request")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	code := map[int]string{400: "invalid_argument", 401: "unauthenticated", 403: "permission_denied", 404: "not_found", 405: "unimplemented", 409: "conflict", 429: "resource_exhausted", 503: "unavailable"}[status]
	if code == "" {
		code = "internal"
	}
	failCode(w, status, code, message)
}
func failCode(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message, "error": message})
}
func success(w http.ResponseWriter) { writeJSON(w, 200, map[string]bool{"success": true}) }
func (h *Handler) setCookie(w http.ResponseWriter, token string, age int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/api/sundayopen", HttpOnly: true, Secure: h.cookieSecure(), SameSite: http.SameSiteLaxMode, MaxAge: age})
}
func (h *Handler) cookieSecure() bool {
	secure := false
	// Only an entirely loopback HTTP configuration may disable Secure for development.
	// A production origin must not be downgraded by the order of the allowlist.
	for _, origin := range strings.Split(h.cfg.Origin, ",") {
		u, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || u.Scheme != "http" || (u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback()) {
			secure = true
			break
		}
	}
	return secure
}
func (h *Handler) authenticate(r *http.Request) (*principal, error) {
	c, err := r.Cookie(cookieName)
	if err != nil || len(c.Value) != 43 {
		return nil, sql.ErrNoRows
	}
	var p principal
	err = h.db.NewRaw(`SELECT `+accountProjection("a")+` FROM sunday_open_accounts a JOIN sunday_open_sessions s ON s.account_id=a.id WHERE s.token_hash=? AND s.expires_at>NOW()`, h.digest("session", c.Value)).Scan(r.Context(), &p.Account)
	if err != nil {
		return nil, err
	}
	p.token = c.Value
	return &p, nil
}
func (h *Handler) writeSession(w http.ResponseWriter, p *principal) {
	writeJSON(w, 200, map[string]any{"account": p.Account, "csrfToken": h.csrf(p.ID, p.token)})
}
func (h *Handler) requestCode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		FullName string `json:"fullName"`
	}
	if !decode(w, r, &input) {
		return
	}
	email, err := normalizeEmail(input.Email)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if input.FullName != "" && !validName(input.FullName) {
		fail(w, 400, "Enter a valid full name")
		return
	}
	ip := h.requestIP(r)
	ipHash := h.digest("ip", ip)
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		fail(w, 503, "Could not create code")
		return
	}
	code := strings.Repeat("0", 6-len(n.String())) + n.String()
	var challengeID string
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		// IP first, then email: serializes limits across backend replicas.
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "sunday-ip:"+ipHash); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "sunday-email:"+email); err != nil {
			return err
		}
		var legacyIdentity bool
		if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM sunday_open_accounts WHERE login_email=? AND email IS NULL)`, email).Scan(ctx, &legacyIdentity); err != nil {
			return err
		}
		if legacyIdentity {
			return errLegacyIdentity
		}
		var counts struct {
			Cooldown   int
			EmailCount int
			IPCount    int
		}
		if err := tx.NewRaw(`SELECT count(*) FILTER(WHERE email=? AND created_at>NOW()-INTERVAL '60 seconds') AS cooldown,count(*) FILTER(WHERE email=?) AS email_count,count(*) FILTER(WHERE ip_hash=?) AS ip_count FROM sunday_open_auth_challenges WHERE created_at>NOW()-INTERVAL '1 hour'`, email, email, ipHash).Scan(ctx, &counts); err != nil {
			return err
		}
		if counts.Cooldown > 0 || counts.EmailCount >= 6 || counts.IPCount >= 30 {
			return errRateLimited
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_auth_challenges SET consumed_at=NOW() WHERE email=? AND consumed_at IS NULL`, email); err != nil {
			return err
		}
		return tx.NewRaw(`INSERT INTO sunday_open_auth_challenges(email,full_name,code_hash,ip_hash,expires_at) VALUES(?,?,?, ?,NOW()+INTERVAL '10 minutes') RETURNING id`, email, input.FullName, h.digest("code", email+"\x00"+code), ipHash).Scan(ctx, &challengeID)
	})
	if errors.Is(err, errLegacyIdentity) {
		failCode(w, 409, "failed_precondition", "Sign in with the Telegram account connected to this participant")
		return
	}
	if errors.Is(err, errRateLimited) {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "Please wait before requesting another code")
		return
	}
	if err != nil {
		fail(w, 503, "Could not request sign-in code")
		return
	}
	if err = h.mailer.SendCode(r.Context(), email, code); err != nil { // Fail closed: an undelivered code is not usable.
		_, _ = h.db.ExecContext(r.Context(), `UPDATE sunday_open_auth_challenges SET consumed_at=NOW() WHERE id=?`, challengeID)
		slog.Warn("Sunday Open code delivery failed")
		fail(w, 503, "Email delivery is unavailable. Please try again later")
		return
	}
	writeJSON(w, 200, map[string]any{"challengeId": challengeID, "expiresIn": 600, "retryAfter": 60})
}

var errRateLimited = errors.New("rate limited")
var errLegacyIdentity = errors.New("use the existing Telegram identity")

// Each configured proxy may assert only its immediate predecessor. Walk from
// the socket peer towards the client, stopping at the first untrusted or invalid
// hop so caller-supplied leading entries cannot bypass rate limiting.
func (h *Handler) requestIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	var networks []*net.IPNet
	for _, raw := range strings.Split(h.cfg.TrustedProxyCIDRs, ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err == nil {
			networks = append(networks, network)
		}
	}
	forwarded := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		trusted := false
		for _, network := range networks {
			if network.Contains(peer) {
				trusted = true
				break
			}
		}
		if !trusted {
			break
		}
		client := net.ParseIP(strings.TrimSpace(forwarded[i]))
		if client == nil {
			break
		}
		peer = client
	}
	if peer != nil {
		return peer.String()
	}
	return "unknown"
}

var errInvalidCode = errors.New("invalid code")

func (h *Handler) verifyCode(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ChallengeID string `json:"challengeId"`
		Code        string `json:"code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.ChallengeID) != 36 || len(input.Code) != 6 || strings.Trim(input.Code, "0123456789") != "" {
		fail(w, 400, "Invalid or expired code")
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 503, "Could not sign in")
		return
	}
	var account Account
	valid := false
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		var challengeEmail string
		if err := tx.NewRaw(`SELECT email FROM sunday_open_auth_challenges WHERE id=?`, input.ChallengeID).Scan(ctx, &challengeEmail); err != nil {
			return err
		}
		if err := lockEmail(ctx, tx, challengeEmail); err != nil {
			return err
		}
		var c struct {
			Email, FullName, CodeHash string
			Attempts                  int
			ExpiresAt                 time.Time
			ConsumedAt                *time.Time
		}
		if err := tx.NewRaw(`SELECT email,full_name,code_hash,attempts,expires_at,consumed_at FROM sunday_open_auth_challenges WHERE id=? FOR UPDATE`, input.ChallengeID).Scan(ctx, &c); err != nil {
			return err
		}
		if c.ConsumedAt != nil || c.Attempts >= 5 || !c.ExpiresAt.After(time.Now()) {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_auth_challenges SET attempts=attempts+1 WHERE id=?`, input.ChallengeID); err != nil {
			return err
		}
		if !hmac.Equal([]byte(c.CodeHash), []byte(h.digest("code", c.Email+"\x00"+input.Code))) {
			return nil
		}
		var legacyIdentity bool
		if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM sunday_open_accounts WHERE login_email=? AND email IS NULL)`, c.Email).Scan(ctx, &legacyIdentity); err != nil {
			return err
		}
		if legacyIdentity {
			_, err := tx.ExecContext(ctx, `UPDATE sunday_open_auth_challenges SET consumed_at=NOW() WHERE id=?`, input.ChallengeID)
			return err
		}
		// Existing account names can only change through the authenticated profile route.
		err := tx.NewRaw(`SELECT `+accountProjection("")+` FROM sunday_open_accounts WHERE email=?`, c.Email).Scan(ctx, &account)
		if errors.Is(err, sql.ErrNoRows) {
			if !validName(c.FullName) {
				return errNameRequired
			}
			err = tx.NewRaw(`INSERT INTO sunday_open_accounts(email,full_name) VALUES(?,?) ON CONFLICT(email) DO UPDATE SET email=EXCLUDED.email RETURNING `+accountProjection(""), c.Email, c.FullName).Scan(ctx, &account)
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sunday_open_auth_challenges SET consumed_at=NOW() WHERE id=?`, input.ChallengeID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO sunday_open_sessions(account_id,token_hash,expires_at) VALUES(?,?,NOW()+INTERVAL '30 days')`, account.ID, h.digest("session", token)); err != nil {
			return err
		}
		// Rotate any prior browser session upon sign in.
		if c, err := r.Cookie(cookieName); err == nil {
			if _, err = tx.ExecContext(ctx, `DELETE FROM sunday_open_sessions WHERE token_hash=?`, h.digest("session", c.Value)); err != nil {
				return err
			}
		}
		valid = true
		return nil
	})
	if errors.Is(err, errNameRequired) {
		fail(w, 400, "Enter your full name when requesting a code")
		return
	}
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !valid) {
		fail(w, 400, "Invalid or expired code")
		return
	}
	if err != nil {
		fail(w, 503, "Could not sign in")
		return
	}
	h.setCookie(w, token, 30*24*3600)
	h.writeSession(w, &principal{Account: account, token: token})
}

var errNameRequired = errors.New("full name required")

func (h *Handler) profile(w http.ResponseWriter, r *http.Request, p *principal) {
	var input struct {
		FullName string  `json:"fullName"`
		Email    *string `json:"email"`
		Phone    *string `json:"phone"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if !validName(input.FullName) {
		fail(w, 400, "Enter your full name")
		return
	}
	email := p.Email
	if input.Email != nil {
		var err error
		email, err = normalizeEmail(*input.Email)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		if !p.IsGuest && email != p.Email {
			fail(w, 400, "Verified email cannot be changed")
			return
		}
	}
	phone := p.Phone
	if input.Phone != nil {
		var err error
		phone, err = normalizePhone(*input.Phone)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
	}
	var updated Account
	if err := h.db.NewRaw(`UPDATE sunday_open_accounts SET full_name=?,contact_email=CASE WHEN ? AND email IS NULL AND password_hash IS NULL THEN ? ELSE contact_email END,phone=CASE WHEN ? THEN NULLIF(?,'') ELSE phone END,updated_at=NOW() WHERE id=? RETURNING `+accountProjection(""), input.FullName, input.Email != nil, email, input.Phone != nil, phone, p.ID).Scan(r.Context(), &updated); err != nil {
		fail(w, 503, "Could not save profile")
		return
	}
	p.Account = updated
	h.writeSession(w, p)
}
