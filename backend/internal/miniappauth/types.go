package miniappauth

import (
	"context"
	"errors"
)

const ClientID = "aitumap-miniapp"
const ProviderID = "telegram-miniapp"
const launchCookie = "aitumap_telegram_login"

var ErrExpired = errors.New("expired or consumed Telegram sign-in ticket")
var ErrNotLinked = errors.New("no confirmed Telegram account link")
var ErrUnavailable = errors.New("Telegram sign-in is temporarily unavailable")

type Config struct {
	Issuer        string
	RedirectURI   string
	ClientSecret  string
	PrivateKeyB64 string
	BotToken      string
}
type Link struct {
	IdentityID string `json:"identityId"`
	TelegramID int64  `json:"telegramId"`
	Subject    string `json:"subject"`
}
type Grant struct {
	Link            Link   `json:"link"`
	AuthenticatedAt int64  `json:"authenticatedAt"`
	Nonce           string `json:"nonce,omitempty"`
	CodeChallenge   string `json:"codeChallenge,omitempty"`
}
type Store interface {
	ReserveLaunch(context.Context, string) (bool, error)
	Put(context.Context, string, string, Grant) error
	Read(context.Context, string, string) (*Grant, error)
	Consume(context.Context, string, string) (*Grant, error)
}
type Resolver interface {
	Resolve(context.Context, int64) (*Link, error)
	Check(context.Context, Link) error
	EnsureCredential(context.Context, Link) error
	CurrentIdentity(context.Context, string) (string, error)
}
