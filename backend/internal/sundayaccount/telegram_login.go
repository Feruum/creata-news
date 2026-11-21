package sundayaccount

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun"
)

type telegramWidgetData struct {
	ID        int64   `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  *string `json:"last_name"`
	Username  *string `json:"username"`
	PhotoURL  *string `json:"photo_url"`
	AuthDate  int64   `json:"auth_date"`
	Hash      string  `json:"hash"`
}

// The browser sends Telegram's original fields unchanged. Contact details are
// used only to create a new participant, never to select an existing identity.
type telegramLoginRequest struct {
	IDToken  string             `json:"idToken"`
	Nonce    string             `json:"nonce"`
	Telegram telegramWidgetData `json:"telegram"`
	FullName string             `json:"fullName"`
	Email    string             `json:"email"`
	// Legacy clients send an empty phone field; nonempty values are rejected.
	Phone string `json:"phone"`
}

func telegramUsernameValid(value string) bool {
	if len(value) > 32 {
		return false
	}
	for _, char := range value {
		if char != '_' && (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func (h *Handler) widgetBotUsername() string {
	username := strings.TrimPrefix(strings.TrimSpace(h.cfg.BotUsername), "@")
	if strings.TrimSpace(h.cfg.BotToken) == "" || len(username) < 5 || !telegramUsernameValid(username) || !strings.HasSuffix(strings.ToLower(username), "bot") {
		return ""
	}
	return username
}

func validTelegramText(value string, maximum int) bool {
	return len(value) <= maximum && utf8.ValidString(value) && !strings.ContainsAny(value, "\r\n\x00")
}

// Telegram's legacy Login Widget signs all supplied fields except hash with
// HMAC-SHA256 using SHA256(bot token) as its key. Optional field presence matters.
// https://core.telegram.org/widgets/login-legacy#checking-authorization
func (h *Handler) verifyTelegramWidget(data telegramWidgetData, now time.Time) bool {
	if data.ID <= 0 || data.ID > 1<<53-1 || strings.TrimSpace(data.FirstName) == "" || !validTelegramText(data.FirstName, 256) {
		return false
	}
	if data.AuthDate < now.Add(-10*time.Minute).Unix() || data.AuthDate > now.Add(30*time.Second).Unix() {
		return false
	}
	fields := []string{"auth_date=" + strconv.FormatInt(data.AuthDate, 10), "first_name=" + data.FirstName, "id=" + strconv.FormatInt(data.ID, 10)}
	if data.LastName != nil {
		if !validTelegramText(*data.LastName, 256) {
			return false
		}
		fields = append(fields, "last_name="+*data.LastName)
	}
	if data.PhotoURL != nil {
		if !validTelegramText(*data.PhotoURL, 2048) {
			return false
		}
		if *data.PhotoURL != "" {
			photo, err := url.Parse(*data.PhotoURL)
			if err != nil || photo.Scheme != "https" || photo.Hostname() == "" || photo.User != nil || photo.Fragment != "" {
				return false
			}
		}
		fields = append(fields, "photo_url="+*data.PhotoURL)
	}
	if data.Username != nil {
		if !telegramUsernameValid(*data.Username) {
			return false
		}
		fields = append(fields, "username="+*data.Username)
	}
	if len(data.Hash) != 64 {
		return false
	}
	received, err := hex.DecodeString(data.Hash)
	if err != nil {
		return false
	}
	secret := sha256.Sum256([]byte(h.cfg.BotToken))
	signature := hmac.New(sha256.New, secret[:])
	_, _ = signature.Write([]byte(strings.Join(fields, "\n")))
	return hmac.Equal(received, signature.Sum(nil))
}

type telegramLoginWindow struct {
	started time.Time
	count   int
}

// Rate-limit invalid proofs before database access, with bounded memory and no
// raw IP storage. Each backend replica maintains its own one-minute windows.
type telegramLoginLimiter struct {
	mu      sync.Mutex
	windows map[string]telegramLoginWindow
	cleanup time.Time
}

func (limit *telegramLoginLimiter) allow(key string, now time.Time) bool {
	limit.mu.Lock()
	defer limit.mu.Unlock()
	if limit.windows == nil {
		limit.windows = make(map[string]telegramLoginWindow)
	}
	if !now.Before(limit.cleanup) || len(limit.windows) >= 4096 {
		for key, window := range limit.windows {
			if !now.Before(window.started.Add(time.Minute)) {
				delete(limit.windows, key)
			}
		}
		limit.cleanup = now.Add(time.Minute)
	}
	window, exists := limit.windows[key]
	if !exists || !now.Before(window.started.Add(time.Minute)) {
		if !exists && len(limit.windows) >= 4096 {
			return false
		}
		window = telegramLoginWindow{started: now}
	}
	if window.count >= 30 {
		return false
	}
	window.count++
	limit.windows[key] = window
	return true
}

var errTelegramOwned = errors.New("Telegram account already linked")

type telegramProfileError struct{ message string }

func (err *telegramProfileError) Error() string { return err.message }

func (h *Handler) telegramLogin(w http.ResponseWriter, r *http.Request) {
	if h.widgetBotUsername() == "" {
		fail(w, 503, "Telegram sign-in is not configured")
		return
	}
	if !h.loginLimiter.allow(h.digest("telegram-login-ip", h.requestIP(r)), time.Now()) {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "Please wait before trying Telegram sign-in again")
		return
	}
	var input telegramLoginRequest
	if !decode(w, r, &input) {
		return
	}
	if input.Phone != "" {
		fail(w, 400, "Invalid request")
		return
	}
	if input.IDToken != "" {
		if input.Telegram.ID != 0 || input.Telegram.Hash != "" {
			fail(w, 400, "Invalid request")
			return
		}
		identity, err := h.verifyTelegramIDToken(r.Context(), input.IDToken, h.telegramBrowserNonce(r, input.Nonce, time.Now()), time.Now())
		if err != nil {
			if errors.Is(err, errTelegramKeys) {
				fail(w, 503, "Telegram sign-in is temporarily unavailable")
			} else {
				fail(w, 401, "Telegram authorization is invalid or expired")
			}
			return
		}
		input.Telegram = identity
		h.setTelegramNonce(w, input.Nonce, "", -1)
	} else if !h.verifyTelegramWidget(input.Telegram, time.Now()) {
		fail(w, 401, "Telegram authorization is invalid or expired")
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
	token, err := randomToken()
	if err != nil {
		fail(w, 503, "Could not sign in with Telegram")
		return
	}
	var account Account
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		var linked struct {
			ID              string
			TelegramUserID  *int64
			TelegramBlocked bool
		}
		if p != nil {
			// Account-before-identity lock ordering matches webhook confirmation.
			if _, err := h.lockSessionAccount(ctx, tx, p); err != nil {
				return err
			}
			if err := tx.NewRaw(`SELECT id,telegram_user_id,telegram_blocked FROM sunday_open_accounts WHERE id=?`, p.ID).Scan(ctx, &linked); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(?)`, input.Telegram.ID); err != nil {
				return err
			}
			var owner string
			err := tx.NewRaw(`SELECT id FROM sunday_open_accounts WHERE telegram_user_id=?`, input.Telegram.ID).Scan(ctx, &owner)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if err == nil && owner != p.ID {
				return errTelegramOwned
			}
		} else {
			// Only a verified Telegram ID can recover a participant. Typed email
			// and phone are never searched and existing contacts never change.
			err := tx.NewRaw(`SELECT id,telegram_user_id,telegram_blocked FROM sunday_open_accounts WHERE telegram_user_id=? FOR NO KEY UPDATE`, input.Telegram.ID).Scan(ctx, &linked)
			if errors.Is(err, sql.ErrNoRows) {
				name := strings.TrimSpace(input.FullName)
				if !validName(name) {
					return &telegramProfileError{"Enter your full name"}
				}
				email, err := normalizeEmail(input.Email)
				if err != nil {
					return &telegramProfileError{err.Error()}
				}
				// A concurrent first login may create this same verified identity.
				// The unique key chooses its existing row without replacing contacts.
				err = tx.NewRaw(`INSERT INTO sunday_open_accounts(contact_email,full_name,telegram_user_id) VALUES(?,?,?) ON CONFLICT(telegram_user_id) DO NOTHING RETURNING id,telegram_user_id,telegram_blocked`, email, name, input.Telegram.ID).Scan(ctx, &linked)
				if errors.Is(err, sql.ErrNoRows) {
					err = tx.NewRaw(`SELECT id,telegram_user_id,telegram_blocked FROM sunday_open_accounts WHERE telegram_user_id=? FOR NO KEY UPDATE`, input.Telegram.ID).Scan(ctx, &linked)
				}
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
		activate := linked.TelegramUserID == nil || *linked.TelegramUserID != input.Telegram.ID || linked.TelegramBlocked
		username := ""
		if input.Telegram.Username != nil {
			username = *input.Telegram.Username
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_telegram_links SET consumed_at=NOW() WHERE account_id=? AND consumed_at IS NULL`, linked.ID); err != nil {
			return err
		}
		if activate {
			// A widget with write access enables future notifications, matching
			// confirmed deep links without replaying pre-link history.
			if _, err := tx.ExecContext(ctx, `UPDATE external_registrations xr SET snapshot=`+snapshotSQL+` FROM events e WHERE xr.event_id=e.id AND xr.sunday_open_account_id=? AND e.status='published' AND e.publish_on_sunday_open`, linked.ID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),locked_until=NULL,last_error='before link activation' WHERE account_id=? AND sent_at IS NULL`, linked.ID); err != nil {
				return err
			}
		}
		// Updating a unique Telegram key strengthens the account row lock. Do it
		// after registration snapshots so the planner can finish its account FK
		// checks while the widget is waiting for a registration row.
		if err := tx.NewRaw(`UPDATE sunday_open_accounts SET telegram_user_id=?,telegram_chat_id=?,telegram_username=NULLIF(?,''),telegram_blocked=false,updated_at=NOW() WHERE id=? RETURNING `+accountProjection(""), input.Telegram.ID, input.Telegram.ID, username, linked.ID).Scan(ctx, &account); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO sunday_open_sessions(account_id,token_hash,expires_at) VALUES(?,?,NOW()+INTERVAL '30 days')`, account.ID, h.digest("session", token)); err != nil {
			return err
		}
		if prior, err := r.Cookie(cookieName); err == nil {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sunday_open_sessions WHERE token_hash=?`, h.digest("session", prior.Value)); err != nil {
				return err
			}
		}
		return nil
	})
	var profileError *telegramProfileError
	switch {
	case errors.Is(err, errInvalidCredentials):
		fail(w, 401, "Sign in to continue")
	case errors.Is(err, errTelegramOwned):
		fail(w, 409, "This Telegram account is already connected to another participant")
	case errors.As(err, &profileError):
		fail(w, 400, profileError.message)
	case err != nil:
		fail(w, 503, "Could not sign in with Telegram")
	default:
		h.setCookie(w, token, 30*24*3600)
		h.writeSession(w, &principal{Account: account, token: token})
	}
}
