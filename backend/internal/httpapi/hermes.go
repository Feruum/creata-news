package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	ory "github.com/ory/kratos-client-go"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/hermes"
)

// WithHermesRoutes runs before the ordinary router, authentication and telemetry middleware.
// The browser endpoints require a fresh cookie. Private endpoints use independent, purpose-bound credentials.
func WithHermesRoutes(next http.Handler, cfg *config.Config, kratos *ory.APIClient, links hermes.TelegramLinkStore, author hermes.EventActions) http.Handler {
	// Use an isolated, bounded Kratos client: do not trace identity responses or forward API credentials.
	var identityClient *ory.APIClient
	if kratos != nil {
		identityConfig := ory.NewConfiguration()
		identityConfig.Servers = kratos.GetConfig().Servers
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		identityConfig.HTTPClient = &http.Client{Timeout: 5 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		identityClient = ory.NewAPIClient(identityConfig)
	}
	ops := hermes.NewClient(cfg)
	gate := auth.NewCookieMiddleware(identityClient)(newHermesHandler(cfg, ops, links))
	eventsGate := privateHermesEvents(cfg, ops, author)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/internal/hermes/events/") {
			eventsGate.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/internal/hermes/user-count" {
			w.Header().Set("Cache-Control", "no-store")
			credential, bearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if r.Method != http.MethodGet || !bearer || r.Header.Get("Cookie") != "" || r.Header.Get("Origin") != "" || !ops.ValidCountCredential(credential) {
				writeHermesError(w, http.StatusForbidden)
				return
			}
			if links == nil {
				writeHermesError(w, http.StatusServiceUnavailable)
				return
			}
			count, err := links.UserCount(r.Context())
			if err != nil {
				writeHermesError(w, http.StatusServiceUnavailable)
				return
			}
			writeJSON(w, http.StatusOK, map[string]int{"users": count})
			return
		}
		if IsHermesRoute(r.URL.Path) {
			gate.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func IsHermesRoute(path string) bool {
	return path == "/admin/hermes/info" || path == "/admin/hermes/access" || path == "/admin/hermes/settings"
}

type hermesPublicInfo struct {
	hermes.Info
	DashboardURL string `json:"dashboard_url"`
}

type hermesSettings struct {
	Action      string `json:"action"`
	BotToken    string `json:"bot_token,omitempty"`
	GitLabToken string `json:"gitlab_token,omitempty"`
	LinearToken string `json:"linear_token,omitempty"`
	LinearTeam  string `json:"linear_team,omitempty"`
	Command     string `json:"command,omitempty"`
	CoverBase64 string `json:"cover_base64,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	FlowID      string `json:"flow_id,omitempty"`
}

func newHermesHandler(cfg *config.Config, ops *hermes.Client, links hermes.TelegramLinkStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Referrer-Policy", "no-referrer")
		method := http.MethodGet
		if r.URL.Path == "/admin/hermes/settings" {
			method = http.MethodPost
		}
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeHermesError(w, http.StatusMethodNotAllowed)
			return
		}
		identity, status := hermesOwner(r)
		if status != 0 {
			writeHermesError(w, status)
			return
		}
		if r.URL.Path != "/admin/hermes/access" && r.Header.Get("X-Studyverse-Hermes") != "1" {
			writeHermesError(w, http.StatusForbidden)
			return
		}
		dashboard, browserOrigin, err := cfg.HermesURLs()
		if err != nil {
			writeHermesError(w, http.StatusServiceUnavailable)
			return
		}
		payload := hermes.Request{Identity: identity}
		operation := strings.TrimPrefix(r.URL.Path, "/admin/hermes/")
		if operation == "settings" {
			dashboardURL, _ := url.Parse(dashboard)
			dashboardOrigin := dashboardURL.Scheme + "://" + dashboardURL.Host
			origin := r.Header.Get("Origin")
			if origin != browserOrigin && origin != dashboardOrigin {
				writeHermesError(w, http.StatusForbidden)
				return
			}
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				writeHermesError(w, http.StatusUnsupportedMediaType)
				return
			}
			var settings hermesSettings
			limit := int64(8192)
			if r.Header.Get("X-Hermes-Event-Cover") == "1" {
				limit = 716800
			}
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&settings); err != nil {
				writeHermesDecodeError(w, err)
				return
			}
			if err := decoder.Decode(new(interface{})); err != io.EOF {
				writeHermesDecodeError(w, err)
				return
			}
			if !settings.valid() || (limit > 8192 && settings.CoverBase64 == "") {
				writeHermesError(w, http.StatusBadRequest)
				return
			}
			payload.Action, payload.BotToken, payload.GitLabToken, payload.FlowID = settings.Action, settings.BotToken, settings.GitLabToken, settings.FlowID
			payload.LinearToken, payload.LinearTeam = settings.LinearToken, settings.LinearTeam
			payload.Command, payload.RequestID = settings.Command, settings.RequestID
			payload.CoverBase64 = settings.CoverBase64
			if settings.Action == "bind_telegram" {
				// Only the pinned owner may resolve a local Telegram link. Ops checks again on the mutation.
				if _, err := ops.Call(r.Context(), "access", hermes.Request{Identity: identity}); err != nil {
					writeHermesDependencyError(w, err)
					return
				}
				if links == nil {
					writeHermesError(w, http.StatusServiceUnavailable)
					return
				}
				link, err := links.VerifiedLink(r.Context(), identity.KratosID)
				if err != nil {
					writeHermesDependencyError(w, err)
					return
				}
				id, err := strconv.ParseInt(link.ID, 10, 64)
				if err != nil || id <= 0 || strconv.FormatInt(id, 10) != link.ID || link.LinkedAt.IsZero() || link.LinkedAt.After(time.Now()) {
					writeHermesError(w, http.StatusForbidden)
					return
				}
				payload.Telegram = &link
			}
		}
		info, err := ops.Call(r.Context(), operation, payload)
		if err != nil {
			writeHermesDependencyError(w, err)
			return
		}
		if operation == "access" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, http.StatusOK, hermesPublicInfo{Info: info, DashboardURL: dashboard})
	})
}

func hermesOwner(r *http.Request) (hermes.Identity, int) {
	if auth.IsSessionValidationUnavailable(r.Context()) {
		return hermes.Identity{}, http.StatusServiceUnavailable
	}
	session := auth.GetSession(r.Context())
	if session == nil || !session.GetActive() || !session.GetExpiresAt().After(time.Now()) || session.Identity == nil ||
		session.Identity.Id == "" || session.Identity.Id != auth.GetUserID(r.Context()) || session.Identity.GetState() != "active" {
		return hermes.Identity{}, http.StatusUnauthorized
	}
	// Ops matches only this server-authenticated UUID against the deployment's owner policy.
	return hermes.Identity{KratosID: session.Identity.Id}, 0
}

var consoleRequestID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var linearTeamKey = regexp.MustCompile(`^[A-Za-z0-9]{1,16}$`)

func (s hermesSettings) valid() bool {
	action := s.Action
	s.Action = ""
	switch action {
	case "configure_telegram":
		if !validHermesSecret(s.BotToken) {
			return false
		}
		s.BotToken = ""
	case "configure_gitlab":
		if !validHermesSecret(s.GitLabToken) {
			return false
		}
		s.GitLabToken = ""
	case "configure_linear":
		if !validHermesSecret(s.LinearToken) || !linearTeamKey.MatchString(s.LinearTeam) {
			return false
		}
		s.LinearToken, s.LinearTeam = "", ""
	case "console", "console_result":
		if !consoleRequestID.MatchString(s.RequestID) {
			return false
		}
		s.RequestID = ""
		if action == "console" {
			if strings.TrimSpace(s.Command) == "" || len(s.Command) > 4096 || strings.ContainsAny(s.Command, "\r\n\x00\x7f") ||
				(s.CoverBase64 != "" && (len(s.CoverBase64) > 700000 || !strings.HasPrefix(strings.ToLower(s.Command), "event "))) {
				return false
			}
			s.CoverBase64 = ""
			s.Command = ""
		}
	case "poll_chatgpt":
		if strings.TrimSpace(s.FlowID) == "" || len(s.FlowID) > 128 {
			return false
		}
		s.FlowID = ""
	case "claim", "bind_telegram", "disconnect_telegram", "disconnect_linear", "test", "connect_chatgpt":
	default:
		return false
	}
	return s == (hermesSettings{})
}

func validHermesSecret(value string) bool {
	return len(value) > 0 && len(value) <= 4096 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, " \t\r\n")
}

func writeHermesDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeHermesError(w, http.StatusRequestEntityTooLarge)
		return
	}
	writeHermesError(w, http.StatusBadRequest)
}

func writeHermesDependencyError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	if errors.Is(err, hermes.ErrDenied) {
		status = http.StatusForbidden
	}
	if errors.Is(err, hermes.ErrInvalid) {
		status = http.StatusBadRequest
	}
	writeHermesError(w, status)
}

func writeHermesError(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}
