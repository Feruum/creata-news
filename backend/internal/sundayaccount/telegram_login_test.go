package sundayaccount

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

// This signer follows Telegram's documented algorithm independently of the
// production verifier, including optional fields in alphabetical order.
func signedWidgetPayload(token string, id int64, at time.Time) map[string]any {
	data := map[string]any{
		"id": id, "first_name": "Sunday", "last_name": "Visitor",
		"username": "sunday_visitor", "photo_url": "https://t.me/i/userpic/320/avatar.jpg",
		"auth_date": at.Unix(),
	}
	signWidgetPayload(token, data)
	return data
}

func signWidgetPayload(token string, data map[string]any) {
	var fields []string
	for key, value := range data {
		if key != "hash" {
			fields = append(fields, key+"="+fmt.Sprint(value))
		}
	}
	sort.Strings(fields)
	secret := sha256.Sum256([]byte(token))
	signature := hmac.New(sha256.New, secret[:])
	_, _ = signature.Write([]byte(strings.Join(fields, "\n")))
	data["hash"] = hex.EncodeToString(signature.Sum(nil))
}

func widgetRequest(h *Handler, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest("POST", prefix+"telegram-login", bytes.NewReader(raw))
	r.Header.Set("Origin", "https://sunday.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func widgetTestHandler() *Handler {
	return New(nil, Config{
		Origin: "https://sunday.example", Secret: strings.Repeat("s", 32),
		BotToken: "12345:test-bot-token", BotUsername: "sundayopen_bot",
	}, nil)
}

func TestTelegramConfigExposesOnlyConfiguredUsername(t *testing.T) {
	for _, tc := range []struct {
		name, token, username, want string
	}{
		{"configured", "private-token", "@sundayopen_bot", "sundayopen_bot"},
		{"no token", "", "sundayopen_bot", ""},
		{"no username", "private-token", "", ""},
		{"invalid username", "private-token", "bad/name", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := widgetTestHandler()
			h.cfg.BotToken, h.cfg.BotUsername = tc.token, tc.username
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", prefix+"telegram-config", nil))
			var result map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result["botUsername"] != tc.want || len(result) != 2 {
				t.Fatalf("config %d %s: %v", w.Code, w.Body, err)
			}
			if strings.Contains(w.Body.String(), "private-token") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("configuration exposed a secret or was cacheable")
			}
		})
	}
}

func TestTelegramWidgetRejectsInvalidAuthorizationBeforeDatabaseAccess(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		resign bool
	}{
		{"tampered name", func(v map[string]any) { v["first_name"] = "Someone else" }, false},
		{"tampered optional photo", func(v map[string]any) { v["photo_url"] = "https://example.test/other.jpg" }, false},
		{"expired", func(v map[string]any) { v["auth_date"] = time.Now().Add(-11 * time.Minute).Unix() }, true},
		{"future timestamp", func(v map[string]any) { v["auth_date"] = time.Now().Add(2 * time.Minute).Unix() }, true},
		{"zero identity", func(v map[string]any) { v["id"] = int64(0) }, true},
		{"negative identity", func(v map[string]any) { v["id"] = int64(-1) }, true},
		{"unsafe identity", func(v map[string]any) { v["id"] = int64(1 << 53) }, true},
		{"missing name", func(v map[string]any) { delete(v, "first_name") }, true},
		{"oversized name", func(v map[string]any) { v["first_name"] = strings.Repeat("x", 257) }, true},
		{"line break name", func(v map[string]any) { v["first_name"] = "Sunday\nVisitor" }, true},
		{"invalid username", func(v map[string]any) { v["username"] = "bad/name" }, true},
		{"unsafe photo", func(v map[string]any) { v["photo_url"] = "javascript:alert(1)" }, true},
		{"photo credentials", func(v map[string]any) { v["photo_url"] = "https://user:password@t.me/photo.jpg" }, true},
		{"oversized photo", func(v map[string]any) { v["photo_url"] = "https://t.me/" + strings.Repeat("x", 2048) }, true},
		{"invalid hash encoding", func(v map[string]any) { v["hash"] = strings.Repeat("g", 64) }, false},
		{"short hash", func(v map[string]any) { v["hash"] = "abcd" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := widgetTestHandler()
			data := signedWidgetPayload(h.cfg.BotToken, 123456789, time.Now())
			tc.mutate(data)
			if tc.resign {
				signWidgetPayload(h.cfg.BotToken, data)
			}
			w := widgetRequest(h, map[string]any{"telegram": data})
			if w.Code != 401 || !strings.Contains(w.Body.String(), "Telegram authorization is invalid or expired") {
				t.Fatalf("invalid authorization %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestTelegramWidgetUsesOnlyConfiguredBotToken(t *testing.T) {
	h := widgetTestHandler()
	data := signedWidgetPayload("another-bot-token", 123456789, time.Now())
	if w := widgetRequest(h, map[string]any{"telegram": data}); w.Code != 401 || !strings.Contains(w.Body.String(), "Telegram authorization") {
		t.Fatalf("accepted another bot's proof: %d %s", w.Code, w.Body)
	}
}

func TestTelegramWidgetRejectsUnknownAndOversizedPayloads(t *testing.T) {
	for _, raw := range []string{
		`{"telegram":{"id":123,"unknown":"field"}}`,
		`{"telegram":{"id":"123"}}`,
		`{"telegram":{},"password":"unneeded"}`,
		`{"telegram":{},"phone":"+77001234567"}`,
		`{"telegram":{},"phone":" "}`,
		`{"telegram":{},"phone":77001234567}`,
		`{"telegram":{},"phone":true}`,
		`{"telegram":{},"phone":[]}`,
		`{"telegram":{},"phone":{}}`,
		`{"telegram":{},"fullName":"` + strings.Repeat("x", 16384) + `"}`,
		`{"telegram":{}} {"telegram":{}}`,
	} {
		h := widgetTestHandler()
		r := httptest.NewRequest("POST", prefix+"telegram-login", strings.NewReader(raw))
		r.Header.Set("Origin", "https://sunday.example")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("invalid payload accepted: %d %s", w.Code, w.Body)
		}
	}
}

func TestTelegramLoginAcceptsEmptyLegacyPhone(t *testing.T) {
	for _, phone := range []any{"", nil} {
		h := widgetTestHandler()
		w := widgetRequest(h, map[string]any{"idToken": "invalid", "phone": phone})
		if w.Code != 401 {
			t.Fatalf("empty legacy phone rejected before proof validation: %d %s", w.Code, w.Body)
		}
	}
}

func TestTelegramWidgetRateLimitsInvalidAttempts(t *testing.T) {
	h := widgetTestHandler()
	for i := 0; i < 31; i++ {
		w := widgetRequest(h, map[string]any{"telegram": map[string]any{}})
		if i == 30 && (w.Code != 429 || w.Header().Get("Retry-After") == "") {
			t.Fatalf("login attempts were not limited: %d %s", w.Code, w.Body)
		}
	}
}

func TestTelegramWidgetRequiresAllowedOrigin(t *testing.T) {
	h := widgetTestHandler()
	for _, origin := range []string{"", "null", "https://evil.example"} {
		r := httptest.NewRequest("POST", prefix+"telegram-login", strings.NewReader(`{"telegram":{}}`))
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("accepted origin %q: %d", origin, w.Code)
		}
	}
}

func TestPhoneMayBeOmitted(t *testing.T) {
	for _, raw := range []string{"", "  "} {
		if phone, err := normalizePhone(raw); err != nil || phone != "" {
			t.Fatalf("optional phone %q: %q %v", raw, phone, err)
		}
	}
}
