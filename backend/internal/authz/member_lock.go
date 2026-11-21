package authz

import (
	"context"
	"fmt"
	"github.com/uptrace/bun"
)

// The two-key advisory lock space is reserved for (organization,user) member
// projections. It remains stable through deletion and re-addition of a row.
func LockOrganizationMember(ctx context.Context, tx bun.IDB, orgID, userID int32) error {
	if _, err := tx.NewRaw("SELECT pg_advisory_xact_lock(?,?)", orgID, userID).Exec(ctx); err != nil {
		return fmt.Errorf("lock member projection: %w", err)
	}
	return nil
}
