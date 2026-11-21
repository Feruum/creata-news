package sundayaccount

import (
	"bytes"
	"context"
	"crypto/hmac"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type telegramResponse struct {
	OK         bool `json:"ok"`
	ErrorCode  int  `json:"error_code"`
	Parameters struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}
type telegramError struct{ code, retryAfter int }

func (e *telegramError) Error() string { return fmt.Sprintf("telegram status %d", e.code) }
func (h *Handler) telegram(ctx context.Context, method string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+h.cfg.BotToken+"/"+method, bytes.NewReader(raw))
	if err != nil {
		return errors.New("telegram request failed")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := h.client.Do(req)
	if err != nil {
		return errors.New("telegram transport failed")
	}
	defer response.Body.Close()
	var result telegramResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&result); err != nil {
		return errors.New("telegram response failed")
	}
	if !result.OK {
		return &telegramError{result.ErrorCode, result.Parameters.RetryAfter}
	}
	return nil
}
func (h *Handler) createTelegramLink(w http.ResponseWriter, r *http.Request, p *principal) {
	if h.cfg.BotToken == "" || h.cfg.BotUsername == "" || h.cfg.WebhookSecret == "" {
		fail(w, 503, "Telegram linking is not configured")
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 503, "Could not create Telegram link")
		return
	}
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		// Account lock serializes browser tabs and confirms against unlink/relink.
		var id string
		if err := tx.NewRaw(`SELECT id FROM sunday_open_accounts WHERE id=? FOR UPDATE`, p.ID).Scan(ctx, &id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_telegram_links SET consumed_at=NOW() WHERE account_id=? AND consumed_at IS NULL`, p.ID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO sunday_open_telegram_links(account_id,token_hash,expires_at) VALUES(?,?,NOW()+INTERVAL '10 minutes')`, p.ID, h.digest("telegram", token))
		return err
	})
	if err != nil {
		fail(w, 503, "Could not create Telegram link")
		return
	}
	writeJSON(w, 200, map[string]string{"url": "https://t.me/" + url.PathEscape(strings.TrimPrefix(h.cfg.BotUsername, "@")) + "?start=" + url.QueryEscape(token)})
}

type telegramUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	IsBot    bool   `json:"is_bot"`
}
type telegramMessage struct {
	Text string       `json:"text"`
	From telegramUser `json:"from"`
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
}
type telegramUpdate struct {
	UpdateID *int64           `json:"update_id"`
	Message  *telegramMessage `json:"message"`
	Callback *struct {
		ID      string           `json:"id"`
		Data    string           `json:"data"`
		From    telegramUser     `json:"from"`
		Message *telegramMessage `json:"message"`
	} `json:"callback_query"`
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fail(w, 405, "Method not allowed")
		return
	}
	if h.cfg.WebhookSecret == "" || h.cfg.BotToken == "" || !hmac.Equal([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(h.cfg.WebhookSecret)) {
		fail(w, 403, "Invalid webhook secret")
		return
	}
	var update telegramUpdate
	// Telegram evolves the update schema; unknown fields must remain accepted.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&update); err != nil {
		fail(w, 400, "Invalid Telegram update")
		return
	}
	if update.UpdateID == nil {
		fail(w, 400, "Missing Telegram update ID")
		return
	}
	// The receipt transaction commits only after handling succeeds. Concurrent
	// retries wait for this unique key, while crashes roll back the receipt.
	receipt, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w, 503, "Webhook temporarily unavailable")
		return
	}
	defer receipt.Rollback()
	claimed, err := receipt.ExecContext(r.Context(), `INSERT INTO sunday_open_telegram_updates(update_id) VALUES(?) ON CONFLICT DO NOTHING`, *update.UpdateID)
	if err != nil {
		fail(w, 503, "Webhook temporarily unavailable")
		return
	}
	count, _ := claimed.RowsAffected()
	if count == 0 {
		success(w)
		return
	}
	if m := update.Message; m != nil && m.Chat.Type == "private" && m.From.ID == m.Chat.ID && !m.From.IsBot {
		fields := strings.Fields(m.Text)
		if len(fields) == 2 && (fields[0] == "/start" || fields[0] == "/start@"+strings.TrimPrefix(h.cfg.BotUsername, "@")) && len(fields[1]) == 43 {
			var linkID string
			err := receipt.NewRaw(`UPDATE sunday_open_telegram_links SET telegram_user_id=?,telegram_chat_id=?,telegram_username=? WHERE token_hash=? AND expires_at>NOW() AND consumed_at IS NULL AND (telegram_user_id IS NULL OR telegram_user_id=?) RETURNING id`, m.From.ID, m.Chat.ID, m.From.Username, h.digest("telegram", fields[1]), m.From.ID).Scan(r.Context(), &linkID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				fail(w, 503, "Webhook temporarily unavailable")
				return
			}
			if err == nil { // The opaque callback is bound to this link and private Telegram identity.
				if err := h.telegram(r.Context(), "sendMessage", map[string]any{"chat_id": m.Chat.ID, "text": "Confirm linking this Telegram account to your Sunday Open account. You will receive updates only for events you register for.", "reply_markup": map[string]any{"inline_keyboard": [][]map[string]string{{{"text": "Confirm Sunday Open link", "callback_data": "confirm:" + linkID}}}}}); err != nil {
					fail(w, 503, "Please retry linking")
					return
				}
			}
		}
	}
	if c := update.Callback; c != nil && c.Message != nil && c.Message.Chat.Type == "private" && c.Message.Chat.ID == c.From.ID && !c.From.IsBot && strings.HasPrefix(c.Data, "confirm:") {
		linked := false
		err := func(ctx context.Context, tx bun.Tx) error {
			var link struct {
				ID, AccountID                  string
				TelegramUserID, TelegramChatID int64
				TelegramUsername               string
			}
			// Account then link lock ordering agrees with create/unlink.
			rawID := strings.TrimPrefix(c.Data, "confirm:")
			if len(rawID) != 36 {
				return nil
			}
			var accountID string
			if err := tx.NewRaw(`SELECT account_id FROM sunday_open_telegram_links WHERE id=? AND consumed_at IS NULL AND expires_at>NOW()`, rawID).Scan(ctx, &accountID); err != nil {
				return nil
			}
			var lockedID string
			if err := tx.NewRaw(`SELECT id FROM sunday_open_accounts WHERE id=? FOR UPDATE`, accountID).Scan(ctx, &lockedID); err != nil {
				return err
			}
			if err := tx.NewRaw(`SELECT id,account_id,telegram_user_id,telegram_chat_id,COALESCE(telegram_username,'') AS telegram_username FROM sunday_open_telegram_links WHERE id=? AND telegram_user_id=? AND telegram_chat_id=? AND consumed_at IS NULL AND expires_at>NOW() FOR UPDATE`, rawID, c.From.ID, c.Message.Chat.ID).Scan(ctx, &link); err != nil {
				return nil
			}
			// Unique telegram_user_id prevents attaching one Telegram identity to two Sunday accounts.
			if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(?)`, link.TelegramUserID); err != nil {
				return err
			}
			var inUse bool
			if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM sunday_open_accounts WHERE telegram_user_id=? AND id<>?)`, link.TelegramUserID, link.AccountID).Scan(ctx, &inUse); err != nil {
				return err
			}
			if inUse {
				return nil
			}
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_accounts SET telegram_user_id=?,telegram_chat_id=?,telegram_username=?,telegram_blocked=false,updated_at=NOW() WHERE id=?`, link.TelegramUserID, link.TelegramChatID, link.TelegramUsername, link.AccountID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_telegram_links SET consumed_at=NOW() WHERE account_id=? AND consumed_at IS NULL`, link.AccountID); err != nil {
				return err
			}
			// Linking enables future messages; queued history from before this
			// confirmation is suppressed instead of replayed into a new chat.
			if _, err := tx.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),locked_until=NULL,last_error='before link activation' WHERE account_id=? AND sent_at IS NULL`, link.AccountID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE external_registrations xr SET snapshot=`+snapshotSQL+` FROM events e WHERE xr.event_id=e.id AND xr.sunday_open_account_id=? AND e.status='published' AND e.publish_on_sunday_open`, link.AccountID); err != nil {
				return err
			}
			linked = true
			return nil
		}(r.Context(), receipt)
		if err != nil {
			fail(w, 503, "Webhook temporarily unavailable")
			return
		}
		text := "This link is expired or this Telegram account is already linked. Create a new link in Sunday Open."
		if err == nil && linked {
			text = "Telegram is linked. Sunday Open event updates are enabled."
		}
		_ = h.telegram(r.Context(), "answerCallbackQuery", map[string]any{"callback_query_id": c.ID, "text": text, "show_alert": true})
	}
	if err := receipt.Commit(); err != nil {
		fail(w, 503, "Webhook temporarily unavailable")
		return
	}
	success(w)
}

// Start runs in the existing backend process. Intent is persisted in PostgreSQL
// and leases make work recoverable after restarts. A timeout after Telegram
// accepted a message can lead to a duplicate; delivery is not exactly once.
func (h *Handler) Start(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		h.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (h *Handler) tick(ctx context.Context) {
	if len(h.cfg.Secret) < 32 {
		return
	}
	workCtx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()
	if err := h.planNotifications(workCtx); err != nil {
		return
	}
	if h.cfg.BotToken == "" {
		return
	}
	for i := 0; i < 20; i++ {
		if !h.deliverOne(workCtx) {
			break
		}
	}
}
func (h *Handler) planNotifications(ctx context.Context) error {
	return h.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Only currently published Sunday metadata can update a snapshot; hidden
		// and archived events retain their last participant-visible metadata.
		var rows []struct {
			ID, EventID       int32
			AccountID, Status string
			Snapshot, Current json.RawMessage
			RegisteredAt      time.Time
			EventStatus       string
			Published         bool
			StartTime         time.Time
		}
		if err := tx.NewRaw(`SELECT xr.id,xr.event_id,xr.sunday_open_account_id AS account_id,xr.status,xr.snapshot,xr.registered_at,e.status AS event_status,e.publish_on_sunday_open AS published,e.start_time,`+snapshotSQL+` AS current FROM external_registrations xr JOIN events e ON e.id=xr.event_id JOIN sunday_open_accounts a ON a.id=xr.sunday_open_account_id WHERE xr.id>? AND xr.status='registered' AND (e.end_time>NOW()-INTERVAL '1 day' OR (e.status='cancelled' AND xr.snapshot->>'eventStatus' IS DISTINCT FROM 'cancelled')) ORDER BY xr.id FOR UPDATE OF xr SKIP LOCKED LIMIT 1000`, h.plannerCursor).Scan(ctx, &rows); err != nil {
			return err
		}
		for _, row := range rows {
			var old, current eventSnapshot
			if err := json.Unmarshal(row.Snapshot, &old); err != nil {
				return err
			}
			if err := json.Unmarshal(row.Current, &current); err != nil {
				return err
			}
			payload := row.Snapshot
			if row.EventStatus == "cancelled" && old.EventStatus != "cancelled" {
				old.EventStatus = "cancelled"
				payload, _ = json.Marshal(old)
				if err := enqueue(ctx, tx, fmt.Sprintf("event-cancelled:%d:%d", row.ID, row.RegisteredAt.UnixMicro()), row.AccountID, row.ID, row.EventID, "event_cancelled", payload, time.Now()); err != nil {
					return err
				}
			} else if row.Published && row.EventStatus == "published" {
				payload = row.Current
				if !old.StartTime.Equal(current.StartTime) || !old.EndTime.Equal(current.EndTime) || old.LocationText != current.LocationText || old.LocationRevision != current.LocationRevision {
					// Only time/place changes produce change notifications. A title-only edit does not.
					revision := h.digest("event-revision", current.EventRevision+current.StartTime.String()+current.EndTime.String()+current.LocationRevision)
					if err := enqueue(ctx, tx, fmt.Sprintf("changed:%d:%d:%s", row.ID, row.RegisteredAt.UnixMicro(), revision), row.AccountID, row.ID, row.EventID, "event_changed", payload, time.Now()); err != nil {
						return err
					}
				}
			}
			if _, err := tx.ExecContext(ctx, `UPDATE external_registrations SET snapshot=?::jsonb WHERE id=?`, string(payload), row.ID); err != nil {
				return err
			}
			if row.EventStatus != "published" || !row.Published {
				continue
			}
			for _, reminder := range []struct {
				name     string
				duration time.Duration
			}{{"24h", 24 * time.Hour}, {"1h", time.Hour}, {"15m", 15 * time.Minute}} {
				due := row.StartTime.Add(-reminder.duration)
				// Do not send old reminders for an event booked after their due time.
				if due.Before(row.RegisteredAt) || due.After(time.Now()) || due.Before(time.Now().Add(-5*time.Minute)) || !row.StartTime.After(time.Now()) {
					continue
				}
				key := fmt.Sprintf("reminder:%d:%d:%s:%d", row.ID, row.RegisteredAt.UnixMicro(), reminder.name, row.StartTime.Unix())
				if err := enqueue(ctx, tx, key, row.AccountID, row.ID, row.EventID, "reminder_"+reminder.name, payload, time.Now()); err != nil {
					return err
				}
			}
		}
		if len(rows) == 1000 {
			h.plannerCursor = rows[len(rows)-1].ID
		} else {
			h.plannerCursor = 0
		}
		return nil
	})
}
func (h *Handler) deliverOne(ctx context.Context) bool {
	var job struct {
		ID              int64
		AccountID, Kind string
		RegistrationID  *int32
		EventID         *int32
		Payload         json.RawMessage
		Attempts        int
		ChatID          *int64
		Blocked         bool
		CreatedAt       time.Time
	}
	// A short lease is committed before contacting Telegram; no network request holds a database transaction.
	err := h.db.NewRaw(`WITH claimed AS (SELECT id FROM sunday_open_notification_deliveries WHERE sent_at IS NULL AND attempts<12 AND available_at<=NOW() AND (locked_until IS NULL OR locked_until<NOW()) ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE sunday_open_notification_deliveries d SET locked_until=NOW()+INTERVAL '2 minutes',attempts=attempts+1 FROM claimed,sunday_open_accounts a WHERE d.id=claimed.id AND a.id=d.account_id RETURNING d.id,d.account_id,d.kind,d.registration_id,d.event_id,d.payload,d.attempts,d.created_at,a.telegram_chat_id AS chat_id,a.telegram_blocked AS blocked`).Scan(ctx, &job)
	if err != nil {
		return false
	}
	discard := job.ChatID == nil || job.Blocked
	var snapshot eventSnapshot
	_ = json.Unmarshal(job.Payload, &snapshot)
	if job.RegistrationID != nil {
		var current struct {
			Status             string
			RegisteredAt       time.Time
			EventStatus        string
			Published          bool
			StartTime, EndTime time.Time
			LocationRevision   string
		}
		err = h.db.NewRaw(`SELECT xr.status,xr.registered_at,e.status AS event_status,e.publish_on_sunday_open AS published,e.start_time,e.end_time,md5(e.structured_location::text) AS location_revision FROM external_registrations xr JOIN events e ON e.id=xr.event_id WHERE xr.id=? AND xr.sunday_open_account_id=?`, *job.RegistrationID, job.AccountID).Scan(ctx, &current)
		if errors.Is(err, sql.ErrNoRows) {
			discard = true
		} else if err != nil {
			return false
		} else {
			discard = discard || current.RegisteredAt.After(job.CreatedAt)
			metadataMatches := current.StartTime.Equal(snapshot.StartTime) && current.EndTime.Equal(snapshot.EndTime) && current.LocationRevision == snapshot.LocationRevision
			if job.Kind == "registered" || strings.HasPrefix(job.Kind, "reminder_") {
				discard = discard || !metadataMatches
			}
			switch job.Kind {
			case "cancelled":
				discard = discard || current.Status != "cancelled"
			case "event_cancelled":
				discard = discard || current.Status != "registered" || current.EventStatus != "cancelled"
			case "registered":
				discard = discard || current.Status != "registered" || current.EventStatus != "published" || !current.Published
			case "event_changed":
				discard = discard || current.Status != "registered" || current.EventStatus != "published" || !current.Published || !current.StartTime.Equal(snapshot.StartTime) || !current.EndTime.Equal(snapshot.EndTime) || current.LocationRevision != snapshot.LocationRevision
			}
		}
	}
	if strings.HasPrefix(job.Kind, "reminder_") && job.RegistrationID != nil {
		var valid bool
		err = h.db.NewRaw(`SELECT EXISTS(SELECT 1 FROM external_registrations xr JOIN events e ON e.id=xr.event_id WHERE xr.id=? AND xr.status='registered' AND xr.sunday_open_account_id=? AND xr.registered_at<=? AND e.status='published' AND e.publish_on_sunday_open AND e.start_time=? AND e.start_time>NOW())`, *job.RegistrationID, job.AccountID, job.CreatedAt, snapshot.StartTime).Scan(ctx, &valid)
		if err != nil {
			return false
		}
		discard = discard || !valid
		duration := map[string]time.Duration{"reminder_24h": 24 * time.Hour, "reminder_1h": time.Hour, "reminder_15m": 15 * time.Minute}[job.Kind]
		discard = discard || snapshot.StartTime.Add(-duration).Before(time.Now().Add(-5*time.Minute))
	}
	if discard {
		_, _ = h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),locked_until=NULL,last_error='not eligible' WHERE id=?`, job.ID)
		return true
	}
	text := notificationText(job.Kind, snapshot.Registration)
	if snapshot.Slug != "" {
		text += "\n" + strings.TrimRight(strings.Split(h.cfg.Origin, ",")[0], "/") + "/events/" + url.PathEscape(snapshot.Slug)
	}
	err = h.telegram(ctx, "sendMessage", map[string]any{"chat_id": *job.ChatID, "text": text, "link_preview_options": map[string]bool{"is_disabled": false}})
	if err == nil {
		_, _ = h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),locked_until=NULL,last_error=NULL WHERE id=?`, job.ID)
		return true
	}
	retry := time.Minute * time.Duration(1<<min(job.Attempts, 8))
	var tg *telegramError
	if errors.As(err, &tg) {
		if tg.code == 403 {
			_, _ = h.db.ExecContext(ctx, `UPDATE sunday_open_accounts SET telegram_blocked=true,updated_at=NOW() WHERE id=? AND telegram_chat_id=?`, job.AccountID, *job.ChatID)
			_, _ = h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET sent_at=NOW(),locked_until=NULL,last_error='telegram blocked' WHERE id=?`, job.ID)
			return true
		}
		if tg.retryAfter > 0 {
			retry = time.Duration(tg.retryAfter) * time.Second
		}
	}
	_, _ = h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET locked_until=NULL,available_at=?,last_error='telegram delivery failed' WHERE id=?`, time.Now().Add(retry), job.ID)
	return true
}
func notificationText(kind string, event Registration) string {
	label := "Sunday Open registration confirmed"
	switch kind {
	case "cancelled":
		label = "Your Sunday Open registration was cancelled"
	case "event_cancelled":
		label = "Your Sunday Open event was cancelled"
	case "event_changed":
		label = "Your Sunday Open event time or location changed"
	case "reminder_24h":
		label = "Sunday Open reminder: 24 hours"
	case "reminder_1h":
		label = "Sunday Open reminder: 1 hour"
	case "reminder_15m":
		label = "Sunday Open reminder: 15 minutes"
	}
	return label + "\n" + event.Title + "\n" + event.StartTime.Format("02 Jan 2006 15:04 MST") + "\n" + event.LocationText
}
