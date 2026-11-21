//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func startGuest(t *testing.T, h *Handler, email, phone string) (*http.Cookie, string, Account) {
	t.Helper()
	w := callAccount(t, h, "POST", "start", map[string]string{"fullName": "Guest Visitor", "email": email, "phone": phone}, nil, "")
	if w.Code != 200 {
		t.Fatalf("guest start %d %s", w.Code, w.Body)
	}
	var result struct {
		Account Account `json:"account"`
		CSRF    string  `json:"csrfToken"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return w.Result().Cookies()[0], result.CSRF, result.Account
}
func TestGuestStartRequiresContactDetailsWithoutEmailDelivery(t *testing.T) {
	h, m := accountFixture(t)
	m.fail = true
	for _, missing := range []string{"fullName", "email"} {
		input := map[string]string{"fullName": "Visitor", "email": "guest@example.test", "phone": "+7 700 123 45 67"}
		delete(input, missing)
		if w := callAccount(t, h, "POST", "start", input, nil, ""); w.Code != 400 {
			t.Fatalf("missing %s accepted: %d", missing, w.Code)
		}
	}
	cookie, csrf, account := startGuest(t, h, " GUEST@Example.test ", "+7 (700) 123-45-67")
	if account.Email != "guest@example.test" || csrf == "" || cookie == nil || account.Phone != "+77001234567" || !account.IsGuest {
		t.Fatalf("invalid guest account %+v", account)
	}
	if m.code != "" {
		t.Fatal("guest flow called email sender")
	}
	var verified bool
	if err := h.db.NewRaw(`SELECT email IS NOT NULL FROM sunday_open_accounts WHERE id=?`, account.ID).Scan(context.Background(), &verified); err != nil || verified {
		t.Fatal("guest contact stored as verified identity", err)
	}
	w := callAccount(t, h, "POST", "start", map[string]string{"fullName": "Overwrite", "email": "other@example.test", "phone": "+77009999999"}, cookie, "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "Overwrite") || len(w.Result().Cookies()) > 0 {
		t.Fatal("existing session overwritten", w.Body.String())
	}
}
func TestGuestBrowsersDoNotMergeByEmailOrTakeOverVerifiedAccount(t *testing.T) {
	h, m := accountFixture(t)
	verifiedCookie, verifiedCSRF, verifiedID := loginAccount(t, h, m, "same@example.test")
	w := callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "Verified", "email": "attacker@example.test", "phone": "+77005555555"}, verifiedCookie, verifiedCSRF)
	if w.Code != 400 {
		t.Fatal("verified identity email was mutable")
	}
	first, firstCSRF, a := startGuest(t, h, "same@example.test", "+77001111111")
	second, _, b := startGuest(t, h, "same@example.test", "+77002222222")
	if a.ID == b.ID || a.ID == verifiedID || b.ID == verifiedID {
		t.Fatal("unverified email merged identities")
	}
	w = callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "My guest", "email": "changed@example.test", "phone": "+7 700 333 33 33"}, first, firstCSRF)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "changed@example.test") {
		t.Fatal("guest profile not updated", w.Body.String())
	}
	for _, cookie := range []*http.Cookie{second, verifiedCookie} {
		w := callAccount(t, h, "GET", "session", nil, cookie, "")
		if strings.Contains(w.Body.String(), "changed@example.test") || !strings.Contains(w.Body.String(), "same@example.test") {
			t.Fatal("guest edit changed someone else's identity", w.Body.String())
		}
	}
}

func TestGuestRateLimitAndVerifiedRegistrationWithoutPhone(t *testing.T) {
	h, m := accountFixture(t)
	for i := 0; i < 3; i++ {
		startGuest(t, h, fmt.Sprintf("guest%d@example.test", i), fmt.Sprintf("+7700111111%d", i))
	}
	if w := callAccount(t, h, "POST", "start", map[string]string{"fullName": "Visitor", "email": "limited@example.test", "phone": "+77001111119"}, nil, ""); w.Code != 429 {
		t.Fatalf("guest rate limit missing: %d", w.Code)
	}
	cookie, csrf, id := loginAccount(t, h, m, "needsphone@example.test")
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sunday_open_accounts SET phone=NULL WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,'s','t')`); err != nil {
		t.Fatal(err)
	}
	if w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf); w.Code != 200 {
		t.Fatalf("verified registration without phone failed: %d %s", w.Code, w.Body)
	}
}
func TestGuestRegistrationIncludesPhoneAndRejectsDuplicateWithoutOwnershipTransfer(t *testing.T) {
	h, _ := accountFixture(t)
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,'s','t')`); err != nil {
		t.Fatal(err)
	}
	first, firstCSRF, a := startGuest(t, h, "same@example.test", "+77001111111")
	second, secondCSRF, _ := startGuest(t, h, "same@example.test", "+7 (700) 111-11-11")
	w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, first, firstCSRF)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var phone, owner string
	if err := h.db.NewRaw(`SELECT phone,sunday_open_account_id FROM external_registrations WHERE event_id=1`).Scan(ctx, &phone, &owner); err != nil || phone != "+77001111111" || owner != a.ID {
		t.Fatalf("missing contact/owner: %s %s %v", phone, owner, err)
	}
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, second, secondCSRF)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "already_exists") {
		t.Fatalf("phone conflict not mapped %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "GET", "registrations", nil, second, "")
	if strings.Contains(w.Body.String(), `"eventId":1`) {
		t.Fatal("duplicate browser gained registration access")
	}
}
