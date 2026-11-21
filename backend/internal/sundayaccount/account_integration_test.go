//go:build integration

package sundayaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/studyverse/ems-backend/internal/testsupport"
)

type recordingMailer struct {
	code string
	fail bool
}

func (m *recordingMailer) SendCode(_ context.Context, _ string, code string) error {
	m.code = code
	if m.fail {
		return fmt.Errorf("offline")
	}
	return nil
}
func accountFixture(t *testing.T) (*Handler, *recordingMailer) {
	t.Helper()
	db := testsupport.IsolatedDB(t)
	for _, table := range []string{"events", "event_slugs", "event_registrations", "sunday_open_accounts", "sunday_open_onboarding", "sunday_open_auth_challenges", "sunday_open_sessions", "sunday_open_telegram_links", "sunday_open_telegram_updates", "external_registrations", "sunday_open_notification_deliveries"} {
		if _, err := db.ExecContext(context.Background(), "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	m := &recordingMailer{}
	return New(db, Config{Origin: "https://sunday.example", Secret: strings.Repeat("s", 32)}, m), m
}
func callAccount(t *testing.T, h *Handler, method, path string, body any, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, prefix+path, bytes.NewReader(raw))
	r.Header.Set("Origin", "https://sunday.example")
	r.Header.Set("X-CSRF-Token", csrf)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

var nextTestPhone atomic.Int64

func loginAccount(t *testing.T, h *Handler, m *recordingMailer, email string) (*http.Cookie, string, string) {
	t.Helper()
	w := callAccount(t, h, "POST", "request-code", map[string]string{"email": email, "fullName": "Sunday Visitor"}, nil, "")
	if w.Code != 200 {
		t.Fatalf("request %d %s", w.Code, w.Body)
	}
	var challenge struct {
		ChallengeID string `json:"challengeId"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &challenge); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "verify-code", map[string]string{"challengeId": challenge.ChallengeID, "code": m.code}, nil, "")
	if w.Code != 200 {
		t.Fatalf("verify %d %s", w.Code, w.Body)
	}
	var session struct {
		Account Account `json:"account"`
		CSRF    string  `json:"csrfToken"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	cookie := w.Result().Cookies()[0]
	profile := callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": session.Account.FullName, "phone": fmt.Sprintf("+7700%07d", nextTestPhone.Add(1))}, cookie, session.CSRF)
	if profile.Code != 200 {
		t.Fatalf("complete verified profile %d %s", profile.Code, profile.Body)
	}
	return cookie, session.CSRF, session.Account.ID
}
func TestEmailCodeSessionsAndOwnership(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, id := loginAccount(t, h, m, "VISITOR@example.test")
	if id == "" || csrf == "" {
		t.Fatal("missing identity")
	}
	w := callAccount(t, h, "GET", "session", nil, cookie, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "visitor@example.test") {
		t.Fatalf("session %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "Changed"}, cookie, "")
	if w.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d", w.Code)
	}
	w = callAccount(t, h, "PATCH", "profile", map[string]string{"fullName": "Changed"}, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("profile %d %s", w.Code, w.Body)
	}
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,capacity,check_in_secret,check_in_token) VALUES(1,'Visible','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{"text":"Hall"}','published',true,1,'a','b'),(2,'Hidden','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',false,1,'c','d'); INSERT INTO event_slugs(event_id,slug,is_current) VALUES(1,'visible',true)`); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 2}, cookie, csrf)
	if w.Code != 404 {
		t.Fatalf("hidden registration %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("registration %d %s", w.Code, w.Body)
	}
	var saved struct {
		Registration Registration `json:"registration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 409 {
		t.Fatalf("duplicate %d", w.Code)
	}
	otherCookie, otherCSRF, _ := loginAccount(t, h, m, "other@example.test")
	w = callAccount(t, h, "DELETE", fmt.Sprintf("registrations/%d", saved.Registration.ID), nil, otherCookie, otherCSRF)
	if w.Code != 404 {
		t.Fatalf("cross-account cancellation %d", w.Code)
	}
	if _, err := h.db.ExecContext(context.Background(), `UPDATE events SET publish_on_sunday_open=false,title='Private revised title' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "GET", "registrations", nil, cookie, "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "Private revised title") || !strings.Contains(w.Body.String(), "Visible") {
		t.Fatalf("history leaks hidden revision: %d %s", w.Code, w.Body)
	}
	w = callAccount(t, h, "DELETE", fmt.Sprintf("registrations/%d", saved.Registration.ID), nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("cancel %d %s", w.Code, w.Body)
	}
	if _, err := h.db.ExecContext(context.Background(), `UPDATE events SET publish_on_sunday_open=true WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatalf("restore %d %s", w.Code, w.Body)
	}
	var restored struct {
		Registration Registration `json:"registration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &restored)
	if restored.Registration.ID != saved.Registration.ID {
		t.Fatal("created duplicate instead of restoring")
	}
	w = callAccount(t, h, "POST", "logout", nil, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = callAccount(t, h, "GET", "session", nil, cookie, "")
	if !strings.Contains(w.Body.String(), `"account":null`) {
		t.Fatal("logged-out session survived")
	}
}
func TestCodeAttemptLimitCooldownAndSingleUse(t *testing.T) {
	h, m := accountFixture(t)
	w := callAccount(t, h, "POST", "request-code", map[string]string{"email": "limited@example.test", "fullName": "Visitor"}, nil, "")
	var challenge struct {
		ID string `json:"challengeId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &challenge)
	correct := m.code
	wrong := "000000"
	if correct == wrong {
		wrong = "111111"
	}
	for i := 0; i < 5; i++ {
		w = callAccount(t, h, "POST", "verify-code", map[string]string{"challengeId": challenge.ID, "code": wrong}, nil, "")
		if w.Code != 400 {
			t.Fatalf("wrong code accepted %d", w.Code)
		}
	}
	w = callAccount(t, h, "POST", "verify-code", map[string]string{"challengeId": challenge.ID, "code": correct}, nil, "")
	if w.Code != 400 {
		t.Fatal("sixth attempt accepted")
	}
	w = callAccount(t, h, "POST", "request-code", map[string]string{"email": "limited@example.test", "fullName": "Visitor"}, nil, "")
	if w.Code != 429 {
		t.Fatalf("cooldown missing %d", w.Code)
	}
	cookie, _, _ := loginAccount(t, h, m, "single@example.test")
	if cookie == nil {
		t.Fatal("missing cookie")
	}
	var singleID string
	if err := h.db.NewRaw(`SELECT id FROM sunday_open_auth_challenges WHERE email='single@example.test'`).Scan(context.Background(), &singleID); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "verify-code", map[string]string{"challengeId": singleID, "code": m.code}, nil, "")
	if w.Code != 400 {
		t.Fatal("consumed code reused")
	}
}

func TestResendInvalidatesPreviousChallenge(t *testing.T) {
	h, m := accountFixture(t)
	w := callAccount(t, h, "POST", "request-code", map[string]string{"email": "resend@example.test", "fullName": "Visitor"}, nil, "")
	var first struct {
		ID string `json:"challengeId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &first)
	oldCode := m.code
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sunday_open_auth_challenges SET created_at=NOW()-INTERVAL '61 seconds' WHERE id=?`, first.ID); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "request-code", map[string]string{"email": "resend@example.test", "fullName": "Visitor"}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = callAccount(t, h, "POST", "verify-code", map[string]string{"challengeId": first.ID, "code": oldCode}, nil, "")
	if w.Code != 400 {
		t.Fatal("older emailed code remains valid after resend")
	}
}
