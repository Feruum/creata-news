package sundayaccount

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/telegramconfig"
)

func setupKeyResponse(t *testing.T, handler *Handler) map[string]string {
	t.Helper()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, prefix+"telegram-setup-key", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("setup key returned %d, want 200: %s", w.Code, w.Body)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("setup key must not be cached")
	}
	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(w.Body.String(), handler.cfg.Secret) || strings.Contains(w.Body.String(), handler.cfg.BotToken) {
		t.Fatal("setup key response contains private configuration")
	}
	return result
}

func TestTelegramSetupKeyIsPublicAndStableForThisServer(t *testing.T) {
	h := widgetTestHandler()
	first := setupKeyResponse(t, h)
	second := setupKeyResponse(t, widgetTestHandler())
	publicKey, err := base64.RawURLEncoding.Strict().DecodeString(first["publicKey"])
	if err != nil || len(publicKey) != 32 || first["algorithm"] != "nacl-box-sealed-v1" || len(first["keyId"]) != 64 || len(first) != 3 {
		t.Fatalf("invalid public key descriptor: %#v", first)
	}
	if first["publicKey"] != second["publicKey"] || first["keyId"] != second["keyId"] {
		t.Fatal("same persistent server secret must produce the same encryption key")
	}
	other := widgetTestHandler()
	other.cfg.Secret = strings.Repeat("t", 32)
	if setupKeyResponse(t, other)["publicKey"] == first["publicKey"] {
		t.Fatal("independent environments must have different encryption keys")
	}
}

func TestTelegramSetupKeyCannotWriteConfiguration(t *testing.T) {
	h := widgetTestHandler()
	r := httptest.NewRequest(http.MethodPost, prefix+"telegram-setup-key", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://sunday.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET" {
		t.Fatalf("setup key must be GET-only, got %d", w.Code)
	}
	if h.cfg.BotToken != "12345:test-bot-token" {
		t.Fatal("public setup route changed bot credentials")
	}
}

func TestTelegramSetupKeyRequiresPrivateServerSecret(t *testing.T) {
	h := widgetTestHandler()
	h.cfg.Secret = ""
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, prefix+"telegram-setup-key", nil))
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "publicKey") {
		t.Fatalf("unconfigured server exposed setup key: %d %s", w.Code, w.Body)
	}
}

func TestTelegramEncryptedConfigurationLoadsCredentialsAtomically(t *testing.T) {
	base := widgetTestHandler().cfg
	base.MapAppURL = "https://map.webspace.cat"
	want := telegramconfig.Credentials{
		BotToken: "12345678:" + strings.Repeat("a", 35), BotUsername: "sundayopen_bot",
		WebhookSecret: strings.Repeat("w", 43),
	}
	key, err := telegramconfig.PublicKey(base.Secret, base.MapAppURL)
	if err != nil {
		t.Fatal(err)
	}
	base.BotConfigEncrypted, err = telegramconfig.Seal(key, want)
	if err != nil {
		t.Fatal(err)
	}
	base.BotToken, base.BotUsername, base.WebhookSecret = "legacy-token", "legacy_bot", "legacy-webhook"
	h := New(nil, base, nil)
	if h.cfg.BotToken != want.BotToken || h.cfg.BotUsername != want.BotUsername || h.cfg.WebhookSecret != want.WebhookSecret {
		t.Fatal("encrypted bundle was mixed with legacy credentials")
	}
	if h.widgetBotUsername() != want.BotUsername {
		t.Fatal("encrypted credentials did not enable Telegram login")
	}
	raw, _ := json.Marshal(signedWidgetPayload(want.BotToken, 1234, time.Now()))
	var widget telegramWidgetData
	if err := json.Unmarshal(raw, &widget); err != nil || !h.verifyTelegramWidget(widget, time.Now()) {
		t.Fatal("widget verifier did not use the decrypted bot token")
	}
	for _, change := range []func(*Config){
		func(cfg *Config) { cfg.BotConfigEncrypted = "malformed" },
		func(cfg *Config) { cfg.Secret = strings.Repeat("t", 32) },
		func(cfg *Config) { cfg.MapAppURL = "https://map.dev.webspace.cat" },
	} {
		broken := base
		change(&broken)
		disabled := New(nil, broken, nil)
		if disabled.cfg.BotToken != "" || disabled.cfg.BotUsername != "" || disabled.cfg.WebhookSecret != "" || disabled.widgetBotUsername() != "" {
			t.Fatal("invalid encrypted bundle silently fell back to other bot credentials")
		}
	}
	base.BotConfigEncrypted = ""
	legacy := New(nil, base, nil)
	if legacy.cfg.BotToken != base.BotToken || legacy.cfg.BotUsername != base.BotUsername || legacy.cfg.WebhookSecret != base.WebhookSecret {
		t.Fatal("unconfigured encrypted source changed legacy behavior")
	}
}
