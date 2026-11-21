package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	// Server
	Port string `koanf:"PORT"`
	Host string `koanf:"HOST"`

	// Database
	DatabaseURL      string `koanf:"DATABASE_URL"`
	DatabaseMaxConns int32  `koanf:"DATABASE_MAX_CONNS"`
	RedisURL         string `koanf:"REDIS_URL"`

	// CORS
	CORSOrigins []string `koanf:"CORS_ORIGINS"`

	// Kratos
	KratosPublicURL         string `koanf:"KRATOS_PUBLIC_URL"`
	KratosAdminURL          string `koanf:"KRATOS_ADMIN_URL"`
	KratosSessionCookieName string `koanf:"KRATOS_SESSION_COOKIE_NAME"`

	// SpiceDB
	SpiceDBEndpoint     string `koanf:"SPICEDB_ENDPOINT"`
	SpiceDBPresharedKey string `koanf:"SPICEDB_PRESHARED_KEY"`
	SpiceDBInsecure     bool   `koanf:"SPICEDB_INSECURE"`
	SpiceDBSkipVerifyCA bool   `koanf:"SPICEDB_SKIP_VERIFY_CA"`

	// Platform
	PlatformID string `koanf:"PLATFORM_ID"`

	// Meilisearch
	MeilisearchURL       string `koanf:"MEILISEARCH_URL"`
	MeilisearchMasterKey string `koanf:"MEILISEARCH_MASTER_KEY"`

	// Storage
	StorageProvider         string `koanf:"STORAGE_PROVIDER"`
	StorageEndpoint         string `koanf:"STORAGE_ENDPOINT"`
	StorageRegion           string `koanf:"STORAGE_REGION"`
	StorageAccountName      string `koanf:"STORAGE_ACCOUNT_NAME"`
	StorageAccountKey       string `koanf:"STORAGE_ACCOUNT_KEY"`
	StorageContainer        string `koanf:"STORAGE_CONTAINER"`
	StoragePrivateContainer string `koanf:"STORAGE_PRIVATE_CONTAINER"`
	StoragePrivateEndpoint  string `koanf:"STORAGE_PRIVATE_ENDPOINT"`
	StoragePublicURL        string `koanf:"STORAGE_PUBLIC_URL"`

	// Telegram Bot
	TelegramBotURL               string `koanf:"TELEGRAM_BOT_URL"`
	TelegramBotToken             string `koanf:"TELEGRAM_BOT_TOKEN"`
	TelegramMiniAppClientSecret  string `koanf:"TELEGRAM_MINIAPP_CLIENT_SECRET"`
	TelegramMiniAppPrivateKeyB64 string `koanf:"TELEGRAM_MINIAPP_PRIVATE_KEY_B64"`
	TelegramMiniAppRedirectURI   string `koanf:"TELEGRAM_MINIAPP_REDIRECT_URI"`

	// Map App (Next.js frontend)
	MapAppURL string `koanf:"MAP_APP_URL"`
	// Optional server-only SigNoz UI origin and interactive login credentials.
	SigNozUIURL         string `koanf:"SIGNOZ_UI_URL"`
	SigNozLoginEmail    string `koanf:"SIGNOZ_LOGIN_EMAIL"`
	SigNozLoginPassword string `koanf:"SIGNOZ_LOGIN_PASSWORD"`
	// Private Hermes control plane. Credentials are read from the mounted token file.
	HermesOpsURL        string `koanf:"HERMES_OPS_URL"`
	HermesOpsTokenFile  string `koanf:"HERMES_OPS_TOKEN_FILE"`
	HermesDashboardURL  string `koanf:"HERMES_DASHBOARD_URL"`
	HermesAllowedOrigin string `koanf:"HERMES_ALLOWED_ORIGIN"`
	HermesOwnerIDSHA256 string `koanf:"HERMES_OWNER_ID_SHA256"`
	// Internal RPC secrets
	InternalReminderSecret               string `koanf:"INTERNAL_REMINDER_SECRET"`
	ExternalRegistrationCancelSecret     string `koanf:"EXTERNAL_REGISTRATION_CANCEL_SECRET"`
	SundayOpenOrigins                    string `koanf:"SUNDAYOPEN_ORIGINS"`
	SundayOpenTrustedProxyCIDRs          string `koanf:"SUNDAYOPEN_TRUSTED_PROXY_CIDRS"`
	SundayOpenAuthSecret                 string `koanf:"SUNDAYOPEN_AUTH_SECRET"`
	SundayOpenSMTPURL                    string `koanf:"SUNDAYOPEN_SMTP_URL"`
	SundayOpenSMTPFrom                   string `koanf:"SUNDAYOPEN_SMTP_FROM"`
	SundayOpenTelegramBotToken           string `koanf:"SUNDAYOPEN_TELEGRAM_BOT_TOKEN"`
	SundayOpenTelegramBotUsername        string `koanf:"SUNDAYOPEN_TELEGRAM_BOT_USERNAME"`
	SundayOpenTelegramBotConfigEncrypted string `koanf:"SUNDAYOPEN_TELEGRAM_BOT_CONFIG_ENCRYPTED"`
	SundayOpenTelegramWebhookSecret      string `koanf:"SUNDAYOPEN_TELEGRAM_WEBHOOK_SECRET"`

	// Logging
	LogLevel string `koanf:"LOG_LEVEL"`

	// Telemetry (OpenTelemetry)
	OTELEnabled        bool   `koanf:"OTEL_ENABLED"`
	OTELEndpoint       string `koanf:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTELServiceName    string `koanf:"OTEL_SERVICE_NAME"`
	OTELServiceVersion string `koanf:"OTEL_SERVICE_VERSION"`
	OTELEnvironment    string `koanf:"OTEL_ENVIRONMENT"`
}

func Load() *Config {
	k := koanf.New(".")

	// 1. Defaults
	_ = k.Load(confmap.Provider(map[string]interface{}{
		"PORT":                        "5555",
		"HOST":                        "0.0.0.0",
		"DATABASE_URL":                "postgres://postgres:postgres@localhost:5432/eventdb?sslmode=disable",
		"DATABASE_MAX_CONNS":          16,
		"REDIS_URL":                   "redis://:secret@localhost:6379/0",
		"CORS_ORIGINS":                "http://localhost:5173,http://localhost:6868,http://localhost:6869,http://localhost:3002",
		"KRATOS_PUBLIC_URL":           "http://localhost:4433",
		"KRATOS_ADMIN_URL":            "http://localhost:4434",
		"KRATOS_SESSION_COOKIE_NAME":  "ory_kratos_session",
		"SPICEDB_ENDPOINT":            "localhost:50051",
		"SPICEDB_PRESHARED_KEY":       "",
		"SPICEDB_INSECURE":            true,
		"SPICEDB_SKIP_VERIFY_CA":      false,
		"PLATFORM_ID":                 "default",
		"MEILISEARCH_URL":             "http://localhost:7700",
		"MEILISEARCH_MASTER_KEY":      "masterKey123",
		"STORAGE_PROVIDER":            "rustfs",
		"STORAGE_ENDPOINT":            "",
		"STORAGE_REGION":              "us-east-1",
		"STORAGE_ACCOUNT_NAME":        "",
		"STORAGE_ACCOUNT_KEY":         "",
		"STORAGE_CONTAINER":           "ems-images",
		"STORAGE_PRIVATE_CONTAINER":   "ems-registration-files",
		"STORAGE_PRIVATE_ENDPOINT":    "",
		"STORAGE_PUBLIC_URL":          "",
		"TELEGRAM_BOT_URL":            "http://localhost:3001",
		"TELEGRAM_BOT_TOKEN":          "",
		"MAP_APP_URL":                 "https://map.webspace.cat",
		"SIGNOZ_UI_URL":               "",
		"SIGNOZ_LOGIN_EMAIL":          "",
		"SIGNOZ_LOGIN_PASSWORD":       "",
		"HERMES_OPS_URL":              "",
		"HERMES_OPS_TOKEN_FILE":       "",
		"HERMES_DASHBOARD_URL":        "https://auth.webspace.cat/hermes/",
		"HERMES_ALLOWED_ORIGIN":       "https://map.webspace.cat",
		"LOG_LEVEL":                   "debug",
		"INTERNAL_REMINDER_SECRET":    "",
		"SUNDAYOPEN_ORIGINS":          "https://sunday-open.vercel.app,http://localhost:3002,http://localhost:3003",
		"OTEL_ENABLED":                false,
		"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317",
		"OTEL_SERVICE_NAME":           "ems-backend",
		"OTEL_SERVICE_VERSION":        "",
		"OTEL_ENVIRONMENT":            "development",
	}, "."), nil)

	// 2. .env file (walk up from cwd to find it)
	if envPath := findEnvFile(); envPath != "" {
		if err := k.Load(file.Provider(envPath), dotenv.Parser()); err != nil {
			slog.Warn("Failed to load .env file", "path", envPath, "error", err)
		}
	}

	// 3. OS environment variables (highest priority)
	_ = k.Load(env.Provider(".", env.Opt{}), nil)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		slog.Error("Failed to unmarshal config", "error", err)
		os.Exit(1)
	}

	// Keep existing production settings compatible with the site's custom domain.
	// Separate development/custom deployments retain their explicit origin boundary.
	origins := strings.Split(cfg.SundayOpenOrigins, ",")
	vercel, social := false, false
	for _, origin := range origins {
		vercel = vercel || strings.TrimSpace(origin) == "https://sunday-open.vercel.app"
		social = social || strings.TrimSpace(origin) == "https://sundayopen.social"
	}
	if vercel && !social {
		cfg.SundayOpenOrigins = "https://sundayopen.social," + cfg.SundayOpenOrigins
	}

	// Handle CORS_ORIGINS: koanf loads it as a string, split into slice
	if origins := k.String("CORS_ORIGINS"); origins != "" {
		cfg.CORSOrigins = strings.Split(origins, ",")
	}

	return &cfg
}

// SigNozOrigin validates the optional UI origin without including its value in errors.
// Invalid optional configuration disables these endpoints, rather than stopping the application.
func (c *Config) SigNozOrigin() (string, error) {
	if c == nil {
		return "", errors.New("SigNoz configuration unavailable")
	}
	raw := strings.TrimSpace(c.SigNozUIURL)
	if raw == "" {
		return "", nil
	}
	origin, err := url.Parse(raw)
	// Browsers reject percent-escaped host delimiters and IPv6 zone identifiers.
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || strings.Contains(origin.Host, "%") || origin.User != nil ||
		(origin.Path != "" && origin.Path != "/") || origin.RawPath != "" || origin.RawQuery != "" ||
		origin.ForceQuery || strings.ContainsAny(raw, "#\\") {
		return "", errors.New("invalid SigNoz configuration")
	}
	if port := origin.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return "", errors.New("invalid SigNoz configuration")
		}
	}
	origin.Path = ""
	return origin.String(), nil
}

// HermesURLs restricts public redirects to the owner dashboard and explicit loopback development URLs.
// Errors deliberately exclude configuration values because they can contain accidental credentials.
func (c *Config) HermesURLs() (dashboard, browserOrigin string, err error) {
	if c == nil {
		return "", "", errors.New("Hermes configuration unavailable")
	}
	dashboard, browserOrigin = c.HermesDashboardURL, c.HermesAllowedOrigin
	if dashboard == "" {
		dashboard = "https://auth.webspace.cat/hermes/"
	}
	if browserOrigin == "" {
		browserOrigin = "https://map.webspace.cat"
	}
	if dashboard != "https://auth.webspace.cat/hermes/" {
		u, valid := hermesHTTPURL(dashboard)
		if !valid || !hermesLoopback(u) || u.Path != "/hermes/" {
			return "", "", errors.New("invalid Hermes dashboard configuration")
		}
	}
	if browserOrigin != "https://map.webspace.cat" {
		u, valid := hermesHTTPURL(browserOrigin)
		if !valid || !hermesLoopback(u) || u.Path != "" {
			return "", "", errors.New("invalid Hermes browser origin configuration")
		}
	}
	return dashboard, browserOrigin, nil
}

func (c *Config) HermesOpsOrigin() (string, error) {
	if c == nil || c.HermesOpsURL == "" {
		return "", errors.New("Hermes ops configuration unavailable")
	}
	u, valid := hermesHTTPURL(c.HermesOpsURL)
	if !valid || (u.Path != "" && u.Path != "/") {
		return "", errors.New("invalid Hermes ops configuration")
	}
	u.Path = ""
	return u.String(), nil
}

func hermesHTTPURL(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil ||
		u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || strings.ContainsAny(raw, "?#\\% \t\r\n") {
		return nil, false
	}
	if port := u.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, false
		}
	}
	return u, true
}

func hermesLoopback(u *url.URL) bool {
	return u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}

// findEnvFile walks up from cwd (or ENV_FILE) to locate the nearest .env file.
func findEnvFile() string {
	if explicit := os.Getenv("ENV_FILE"); explicit != "" {
		if _, err := os.Stat(explicit); err == nil {
			return explicit
		}
		return ""
	}

	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		candidate := filepath.Join(wd, ".env")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return ""
		}
		wd = parent
	}
}
