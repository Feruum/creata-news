package authz

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/uptrace/bun"
	"log/slog"
)

// platformGrantCurrent gates graph grants immediately after a durable role change.
// Users without a desired-state row retain legacy manually provisioned roles.
func (e *Enforcer) platformGrantCurrentResult(ctx context.Context, userID int32, admin bool) (bool, error) {
	var role string
	err := e.db.NewRaw("SELECT role FROM platform_user_roles WHERE user_id = ?", userID).Scan(ctx, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "read desired platform role", "user_id", userID, "error", err)
		return false, fmt.Errorf("read desired platform role: %w", err)
	}
	if admin {
		return role == "administrator", nil
	}
	return role == "administrator" || role == "staff", nil
}

func (e *Enforcer) platformGrantCurrent(ctx context.Context, userID int32, admin bool) bool {
	current, err := e.platformGrantCurrentResult(ctx, userID, admin)
	return err == nil && current
}

func (e *Enforcer) checkPlatformStaffResult(ctx context.Context, userID int32) (bool, error) {
	if e.perms == nil {
		return false, errors.New("authorization service unavailable")
	}
	current, err := e.platformGrantCurrentResult(ctx, userID, false)
	if err != nil || !current {
		return false, err
	}
	staff, err := e.perms.IsGlobalStaff(ctx, fmt.Sprint(userID))
	if err != nil || staff {
		return staff, err
	}
	adminCurrent, err := e.platformGrantCurrentResult(ctx, userID, true)
	if err != nil || !adminCurrent {
		return false, err
	}
	return e.perms.CheckPermission(ctx, fmt.Sprint(userID), "platform", e.perms.GetPlatformID(), "manage_system")
}

// SyncPlatformUserRole reads the latest desired state while holding the same
// user lock as assignments, preventing an older reconciliation overwriting one.
func (e *Enforcer) SyncPlatformUserRole(ctx context.Context, userID int32) error {
	if e.perms == nil {
		return errors.New("authorization service unavailable")
	}
	return e.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var lockedID int32
		if err := tx.NewRaw("SELECT id FROM users WHERE id = ? FOR UPDATE", userID).Scan(ctx, &lockedID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("lock platform role user: %w", err)
		}
		var role string
		if err := tx.NewRaw("SELECT role FROM platform_user_roles WHERE user_id = ?", userID).Scan(ctx, &role); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("read desired platform role: %w", err)
		}
		if role == "none" {
			role = ""
		}
		return e.perms.SetPlatformRole(ctx, fmt.Sprintf("%d", userID), role)
	})
}

// StoreDesiredPlatformRole is used inside a caller's user-locked transaction.
func StoreDesiredPlatformRole(ctx context.Context, tx bun.IDB, userID int32, role string) error {
	if role == "" {
		role = "none"
	}
	if role != "none" && role != "staff" && role != "administrator" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported platform role"))
	}
	if _, err := tx.NewRaw("INSERT INTO platform_user_roles(user_id,role) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET role=EXCLUDED.role,updated_at=NOW()", userID, role).Exec(ctx); err != nil {
		return fmt.Errorf("store desired platform role: %w", err)
	}
	return events.EnqueuePlatformRoleSync(ctx, tx, userID)
}

// AssignPlatformRole requires the caller to authorize assignment first. It is
// shared by the protected bootstrap endpoint and administrator RPC.
func (e *Enforcer) AssignPlatformRole(ctx context.Context, userID int32, role string) error {
	if role != "" && role != "none" && role != "staff" && role != "administrator" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported platform role"))
	}
	err := e.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var lockedID int32
		if err := tx.NewRaw("SELECT id FROM users WHERE id=? FOR UPDATE", userID).Scan(ctx, &lockedID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return connect.NewError(connect.CodeNotFound, errors.New("user not found"))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("lock platform user: %w", err))
		}
		if err := StoreDesiredPlatformRole(ctx, tx, userID, role); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := e.SyncPlatformUserRole(ctx, userID); err != nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("reconcile platform role: %w", err))
	}
	return nil
}
