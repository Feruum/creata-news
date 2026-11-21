//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func configureWidget(h *Handler) {
	h.cfg.BotToken = "12345:test-bot-token"
	h.cfg.BotUsername = "sundayopen_bot"
}

func widgetSession(t *testing.T, raw []byte) (Account, string) {
	t.Helper()
	var session struct {
		Account Account `json:"account"`
		CSRF    string  `json:"csrfToken"`
	}
	if err := json.Unmarshal(raw, &session); err != nil || session.Account.ID == "" || session.CSRF == "" {
		t.Fatalf("invalid session %s: %v", raw, err)
	}
	return session.Account, session.CSRF
}

func TestTelegramWidgetCreatesAndRecoversOnlyByVerifiedIdentity(t *testing.T) {
	h, m := accountFixture(t)
	configureWidget(h)
	m.fail = true
	data := signedWidgetPayload(h.cfg.BotToken, 123456789, time.Now())
	w := callAccount(t, h, "POST", "telegram-login", map[string]any{
		"telegram": data, "fullName": "  Sunday Participant  ", "email": " PARTICIPANT@Example.test ",
	}, nil, "")
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	account, csrf := widgetSession(t, w.Body.Bytes())
	if account.FullName != "Sunday Participant" || account.Email != "participant@example.test" || account.Phone != "" || !account.TelegramLinked {
		t.Fatalf("wrong participant %+v", account)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].MaxAge != 30*24*3600 {
		t.Fatalf("missing safe session cookie %+v", cookies)
	}
	var stored struct {
		ID, ChatID int64
		Verified   bool
	}
	if err := h.db.NewRaw(`SELECT telegram_user_id AS id,telegram_chat_id AS chat_id,email IS NOT NULL AS verified FROM sunday_open_accounts WHERE id=?`, account.ID).Scan(context.Background(), &stored); err != nil {
		t.Fatal(err)
	}
	if stored.ID != 123456789 || stored.ChatID != stored.ID || stored.Verified || m.code != "" {
		t.Fatalf("contact promoted to identity or Telegram not connected: %+v", stored)
	}
	if w := callAccount(t, h, "POST", "logout", nil, cookies[0], csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = callAccount(t, h, "POST", "telegram-login", map[string]any{
		"telegram": data, "fullName": "Overwrite", "email": "not even an email",
	}, nil, "")
	if w.Code != 200 {
		t.Fatalf("recover %d %s", w.Code, w.Body)
	}
	recovered, _ := widgetSession(t, w.Body.Bytes())
	if recovered.ID != account.ID || recovered.FullName != account.FullName || recovered.Email != account.Email {
		t.Fatalf("recovery overwrote contact details %+v", recovered)
	}
	otherData := signedWidgetPayload(h.cfg.BotToken, 987654321, time.Now())
	w = callAccount(t, h, "POST", "telegram-login", map[string]any{
		"telegram": otherData, "fullName": "Other Participant", "email": account.Email,
	}, nil, "")
	if w.Code != 200 {
		t.Fatalf("second identity %d %s", w.Code, w.Body)
	}
	other, _ := widgetSession(t, w.Body.Bytes())
	if other.ID == account.ID {
		t.Fatal("matching contact email merged Telegram identities")
	}
}

func TestTelegramWidgetDoesNotClaimGuestOrEmailVerifiedAccountsByContact(t *testing.T) {
	h, m := accountFixture(t)
	configureWidget(h)
	_, _, verifiedID := loginAccount(t, h, m, "same@example.test")
	_, _, guest := startGuest(t, h, "same@example.test", "")
	w := callAccount(t, h, "POST", "telegram-login", map[string]any{
		"telegram": signedWidgetPayload(h.cfg.BotToken, 333, time.Now()),
		"fullName": "New Identity", "email": "same@example.test",
	}, nil, "")
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	account, _ := widgetSession(t, w.Body.Bytes())
	if account.ID == guest.ID || account.ID == verifiedID {
		t.Fatal("typed contact email claimed an existing account")
	}
}

func TestTelegramWidgetNewIdentityRequiresFullNameAndEmail(t *testing.T) {
	h, _ := accountFixture(t)
	configureWidget(h)
	for _, tc := range []struct{ fullName, email string }{
		{"", "person@example.test"}, {"Person", ""}, {"Person", "invalid"},
	} {
		w := callAccount(t, h, "POST", "telegram-login", map[string]any{
			"telegram": signedWidgetPayload(h.cfg.BotToken, 123, time.Now()),
			"fullName": tc.fullName, "email": tc.email,
		}, nil, "")
		if w.Code != 400 || len(w.Result().Cookies()) != 0 {
			t.Fatalf("invalid profile created session: %d %s", w.Code, w.Body)
		}
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_accounts`).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("invalid profile left account behind: %d %v", count, err)
	}
}

func TestTelegramWidgetAcceptsMissingOrEmptyOptionalProfileFields(t *testing.T) {
	for _, includeEmpty := range []bool{false, true} {
		t.Run(map[bool]string{false: "absent", true: "empty"}[includeEmpty], func(t *testing.T) {
			h, _ := accountFixture(t)
			configureWidget(h)
			data := map[string]any{"id": int64(123456789), "first_name": "Visitor", "auth_date": time.Now().Unix()}
			if includeEmpty {
				data["last_name"], data["username"], data["photo_url"] = "", "", ""
			}
			signWidgetPayload(h.cfg.BotToken, data)
			w := callAccount(t, h, "POST", "telegram-login", map[string]any{
				"telegram": data, "fullName": "Sunday Participant", "email": "participant@example.test",
			}, nil, "")
			if w.Code != 200 {
				t.Fatalf("valid optional fields %d %s", w.Code, w.Body)
			}
			account, _ := widgetSession(t, w.Body.Bytes())
			if !account.TelegramLinked || account.TelegramUsername != nil {
				t.Fatalf("optional profile fields were required %+v", account)
			}
		})
	}
}

func TestTelegramWidgetRecoveryKeepsPendingFutureNotifications(t *testing.T) {
	h, _ := accountFixture(t)
	configureWidget(h)
	data := signedWidgetPayload(h.cfg.BotToken, 123456789, time.Now())
	w := callAccount(t, h, "POST", "telegram-login", map[string]any{
		"telegram": data, "fullName": "Sunday Participant", "email": "participant@example.test",
	}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	account, _ := widgetSession(t, w.Body.Bytes())
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,kind,payload,available_at) VALUES('future-reminder',?,'reminder_1h','{}',NOW()+INTERVAL '1 hour')`, account.ID); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "telegram-login", map[string]any{"telegram": data}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var pending bool
	if err := h.db.NewRaw(`SELECT sent_at IS NULL FROM sunday_open_notification_deliveries WHERE delivery_key='future-reminder'`).Scan(context.Background(), &pending); err != nil || !pending {
		t.Fatal("signing in again suppressed a previously enabled event reminder", err)
	}
}

func TestTelegramWidgetLinksOwnSessionWithCSRFAndRotatesCookie(t *testing.T) {
	h, _ := accountFixture(t)
	configureWidget(h)
	cookie, csrf, guest := startGuest(t, h, "existing@example.test", "+77001234567")
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,kind,payload) VALUES('before-widget',?,'registered','{}')`, guest.ID); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"telegram": signedWidgetPayload(h.cfg.BotToken, 123456789, time.Now()), "fullName": "Overwrite", "email": "overwrite@example.test"}
	for _, badCSRF := range []string{"", "another-account-token"} {
		if w := callAccount(t, h, "POST", "telegram-login", input, cookie, badCSRF); w.Code != 403 {
			t.Fatalf("link accepted bad CSRF %d %s", w.Code, w.Body)
		}
	}
	w := callAccount(t, h, "POST", "telegram-login", input, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("link %d %s", w.Code, w.Body)
	}
	linked, linkedCSRF := widgetSession(t, w.Body.Bytes())
	newCookie := w.Result().Cookies()[0]
	if linked.ID != guest.ID || !linked.TelegramLinked || linked.Phone != guest.Phone || linked.Email != guest.Email || linked.FullName != guest.FullName || newCookie.Value == cookie.Value || linkedCSRF == csrf {
		t.Fatalf("wrong linked account or unrotated session %+v", linked)
	}
	if w := callAccount(t, h, "GET", "session", nil, cookie, ""); !strings.Contains(w.Body.String(), `"account":null`) {
		t.Fatal("old browser token still authenticates", w.Body.String())
	}
	var suppressed bool
	if err := h.db.NewRaw(`SELECT sent_at IS NOT NULL FROM sunday_open_notification_deliveries WHERE delivery_key='before-widget'`).Scan(ctx, &suppressed); err != nil || !suppressed {
		t.Fatal("link would replay pre-link notifications", err)
	}
	otherCookie, otherCSRF, other := startGuest(t, h, "other@example.test", "")
	w = callAccount(t, h, "POST", "telegram-login", input, otherCookie, otherCSRF)
	if w.Code != 409 {
		t.Fatalf("claimed another participant's Telegram: %d %s", w.Code, w.Body)
	}
	var owner string
	if err := h.db.NewRaw(`SELECT id FROM sunday_open_accounts WHERE telegram_user_id=123456789`).Scan(ctx, &owner); err != nil || owner != guest.ID || owner == other.ID {
		t.Fatal("Telegram ownership transferred", err)
	}
	if w := callAccount(t, h, "GET", "session", nil, newCookie, ""); !strings.Contains(w.Body.String(), guest.ID) {
		t.Fatal("linked session stopped working", w.Body.String())
	}
}

func TestConcurrentTelegramWidgetLoginCreatesOneAccount(t *testing.T) {
	h, _ := accountFixture(t)
	configureWidget(h)
	input := map[string]any{
		"telegram": signedWidgetPayload(h.cfg.BotToken, 777, time.Now()),
		"fullName": "Concurrent Participant", "email": "concurrent@example.test",
	}
	var workers sync.WaitGroup
	results := make(chan string, 2)
	for range 2 {
		workers.Go(func() {
			w := widgetRequest(h, input)
			if w.Code != 200 {
				results <- "error: " + w.Body.String()
				return
			}
			var session struct {
				Account Account `json:"account"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &session)
			results <- session.Account.ID
		})
	}
	workers.Wait()
	close(results)
	var first string
	for id := range results {
		if strings.HasPrefix(id, "error:") || id == "" || (first != "" && first != id) {
			t.Fatalf("concurrent login did not recover one identity: %q then %q", first, id)
		}
		first = id
	}
}

func TestGuestCanRegisterWithoutPhoneAndKeepOptionalStoredPhone(t *testing.T) {
	h, _ := accountFixture(t)
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,'s','t')`); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"first@example.test", "second@example.test"} {
		cookie, csrf, account := startGuest(t, h, email, "")
		w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
		if w.Code != 200 {
			t.Fatalf("register without phone %d %s", w.Code, w.Body)
		}
		var noPhone bool
		if err := h.db.NewRaw(`SELECT phone IS NULL FROM external_registrations WHERE sunday_open_account_id=?`, account.ID).Scan(ctx, &noPhone); err != nil || !noPhone {
			t.Fatal("absent phone must stay NULL so unrelated participants do not conflict", err)
		}
	}
	cookie, csrf, _ := startGuest(t, h, "legacy-phone@example.test", "+77001234567")
	w := callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "Updated name", "email": "legacy-phone@example.test"}, cookie, csrf)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "+77001234567") {
		t.Fatalf("profile edit erased stored phone %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "Updated name", "phone": ""}, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("optional phone could not be cleared %d %s", w.Code, w.Body)
	}
	var session struct {
		Account Account `json:"account"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	if session.Account.Phone != "" {
		t.Fatalf("phone was not cleared %+v", session.Account)
	}
}
