// Package hermes provides the private transport for the owner-only diagnostics service.
package hermes

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDenied      = errors.New("Hermes access denied")
	ErrUnavailable = errors.New("Hermes unavailable")
	ErrInvalid     = errors.New("invalid Hermes request")
)

type Identity struct {
	KratosID string `json:"kratos_id"`
}

type TelegramLink struct {
	ID       string    `json:"id"`
	LinkedAt time.Time `json:"linked_at"`
}

type TelegramLinkStore interface {
	VerifiedLink(context.Context, string) (TelegramLink, error)
	UserCount(context.Context) (int, error)
}

// Request contains only server-established identity and Telegram claims.
// Secret values are write-only and never part of Info.
type Request struct {
	Identity    Identity      `json:"identity"`
	Action      string        `json:"action,omitempty"`
	BotToken    string        `json:"bot_token,omitempty"`
	GitLabToken string        `json:"gitlab_token,omitempty"`
	LinearToken string        `json:"linear_token,omitempty"`
	LinearTeam  string        `json:"linear_team,omitempty"`
	Command     string        `json:"command,omitempty"`
	CoverBase64 string        `json:"cover_base64,omitempty"`
	RequestID   string        `json:"request_id,omitempty"`
	FlowID      string        `json:"flow_id,omitempty"`
	Telegram    *TelegramLink `json:"telegram,omitempty"`
}

type Info struct {
	Claimed            bool           `json:"claimed"`
	TelegramLinked     bool           `json:"telegram_linked"`
	TelegramConfigured bool           `json:"telegram_configured"`
	GitLabConfigured   bool           `json:"gitlab_configured"`
	ChatGPTConnected   bool           `json:"chatgpt_connected"`
	Ready              bool           `json:"ready"`
	TelegramID         string         `json:"telegram_id,omitempty"`
	TelegramGroupID    string         `json:"telegram_group_id,omitempty"`
	LinearConfigured   bool           `json:"linear_configured"`
	LinearTeam         string         `json:"linear_team,omitempty"`
	Console            *ConsoleResult `json:"console,omitempty"`
	ModelHealth        *ModelHealth   `json:"model_health,omitempty"`
	Model              string         `json:"model,omitempty"`
	LastTest           *TestResult    `json:"last_test,omitempty"`
	OAuth              *OAuthFlow     `json:"oauth,omitempty"`
	RecentIncidents    []Incident     `json:"recent_incidents,omitempty"`
}

type ConsoleResult struct {
	Status    string `json:"status"`
	Output    string `json:"output"`
	RequestID string `json:"request_id,omitempty"`
}

type ModelHealth struct {
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	CheckedAt int64  `json:"checked_at"`
}

type TestResult struct {
	OK bool `json:"ok"`
}

type OAuthFlow struct {
	FlowID          string `json:"flow_id"`
	VerificationURL string `json:"verification_url"`
	UserCode        string `json:"user_code"`
	ExpiresAt       *int64 `json:"expires_at,omitempty"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

type Incident struct {
	ID          int64  `json:"id"`
	Source      string `json:"source"`
	Environment string `json:"environment"`
	Service     string `json:"service"`
	Severity    int    `json:"severity"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Result      string `json:"result,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}
