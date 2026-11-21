package sundayaccount

import (
	"context"
	"errors"

	"github.com/uptrace/bun"
)

var errInvalidCredentials = errors.New("invalid credentials")

func lockEmail(ctx context.Context, tx bun.Tx, email string) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "sunday-email:"+email)
	return err
}

// Revalidate a browser session inside the mutation transaction, retaining the
// account lock that serializes registration, attendance and session rotation.
func (h *Handler) lockSessionAccount(ctx context.Context, tx bun.Tx, p *principal) (*Account, error) {
	var current Account
	if err := tx.NewRaw(`SELECT `+accountProjection("")+` FROM sunday_open_accounts WHERE id=? FOR NO KEY UPDATE`, p.ID).Scan(ctx, &current); err != nil {
		return nil, err
	}
	var active bool
	if err := tx.NewRaw(`SELECT EXISTS(SELECT 1 FROM sunday_open_sessions WHERE account_id=? AND token_hash=? AND expires_at>NOW())`, p.ID, h.digest("session", p.token)).Scan(ctx, &active); err != nil {
		return nil, err
	}
	if !active {
		return nil, errInvalidCredentials
	}
	return &current, nil
}
