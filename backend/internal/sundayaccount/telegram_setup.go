package sundayaccount

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/studyverse/ems-backend/internal/telegramconfig"
)

func telegramConfigScope(cfg Config) string {
	if scope := strings.TrimRight(strings.TrimSpace(cfg.MapAppURL), "/"); scope != "" {
		return scope
	}
	return strings.TrimSpace(cfg.Origin)
}

func resolveTelegramConfig(cfg Config) Config {
	if cfg.BotConfigEncrypted == "" {
		return cfg
	}
	// A sealed bundle is one atomic source; invalid bundles never fall back to
	// a different bot or webhook secret from partial legacy environment fields.
	cfg.BotToken, cfg.BotUsername, cfg.WebhookSecret = "", "", ""
	credentials, err := telegramconfig.Open(cfg.Secret, telegramConfigScope(cfg), cfg.BotConfigEncrypted)
	if err != nil {
		slog.Error("Encrypted Sunday Open Telegram configuration is unavailable")
		return cfg
	}
	cfg.BotToken, cfg.BotUsername, cfg.WebhookSecret = credentials.BotToken, credentials.BotUsername, credentials.WebhookSecret
	return cfg
}

func (h *Handler) telegramSetupKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		fail(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	key, err := telegramconfig.PublicKey(h.cfg.Secret, telegramConfigScope(h.cfg))
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "Telegram configuration key is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, key)
}
