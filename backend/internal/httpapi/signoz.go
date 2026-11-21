package httpapi

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ory "github.com/ory/kratos-client-go"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/config"
)

type PlatformAdminAuthorizer interface {
	RequirePlatformAdmin(context.Context) (int32, error)
}

type sigNozInfo struct {
	Configured bool         `json:"configured"`
	URL        string       `json:"url,omitempty"`
	Login      *sigNozLogin `json:"login,omitempty"`
}

type sigNozLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// WithSigNozRoutes selects only the two SigNoz endpoints before the ordinary auth/router chain.
// SigNoz's own bearer/session-token headers must never participate in Studyverse authentication.
func WithSigNozRoutes(next http.Handler, cfg *config.Config, admin PlatformAdminAuthorizer, kratos *ory.APIClient) http.Handler {
	gate := auth.NewCookieMiddleware(kratos)(NewSigNozHandler(cfg, admin))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/signoz/access" || r.URL.Path == "/admin/signoz/info" {
			gate.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewSigNozHandler requires a freshly verified cookie session supplied by NewCookieMiddleware.
// It deliberately uses no body logging, response instrumentation, or persisted credential state.
func NewSigNozHandler(cfg *config.Config, admin PlatformAdminAuthorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeSigNozError(w, http.StatusMethodNotAllowed)
			return
		}
		if auth.IsSessionValidationUnavailable(r.Context()) {
			writeSigNozError(w, http.StatusServiceUnavailable)
			return
		}
		session := auth.GetSession(r.Context())
		userID := auth.GetUserID(r.Context())
		if session == nil || !session.GetActive() || !session.GetExpiresAt().After(time.Now()) ||
			session.Identity == nil || userID == "" || session.Identity.Id != userID {
			writeSigNozError(w, http.StatusUnauthorized)
			return
		}
		if admin == nil {
			writeSigNozError(w, http.StatusServiceUnavailable)
			return
		}
		if _, err := admin.RequirePlatformAdmin(r.Context()); err != nil {
			status := http.StatusServiceUnavailable
			switch connect.CodeOf(err) {
			case connect.CodeUnauthenticated:
				status = http.StatusUnauthorized
			case connect.CodePermissionDenied:
				status = http.StatusForbidden
			}
			writeSigNozError(w, status)
			return
		}
		if r.URL.Path == "/admin/signoz/info" && r.Header.Get("X-Studyverse-SigNoz") != "1" {
			writeSigNozError(w, http.StatusForbidden)
			return
		}
		origin, err := cfg.SigNozOrigin()
		if err != nil {
			writeSigNozError(w, http.StatusServiceUnavailable)
			return
		}
		switch r.URL.Path {
		case "/admin/signoz/access":
			if origin == "" {
				writeSigNozError(w, http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case "/admin/signoz/info":
			info := sigNozInfo{Configured: origin != "", URL: origin}
			if info.Configured && cfg.SigNozLoginEmail != "" && cfg.SigNozLoginPassword != "" {
				info.Login = &sigNozLogin{Email: cfg.SigNozLoginEmail, Password: cfg.SigNozLoginPassword}
			}
			writeJSON(w, http.StatusOK, info)
		default:
			writeSigNozError(w, http.StatusNotFound)
		}
	})
}

func writeSigNozError(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}
