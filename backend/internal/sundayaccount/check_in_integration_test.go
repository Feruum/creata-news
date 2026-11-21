//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/checkin"
)

func seedCheckIn(t *testing.T, h *Handler) {
	t.Helper()
	_, err := h.db.ExecContext(context.Background(), `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,capacity,check_in_secret,check_in_token) VALUES(1,'Sunday QR','{}',1,1,NOW()-INTERVAL '10 minutes',NOW()+INTERVAL '1 hour','{"text":"Hall"}','published',true,2,'secret-one','token-one'),(2,'Other event','{}',1,1,NOW(),NOW()+INTERVAL '1 hour','{}','published',true,2,'secret-two','token-two'); INSERT INTO event_slugs(event_id,slug,is_current) VALUES(1,'sunday-qr',true)`)
	if err != nil {
		t.Fatal(err)
	}
}

func scanGuest(t *testing.T, h *Handler, token, code string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", prefix+"check-in/start?token="+url.QueryEscape(token)+"&code="+url.QueryEscape(code), nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func guestTicket(t *testing.T, h *Handler) string {
	t.Helper()
	w := scanGuest(t, h, "token-one", checkin.Code("secret-one", time.Now().Unix()/30))
	if w.Code != 200 {
		t.Fatalf("scan %d %s", w.Code, w.Body)
	}
	var response struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(response.URL)
	if err != nil || u.Scheme != "https" || u.Host != "sunday.example" || u.Path != "/check-in" {
		t.Fatalf("unsafe redirect: %s", response.URL)
	}
	fragment, _ := url.ParseQuery(u.Fragment)
	ticket := fragment.Get("ticket")
	if ticket == "" {
		t.Fatal("missing event continuation")
	}
	return ticket
}

func TestGuestCheckInResumesSameEventAndPreservesFirstAttendance(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	ticket := guestTicket(t, h)
	// The guest signs in after scanning. The request cannot choose another event or account.
	cookie, csrf, id := loginAccount(t, h, m, "scan@example.test")
	for i := 0; i < 2; i++ {
		w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"eventId":1`) || !strings.Contains(w.Body.String(), `"checkedInAt"`) {
			t.Fatalf("check in %d %s", w.Code, w.Body)
		}
	}
	var total int
	if err := h.db.NewRaw(`SELECT count(*) FROM external_registrations WHERE event_id=1 AND sunday_open_account_id=? AND status='registered' AND checked_in_at IS NOT NULL`, id).Scan(context.Background(), &total); err != nil || total != 1 {
		t.Fatalf("attendance %d %v", total, err)
	}
	var first time.Time
	if err := h.db.NewRaw(`SELECT checked_in_at FROM external_registrations WHERE sunday_open_account_id=?`, id).Scan(context.Background(), &first); err != nil {
		t.Fatal(err)
	}
	w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var after time.Time
	_ = h.db.NewRaw(`SELECT checked_in_at FROM external_registrations WHERE sunday_open_account_id=?`, id).Scan(context.Background(), &after)
	if !first.Equal(after) {
		t.Fatal("repeat check-in replaced first attendance time")
	}
	w = callAccount(t, h, "GET", "registrations", nil, cookie, "")
	if !strings.Contains(w.Body.String(), `"checkedInAt"`) {
		t.Fatalf("history omitted attendance: %s", w.Body)
	}
}

func TestGuestCheckInAllowsTwoParticipantsWithoutPhones(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	ticket := guestTicket(t, h)
	type participant struct {
		cookie *http.Cookie
		csrf   string
		id     string
	}
	var accounts []participant
	for _, email := range []string{"first-walk-in@example.test", "second-walk-in@example.test"} {
		cookie, csrf, account := startGuest(t, h, email, "")
		w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"checkedInAt"`) {
			t.Fatalf("walk-in without phone %d %s", w.Code, w.Body)
		}
		accounts = append(accounts, participant{cookie, csrf, account.ID})
	}
	var counts struct{ Participants, WithoutPhones int }
	if err := h.db.NewRaw(`SELECT count(DISTINCT sunday_open_account_id) AS participants,count(*) FILTER(WHERE phone IS NULL) AS without_phones FROM external_registrations WHERE event_id=1 AND checked_in_at IS NOT NULL`).Scan(context.Background(), &counts); err != nil || counts.Participants != 2 || counts.WithoutPhones != 2 {
		t.Fatalf("unrelated walk-ins shared contact identity or stored empty phones: %+v %v", counts, err)
	}
	// Both participants retain their own existing seat when capacity is full.
	for _, account := range accounts {
		w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, account.cookie, account.csrf)
		if w.Code != 200 {
			t.Fatalf("repeat walk-in without phone %d %s", w.Code, w.Body)
		}
		var seats int
		if err := h.db.NewRaw(`SELECT count(*) FROM external_registrations WHERE event_id=1 AND sunday_open_account_id=?`, account.id).Scan(context.Background(), &seats); err != nil || seats != 1 {
			t.Fatalf("repeat walk-in duplicated attendance: %d %v", seats, err)
		}
	}
}

func TestGuestCheckInRejectsInvalidScansAndRequiresAccountAndCSRF(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	for _, pair := range [][2]string{{"token-one", "forged"}, {"token-two", checkin.Code("secret-one", time.Now().Unix()/30)}, {"missing", "forged"}} {
		if w := scanGuest(t, h, pair[0], pair[1]); w.Code == 200 {
			t.Fatal("invalid scan accepted")
		}
	}
	ticket := guestTicket(t, h)
	if w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, nil, ""); w.Code != 401 {
		t.Fatalf("anonymous check-in: %d", w.Code)
	}
	cookie, csrf, _ := loginAccount(t, h, m, "secure-scan@example.test")
	if w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, ""); w.Code != 403 {
		t.Fatalf("missing CSRF: %d", w.Code)
	}
	for _, invalid := range []string{ticket + "tampered", "", "invalid"} {
		if w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": invalid}, cookie, csrf); w.Code == 200 {
			t.Fatal("forged ticket accepted")
		}
	}
	expired, _ := checkin.Sign(h.cfg.Secret, checkin.Claim{Purpose: "guest", EventID: 1, EventToken: "token-one", ExpiresAt: time.Now().Add(-time.Second).Unix()})
	if w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": expired}, cookie, csrf); w.Code != 400 {
		t.Fatalf("expired ticket: %d %s", w.Code, w.Body)
	}
	if w := callAccount(t, h, "POST", "check-in", map[string]any{"ticket": ticket, "eventId": 2}, cookie, csrf); w.Code != 400 {
		t.Fatalf("event injection: %d", w.Code)
	}
}

func TestGuestCheckInRechecksPublicationAndPhase(t *testing.T) {
	for _, mutation := range []string{"publish_on_sunday_open=false", "status='cancelled'", "completed_at=NOW()", "end_time=NOW()-INTERVAL '4 hours',start_time=NOW()-INTERVAL '5 hours'", "check_in_token='revoked'"} {
		t.Run(mutation, func(t *testing.T) {
			h, m := accountFixture(t)
			seedCheckIn(t, h)
			ticket := guestTicket(t, h)
			cookie, csrf, _ := loginAccount(t, h, m, fmt.Sprintf("recheck-%d@example.test", time.Now().UnixNano()))
			if _, err := h.db.ExecContext(context.Background(), "UPDATE events SET "+mutation+" WHERE id=1"); err != nil {
				t.Fatal(err)
			}
			if w := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf); w.Code == 200 {
				t.Fatalf("accepted changed event: %s", mutation)
			}
		})
	}
}

func TestGuestCheckInReusesRegistrationEvenWhenFull(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	cookie, csrf, id := loginAccount(t, h, m, "registered-scan@example.test")
	w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if _, err := h.db.ExecContext(context.Background(), "UPDATE events SET capacity=1 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	ticket := guestTicket(t, h)
	w = callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("existing seat at full event: %d %s", w.Code, w.Body)
	}
	var count int
	_ = h.db.NewRaw("SELECT count(*) FROM external_registrations WHERE sunday_open_account_id=?", id).Scan(context.Background(), &count)
	if count != 1 {
		t.Fatalf("created duplicate: %d", count)
	}
	other, otherCSRF, _ := loginAccount(t, h, m, "new-scan@example.test")
	w = callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, other, otherCSRF)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "checkedInAt") {
		t.Fatalf("full event blocked new QR participant: %d %s", w.Code, w.Body)
	}
}

func TestGuestCheckInRevalidatesSessionBeforeAttendance(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	ticket := guestTicket(t, h)
	cookie, _, accountID := loginAccount(t, h, m, "revoked-scan@example.test")
	request := httptest.NewRequest("POST", prefix+"check-in", strings.NewReader(`{"ticket":"`+ticket+`"}`))
	request.AddCookie(cookie)
	p, err := h.authenticate(request)
	if err != nil {
		t.Fatal(err)
	}
	// Revoke after initial HTTP authentication, before the attendance transaction.
	if _, err := h.db.ExecContext(context.Background(), `DELETE FROM sunday_open_sessions WHERE account_id=?`, accountID); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	h.completeCheckIn(response, request, p)
	if response.Code != 401 {
		t.Fatalf("revoked session recorded attendance: %d %s", response.Code, response.Body)
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM external_registrations WHERE sunday_open_account_id=?`, accountID).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("revoked session allocated a seat: %d %v", count, err)
	}
}

func TestGuestCheckInAllowsWalkInWhenCampusRegistrationsFillCapacity(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	if _, err := h.db.ExecContext(context.Background(), `UPDATE events SET capacity=1 WHERE id=1; INSERT INTO event_registrations(event_id,user_id,status) VALUES(1,42,'registered')`); err != nil {
		t.Fatal(err)
	}
	cookie, csrf, _ := loginAccount(t, h, m, "campus-full@example.test")
	response := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": guestTicket(t, h)}, cookie, csrf)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "checkedInAt") {
		t.Fatalf("full event blocked guest QR attendance: %d %s", response.Code, response.Body)
	}
}

func TestGuestContinuationBindsConfiguredSundayOrigin(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	h.cfg.Origin = "https://sunday.example,https://second.example"
	h.cfg.MapAppURL = "https://map.example"
	ticket := guestTicket(t, h)
	call := func(origin string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("GET", prefix+"check-in/info", nil)
		request.Header.Set("Origin", origin)
		request.Header.Set("X-Check-In-Ticket", ticket)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, request)
		return response
	}
	if response := call("https://second.example"); response.Code == 200 {
		t.Fatal("guest continuation moved to another configured origin")
	}
	if response := call("https://sunday.example"); response.Code != 200 || !strings.Contains(response.Body.String(), "https://map.example/check-in/token-one?university=1") {
		t.Fatalf("original-origin continuation failed: %d %s", response.Code, response.Body)
	}
	// Server-side same-origin proxies may omit Origin while preserving the signed ticket.
	if response := call(""); response.Code != 200 {
		t.Fatalf("same-origin continuation failed: %d %s", response.Code, response.Body)
	}
	h.cfg.Origin = "https://second.example"
	if response := call(""); response.Code == 200 {
		t.Fatal("removed origin retained valid guest continuation")
	}
}
