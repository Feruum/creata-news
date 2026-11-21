package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	ory "github.com/ory/kratos-client-go"
	"github.com/studyverse/ems-backend/internal/telemetry"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserIDKey is the context key for the authenticated user ID
	UserIDKey ContextKey = "user_id"
	// UserEmailKey is the context key for the authenticated user email
	UserEmailKey ContextKey = "user_email"
	// UserFirstNameKey is the context key for the authenticated user's first name
	UserFirstNameKey ContextKey = "user_first_name"
	// UserLastNameKey is the context key for the authenticated user's last name
	UserLastNameKey ContextKey = "user_last_name"
	// SessionKey is the context key for the full session
	SessionKey ContextKey = "session"
	// Cookie-only validation reports dependency failures separately from invalid sessions.
	sessionValidationUnavailableKey ContextKey = "session_validation_unavailable"
)

// NewMiddleware creates an HTTP middleware that validates Kratos sessions
// and injects the user ID into the request context.
//
// Expected auth mechanisms:
// - Browser: Kratos session cookie forwarded automatically by the browser.
// - Native/CLI: X-Session-Token: <token> or Authorization: Bearer <token>.
//
// If the user is not authenticated, the request proceeds without user context
// (public endpoints still work; protected endpoints should call RequireAuth).
func NewMiddleware(kratosClient *ory.APIClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			cookie := r.Header.Get("Cookie")
			sessionToken := r.Header.Get("X-Session-Token")
			if sessionToken == "" {
				sessionToken = extractBearerToken(r.Header.Get("Authorization"))
			}

			if cookie == "" && sessionToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			slog.Debug("Auth middleware", "hasCookie", cookie != "", "hasToken", sessionToken != "", "path", r.URL.Path)

			toSession := kratosClient.FrontendAPI.ToSession(ctx)
			if cookie != "" {
				toSession = toSession.Cookie(cookie)
			}
			if sessionToken != "" {
				toSession = toSession.XSessionToken(sessionToken)
			}

			session, resp, err := toSession.Execute()
			if err != nil {
				status := 0
				if resp != nil {
					status = resp.StatusCode
				}
				slog.Warn("Kratos session validation failed", "error", err, "status", status, "path", r.URL.Path, "hasCookie", cookie != "", "hasToken", sessionToken != "")
				ctx = context.WithValue(ctx, sessionValidationUnavailableKey, status != http.StatusUnauthorized && status != http.StatusForbidden)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if session.Identity == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx = context.WithValue(ctx, UserIDKey, session.Identity.Id)

			if traits, ok := session.Identity.Traits.(map[string]interface{}); ok {
				if email, ok := traits["email"].(string); ok {
					ctx = context.WithValue(ctx, UserEmailKey, email)
				}
				if name, ok := traits["name"].(map[string]interface{}); ok {
					if first, ok := name["first"].(string); ok && first != "" {
						ctx = context.WithValue(ctx, UserFirstNameKey, first)
					}
					if last, ok := name["last"].(string); ok && last != "" {
						ctx = context.WithValue(ctx, UserLastNameKey, last)
					}
				}
			}

			ctx = context.WithValue(ctx, SessionKey, session)

			slog.Debug("Authenticated request", "user_id", session.Identity.Id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewCookieMiddleware always revalidates browser cookies and ignores token headers.
// Existing auth context is cleared so a failed fresh check cannot reuse an earlier identity.
func NewCookieMiddleware(kratosClient *ory.APIClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), UserIDKey, "")
			ctx = context.WithValue(ctx, UserEmailKey, "")
			ctx = context.WithValue(ctx, UserFirstNameKey, "")
			ctx = context.WithValue(ctx, UserLastNameKey, "")
			ctx = context.WithValue(ctx, SessionKey, (*ory.Session)(nil))
			ctx = context.WithValue(ctx, sessionValidationUnavailableKey, false)
			cookie := r.Header.Get("Cookie")
			if cookie == "" {
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if kratosClient == nil {
				ctx = context.WithValue(ctx, sessionValidationUnavailableKey, true)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			session, response, err := kratosClient.FrontendAPI.ToSession(ctx).Cookie(cookie).Execute()
			if err != nil {
				unavailable := response == nil || (response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden)
				ctx = context.WithValue(ctx, sessionValidationUnavailableKey, unavailable)
			} else if session != nil && session.Identity != nil {
				ctx = context.WithValue(ctx, UserIDKey, session.Identity.Id)
				ctx = context.WithValue(ctx, SessionKey, session)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IsSessionValidationUnavailable(ctx context.Context) bool {
	unavailable, _ := ctx.Value(sessionValidationUnavailableKey).(bool)
	return unavailable
}

// extractBearerToken extracts the token from "Bearer <token>" format
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

// GetUserID extracts the user ID from the context
// Returns empty string if not authenticated
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetUserEmail extracts the user email from the context
// Returns empty string if not available
func GetUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}

// GetUserFirstName extracts the user's first name from the context
func GetUserFirstName(ctx context.Context) string {
	if name, ok := ctx.Value(UserFirstNameKey).(string); ok {
		return name
	}
	return ""
}

// GetUserLastName extracts the user's last name from the context
func GetUserLastName(ctx context.Context) string {
	if name, ok := ctx.Value(UserLastNameKey).(string); ok {
		return name
	}
	return ""
}

// IsAuthenticated checks if the request has a valid authenticated user
func IsAuthenticated(ctx context.Context) bool {
	return GetUserID(ctx) != ""
}

// GetSession returns the full Ory session from context
func GetSession(ctx context.Context) *ory.Session {
	if session, ok := ctx.Value(SessionKey).(*ory.Session); ok {
		return session
	}
	return nil
}

// RequireAuth is a helper that returns an error if the user is not authenticated
// Use this in service methods that require authentication
func RequireAuth(ctx context.Context) (string, error) {
	userID := GetUserID(ctx)
	if userID == "" {
		return "", ErrUnauthenticated
	}
	return userID, nil
}

func NewKratosClient(kratosPublicURL string) *ory.APIClient {
	config := ory.NewConfiguration()
	config.Servers = ory.ServerConfigurations{
		{URL: kratosPublicURL},
	}
	config.HTTPClient = telemetry.NewHTTPClient()
	return ory.NewAPIClient(config)
}
