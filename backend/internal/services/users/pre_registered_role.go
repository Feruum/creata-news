package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) applyPreRegisteredRoleInTx(ctx context.Context, tx bun.Tx, email string, userID int32) (bool, error) {
	var lockedID int32
	if err := tx.NewRaw("SELECT id FROM users WHERE id=? FOR UPDATE", userID).Scan(ctx, &lockedID); err != nil {
		return false, fmt.Errorf("lock invitation user: %w", err)
	}
	var invite models.PreRegisteredUser
	if err := tx.NewSelect().Model(&invite).Where("email=?", email).Where("used_at IS NULL").For("UPDATE").Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("read unused invitation: %w", err)
	}
	role := ""
	switch invite.PlatformRole {
	case models.PlatformRoleAdmin:
		role = "administrator"
	case models.PlatformRoleStaff:
		role = "staff"
	default:
		return false, errors.New("unsupported invitation platform role")
	}
	var assigned bool
	if err := tx.NewRaw("SELECT EXISTS(SELECT 1 FROM platform_user_roles WHERE user_id=?)", userID).Scan(ctx, &assigned); err != nil {
		return false, fmt.Errorf("read assigned platform role: %w", err)
	}
	// An explicit assignment, including revocation, is authoritative over invites.
	if !assigned {
		if err := authz.StoreDesiredPlatformRole(ctx, tx, userID, role); err != nil {
			return false, err
		}
	} else if err := events.EnqueuePlatformRoleSync(ctx, tx, userID); err != nil {
		return false, err
	}
	if _, err := tx.NewUpdate().Model(&invite).Set("used_at=NOW()").Set("used_by_user_id=?", userID).Set("updated_at=NOW()").WherePK().Exec(ctx); err != nil {
		return false, fmt.Errorf("consume platform invitation: %w", err)
	}
	return true, nil
}
func (s *Service) applyPreRegisteredRole(ctx context.Context, email string, userID int32) error {
	var changed bool
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var err error
		changed, err = s.applyPreRegisteredRoleInTx(ctx, tx, email, userID)
		return err
	})
	if err != nil {
		return err
	}
	if changed {
		return s.authz.SyncPlatformUserRole(ctx, userID)
	}
	return nil
}
