//go:build integration

package sundayaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeTelegram(h *Handler, body string, received *[]map[string]any) {
	h.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var value map[string]any
		_ = json.NewDecoder(r.Body).Decode(&value)
		*received = append(*received, value)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

var nextUpdateID atomic.Int64

func hook(t *testing.T, h *Handler, body any, secret string) *httptest.ResponseRecorder {
	if value, ok := body.(map[string]any); ok {
		copy := make(map[string]any, len(value)+1)
		for key, item := range value {
			copy[key] = item
		}
		if _, ok := copy["update_id"]; !ok {
			copy["update_id"] = nextUpdateID.Add(1)
		}
		body = copy
	}
	t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest("POST", "/api/sundayopen/telegram/webhook", bytes.NewReader(raw))
	r.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestTelegramLinkRequiresPrivateConfirmationAndSingleUse(t *testing.T) {
	h, m := accountFixture(t)
	h.cfg.BotToken = "test-token"
	h.cfg.BotUsername = "sunday_test_bot"
	h.cfg.WebhookSecret = "webhook-secret"
	var received []map[string]any
	fakeTelegram(h, `{"ok":true}`, &received)
	cookie, csrf, id := loginAccount(t, h, m, "telegram@example.test")
	w := callAccount(t, h, "POST", "telegram-link", nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("link %d %s", w.Code, w.Body)
	}
	var result struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	link, _ := url.Parse(result.URL)
	token := link.Query().Get("start")
	start := map[string]any{"message": map[string]any{"text": "/start " + token, "from": map[string]any{"id": 111}, "chat": map[string]any{"id": 111, "type": "private"}}}
	start["update_id"] = int64(999)
	if w := hook(t, h, start, "wrong"); w.Code != 403 {
		t.Fatal("webhook accepted bad secret")
	}
	if w := hook(t, h, start, h.cfg.WebhookSecret); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := hook(t, h, start, h.cfg.WebhookSecret); w.Code != 200 || len(received) != 1 {
		t.Fatal("repeated webhook resent confirmation")
	}
	var linked bool
	if err := h.db.NewRaw(`SELECT telegram_user_id IS NOT NULL FROM sunday_open_accounts WHERE id=?`, id).Scan(context.Background(), &linked); err != nil || linked {
		t.Fatal("start linked without confirmation", err)
	}
	var linkID string
	if err := h.db.NewRaw(`SELECT id FROM sunday_open_telegram_links WHERE account_id=?`, id).Scan(context.Background(), &linkID); err != nil {
		t.Fatal(err)
	}
	callback := func(user int64) map[string]any {
		return map[string]any{"callback_query": map[string]any{"id": "callback-id", "data": "confirm:" + linkID, "from": map[string]any{"id": user}, "message": map[string]any{"chat": map[string]any{"id": user, "type": "private"}}}}
	}
	hook(t, h, callback(222), h.cfg.WebhookSecret)
	_ = h.db.NewRaw(`SELECT telegram_user_id IS NOT NULL FROM sunday_open_accounts WHERE id=?`, id).Scan(context.Background(), &linked)
	if linked {
		t.Fatal("another Telegram user confirmed link")
	}
	hook(t, h, callback(111), h.cfg.WebhookSecret)
	_ = h.db.NewRaw(`SELECT telegram_user_id IS NOT NULL FROM sunday_open_accounts WHERE id=?`, id).Scan(context.Background(), &linked)
	if !linked {
		t.Fatal("confirmation did not link")
	}
	w = callAccount(t, h, "DELETE", "telegram-link", nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	hook(t, h, callback(111), h.cfg.WebhookSecret)
	_ = h.db.NewRaw(`SELECT telegram_user_id IS NOT NULL FROM sunday_open_accounts WHERE id=?`, id).Scan(context.Background(), &linked)
	if linked {
		t.Fatal("consumed link relinked after unlink")
	}
}
func TestTelegramEventNotificationsAllowLinkPreviews(t *testing.T) {
	h, m := accountFixture(t)
	_, _, id := loginAccount(t, h, m, "preview@example.test")
	h.cfg.BotToken = "test"
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sunday_open_accounts SET telegram_user_id=111,telegram_chat_id=111 WHERE id=?; INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,kind,payload) VALUES('preview',?,'registered','{"slug":"preview-event","title":"Preview event"}')`, id, id); err != nil {
		t.Fatal(err)
	}
	var received []map[string]any
	fakeTelegram(h, `{"ok":true}`, &received)
	if !h.deliverOne(context.Background()) || len(received) != 1 {
		t.Fatal("event notification was not delivered")
	}
	if text, ok := received[0]["text"].(string); !ok || !strings.Contains(text, "https://sunday.example/events/preview-event") {
		t.Fatal("notification is missing its public event link")
	}
	if received[0]["disable_web_page_preview"] == true {
		t.Fatal("notification disables the event link preview with the legacy API option")
	}
	if options, ok := received[0]["link_preview_options"].(map[string]any); ok && options["is_disabled"] == true {
		t.Fatal("notification disables the event link preview")
	}
}

func TestTelegramRetryAfterAndBlockedAreDurable(t *testing.T) {
	h, m := accountFixture(t)
	_, _, id := loginAccount(t, h, m, "delivery@example.test")
	h.cfg.BotToken = "test"
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sunday_open_accounts SET telegram_user_id=111,telegram_chat_id=111 WHERE id=?; INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,kind,payload) VALUES('retry',?,'registered','{}')`, id, id); err != nil {
		t.Fatal(err)
	}
	var received []map[string]any
	fakeTelegram(h, `{"ok":false,"error_code":429,"parameters":{"retry_after":300}}`, &received)
	if !h.deliverOne(context.Background()) {
		t.Fatal("did not claim job")
	}
	var row struct {
		Attempts    int
		AvailableAt time.Time
		SentAt      *time.Time
	}
	if err := h.db.NewRaw(`SELECT attempts,available_at,sent_at FROM sunday_open_notification_deliveries WHERE delivery_key='retry'`).Scan(context.Background(), &row); err != nil {
		t.Fatal(err)
	}
	if row.Attempts != 1 || row.SentAt != nil || row.AvailableAt.Before(time.Now().Add(290*time.Second)) {
		t.Fatalf("lost retry_after: %+v", row)
	}
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sunday_open_notification_deliveries SET available_at=NOW() WHERE delivery_key='retry'`); err != nil {
		t.Fatal(err)
	}
	fakeTelegram(h, `{"ok":false,"error_code":403}`, &received)
	h.deliverOne(context.Background())
	var blocked bool
	_ = h.db.NewRaw(`SELECT telegram_blocked FROM sunday_open_accounts WHERE id=?`, id).Scan(context.Background(), &blocked)
	if !blocked {
		t.Fatal("blocked bot not recorded")
	}
}

func TestFailedConfirmationIsNotSentAfterCancellation(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, id := loginAccount(t, h, m, "staleconfirm@example.test")
	ctx := context.Background()
	h.cfg.BotToken = "test"
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,'s','t'); UPDATE sunday_open_accounts SET telegram_chat_id=111,telegram_user_id=111 WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var saved struct {
		Registration Registration `json:"registration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	var received []map[string]any
	fakeTelegram(h, `{"ok":false,"error_code":500}`, &received)
	h.deliverOne(ctx)
	w = callAccount(t, h, "DELETE", fmt.Sprintf("registrations/%d", saved.Registration.ID), nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	received = nil
	fakeTelegram(h, `{"ok":true}`, &received)
	h.deliverOne(ctx)
	if len(received) != 1 || !strings.Contains(received[0]["text"].(string), "registration was cancelled") {
		t.Fatal("cancellation notification not delivered")
	}
	if _, err := h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET available_at=NOW() WHERE kind='registered'`); err != nil {
		t.Fatal(err)
	}
	h.deliverOne(ctx)
	if len(received) != 1 {
		t.Fatal("stale confirmation delivered after cancellation")
	}
}

func TestQueuedNotificationsRejectChangedTimeAndLocation(t *testing.T) {
	for _, kind := range []string{"registered", "reminder_1h"} {
		t.Run(kind, func(t *testing.T) {
			h, m := accountFixture(t)
			cookie, csrf, id := loginAccount(t, h, m, kind+"@example.test")
			ctx := context.Background()
			h.cfg.BotToken = "test"
			if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '1 hour'-INTERVAL '20 seconds',NOW()+INTERVAL '2 hours','{"type":"external","external_address":"Old venue"}','published',true,'s','t'); UPDATE sunday_open_accounts SET telegram_chat_id=111,telegram_user_id=111 WHERE id=?`, id); err != nil {
				t.Fatal(err)
			}
			w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			if kind == "reminder_1h" {
				if _, err := h.db.ExecContext(ctx, `UPDATE external_registrations SET registered_at=NOW()-INTERVAL '2 hours'; UPDATE sunday_open_notification_deliveries SET sent_at=NOW()`); err != nil {
					t.Fatal(err)
				}
				if err := h.planNotifications(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var received []map[string]any
			fakeTelegram(h, `{"ok":false,"error_code":500}`, &received)
			h.deliverOne(ctx)
			// A confirmation's time is stale after reschedule; a reminder's location
			// is stale even when its start time has stayed exactly the same.
			if kind == "registered" {
				_, _ = h.db.ExecContext(ctx, `UPDATE events SET start_time=start_time+INTERVAL '1 hour',end_time=end_time+INTERVAL '1 hour',updated_at=NOW()`)
			} else {
				_, _ = h.db.ExecContext(ctx, `UPDATE events SET structured_location='{"type":"external","external_address":"Next venue"}',updated_at=NOW()`)
			}
			if err := h.planNotifications(ctx); err != nil {
				t.Fatal(err)
			}
			received = nil
			fakeTelegram(h, `{"ok":true}`, &received)
			h.deliverOne(ctx)
			if len(received) != 1 || !strings.Contains(received[0]["text"].(string), "event time or location changed") {
				t.Fatal("expected current change notification")
			}
			if _, err := h.db.ExecContext(ctx, `UPDATE sunday_open_notification_deliveries SET available_at=NOW() WHERE kind=?`, kind); err != nil {
				t.Fatal(err)
			}
			h.deliverOne(ctx)
			if len(received) != 1 {
				t.Fatal("stale metadata retried after current change notification")
			}
		})
	}
}
