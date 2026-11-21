//go:build integration

package sundayaccount

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func oidcAccountCall(t *testing.T, h *Handler, body any, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest("POST", prefix+"telegram-login", bytes.NewReader(raw))
	r.Header.Set("Origin", "https://sunday.example")
	r.Header.Set("X-CSRF-Token", csrf)
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func oidcChallenge(t *testing.T, h *Handler, cookie *http.Cookie, csrf string) (string, *http.Cookie) {
	t.Helper()
	w := callAccount(t, h, "POST", "telegram-challenge", nil, cookie, csrf)
	var result struct {
		Nonce string `json:"nonce"`
	}
	if json.Unmarshal(w.Body.Bytes(), &result) != nil || w.Code != 200 || result.Nonce == "" {
		t.Fatalf("challenge %d %s", w.Code, w.Body)
	}
	return result.Nonce, w.Result().Cookies()[0]
}

func oidcSessionCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == cookieName {
			return cookie
		}
	}
	t.Fatal("missing session cookie")
	return nil
}

func TestTelegramOIDCCreatesAndRecoversIdentityWithRegistrations(t *testing.T) {
	h, _ := accountFixture(t)
	proofHandler, key, _ := oidcFixture(t)
	h.cfg.BotToken, h.cfg.BotUsername, h.client = proofHandler.cfg.BotToken, proofHandler.cfg.BotUsername, proofHandler.client
	nonce, nonceCookie := oidcChallenge(t, h, nil, "")
	idToken := signOIDC(t, key, oidcClaims(nonce))
	if w := oidcAccountCall(t, h, map[string]string{"idToken": idToken, "nonce": nonce}, nil, ""); w.Code != 401 {
		t.Fatalf("missing browser nonce %d", w.Code)
	}
	w := oidcAccountCall(t, h, map[string]string{"idToken": idToken, "nonce": nonce, "fullName": "OIDC Participant", "email": "oidc@example.test", "phone": ""}, []*http.Cookie{nonceCookie}, "")
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	account, csrf := widgetSession(t, w.Body.Bytes())
	if !account.TelegramLinked || account.FullName != "OIDC Participant" {
		t.Fatalf("wrong account %+v", account)
	}
	sessionCookie := oidcSessionCookie(t, w)
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'OIDC Event','{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{}','published',true,'s','t')`); err != nil {
		t.Fatal(err)
	}
	w = callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, sessionCookie, csrf)
	var saved struct {
		Registration Registration `json:"registration"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil || w.Code != 200 || saved.Registration.ID == 0 {
		t.Fatalf("register %d %s: %v", w.Code, w.Body, err)
	}
	if w := callAccount(t, h, "POST", "logout", nil, sessionCookie, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	for _, tc := range []struct {
		name string
		id   any
	}{
		{"string", "123456789"},
		{"number", int64(123456789)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nonce, nonceCookie := oidcChallenge(t, h, nil, "")
			claims := oidcClaims(nonce)
			claims["id"], claims["sub"] = tc.id, "different-telegram-subject"
			w := oidcAccountCall(t, h, map[string]string{"idToken": signOIDC(t, key, claims), "nonce": nonce, "fullName": "Overwrite", "email": "invalid"}, []*http.Cookie{nonceCookie}, "")
			if w.Code != 200 {
				t.Fatalf("recover %d %s", w.Code, w.Body)
			}
			recovered, _ := widgetSession(t, w.Body.Bytes())
			if recovered.ID != account.ID || recovered.Email != account.Email || recovered.FullName != account.FullName || !recovered.TelegramLinked {
				t.Fatalf("did not recover same identity %+v", recovered)
			}
			w = callAccount(t, h, "GET", "registrations", nil, oidcSessionCookie(t, w), "")
			var history struct {
				Registrations []Registration `json:"registrations"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &history); err != nil || w.Code != 200 || len(history.Registrations) != 1 || history.Registrations[0].ID != saved.Registration.ID || history.Registrations[0].Status != "registered" {
				t.Fatalf("lost registration after recovery %d %s: %v", w.Code, w.Body, err)
			}
		})
	}
}

func TestTelegramOIDCLinksExistingAccountForStringAndNumericIDs(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   any
	}{
		{"string", "123456789"},
		{"number", int64(123456789)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := accountFixture(t)
			proofHandler, key, _ := oidcFixture(t)
			h.cfg.BotToken, h.cfg.BotUsername, h.client = proofHandler.cfg.BotToken, proofHandler.cfg.BotUsername, proofHandler.client
			cookie, csrf, account := startGuest(t, h, "existing-oidc@example.test", "+77001234567")
			if w := callAccount(t, h, "POST", "telegram-challenge", nil, cookie, ""); w.Code != 403 {
				t.Fatalf("challenge skipped authenticated CSRF %d", w.Code)
			}
			nonce, nonceCookie := oidcChallenge(t, h, cookie, csrf)
			claims := oidcClaims(nonce)
			claims["id"] = tc.id
			input := map[string]string{"idToken": signOIDC(t, key, claims), "nonce": nonce, "fullName": "Overwrite", "email": "overwrite@example.test"}
			if w := oidcAccountCall(t, h, input, []*http.Cookie{cookie, nonceCookie}, ""); w.Code != 403 {
				t.Fatalf("link skipped authenticated CSRF %d", w.Code)
			}
			w := oidcAccountCall(t, h, input, []*http.Cookie{cookie, nonceCookie}, csrf)
			if w.Code != 200 {
				t.Fatalf("link %d %s", w.Code, w.Body)
			}
			linked, linkedCSRF := widgetSession(t, w.Body.Bytes())
			linkedCookie := oidcSessionCookie(t, w)
			if linked.ID != account.ID || linked.Email != account.Email || linked.Phone != account.Phone || linked.FullName != account.FullName || !linked.TelegramLinked || linkedCookie.Value == cookie.Value || linkedCSRF == csrf {
				t.Fatalf("wrong linked identity or unrotated session %+v", linked)
			}
			if w := callAccount(t, h, "GET", "session", nil, cookie, ""); !strings.Contains(w.Body.String(), `"account":null`) {
				t.Fatal("old browser token still authenticates", w.Body.String())
			}
			otherCookie, otherCSRF, _ := startGuest(t, h, "other-oidc@example.test", "")
			for _, id := range []any{"123456789", int64(123456789)} {
				nonce, nonceCookie := oidcChallenge(t, h, otherCookie, otherCSRF)
				claims := oidcClaims(nonce)
				claims["id"] = id
				w := oidcAccountCall(t, h, map[string]string{"idToken": signOIDC(t, key, claims), "nonce": nonce}, []*http.Cookie{otherCookie, nonceCookie}, otherCSRF)
				if w.Code != 409 {
					t.Fatalf("stole another account's Telegram identity %d %s", w.Code, w.Body)
				}
			}
			var owner string
			if err := h.db.NewRaw(`SELECT id FROM sunday_open_accounts WHERE telegram_user_id=123456789`).Scan(context.Background(), &owner); err != nil || owner != account.ID {
				t.Fatalf("Telegram ownership transferred to %q: %v", owner, err)
			}
			if w := callAccount(t, h, "GET", "session", nil, linkedCookie, ""); !strings.Contains(w.Body.String(), account.ID) {
				t.Fatal("linked session stopped working", w.Body.String())
			}
		})
	}
}
