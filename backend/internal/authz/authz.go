package authz

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
)

type Enforcer struct {
	db    bun.IDB
	perms *perms.Client
}

func NewEnforcer(db *bun.DB, permsClient *perms.Client) *Enforcer {
	e := &Enforcer{perms: permsClient}
	if db != nil {
		e.db = db
	}
	return e
}

// InTransaction keeps permission reads on the mutation's connection. Concurrent
// writers waiting on locks must not exhaust the pool needed by the lock holder.
func (e *Enforcer) InTransaction(tx bun.Tx) *Enforcer {
	if e == nil {
		return nil
	}
	return &Enforcer{db: tx, perms: e.perms}
}

func (e *Enforcer) GetLocalUserID(ctx context.Context) (int32, error) {
	kratosUserID := auth.GetUserID(ctx)
	if kratosUserID == "" {
		return 0, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	if e.db == nil {
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("user directory unavailable"))
	}

	user := &models.User{}
	err := e.db.NewSelect().Model(user).Column("id").Where("kratos_id = ?", kratosUserID).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not found"))
		}
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("user directory unavailable"))
	}

	return user.ID, nil
}

func (e *Enforcer) RequirePlatformAdmin(ctx context.Context) (int32, error) {
	localUserID, err := e.GetLocalUserID(ctx)
	if err != nil {
		return 0, err
	}

	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying platform admin check", "user_id", localUserID)
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}

	current, err := e.platformGrantCurrentResult(ctx, localUserID, true)
	if err != nil {
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}
	if !current {
		return 0, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("platform administrator access required"))
	}
	allowed, err := e.perms.CheckPermission(ctx, fmt.Sprint(localUserID), "platform", e.perms.GetPlatformID(), "manage_system")
	if err != nil {
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}
	if !allowed {
		return 0, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("platform administrator access required"))
	}

	return localUserID, nil
}

func (e *Enforcer) RequirePlatformStaff(ctx context.Context) (int32, error) {
	localUserID, err := e.GetLocalUserID(ctx)
	if err != nil {
		return 0, err
	}

	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying platform staff check", "user_id", localUserID)
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}

	if !e.CheckPlatformStaff(ctx, localUserID) {
		return 0, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("platform staff access required"))
	}

	return localUserID, nil
}

func (e *Enforcer) RequireOrgPermission(ctx context.Context, orgID int32, permission string) (int32, error) {
	localUserID, err := e.GetLocalUserID(ctx)
	if err != nil {
		return 0, err
	}

	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying org permission check", "user_id", localUserID, "org_id", orgID, "permission", permission)
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}

	allowed := e.CheckOrgPermission(ctx, localUserID, orgID, permission)
	if !allowed {
		return 0, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission '%s' required for this organization", permission))
	}

	return localUserID, nil
}

func (e *Enforcer) RequireResourcePermission(ctx context.Context, resourceType, resourceID, permission string) (int32, error) {
	localUserID, err := e.GetLocalUserID(ctx)
	if err != nil {
		return 0, err
	}

	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying resource permission check", "user_id", localUserID, "resource", resourceType+":"+resourceID, "permission", permission)
		return 0, connect.NewError(connect.CodeUnavailable, fmt.Errorf("authorization service unavailable"))
	}

	allowed := e.CheckResourcePermission(ctx, localUserID, resourceType, resourceID, permission)
	if !allowed {
		return 0, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission '%s' required on %s:%s", permission, resourceType, resourceID))
	}

	return localUserID, nil
}

func (e *Enforcer) CheckPlatformAdmin(ctx context.Context, localUserID int32) bool {
	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying CheckPlatformAdmin", "user_id", localUserID)
		return false
	}

	userIDStr := fmt.Sprintf("%d", localUserID)
	if !e.platformGrantCurrent(ctx, localUserID, true) {
		return false
	}
	isAdmin, _ := e.perms.CheckPermission(ctx, userIDStr, "platform", e.perms.GetPlatformID(), "manage_system")
	return isAdmin
}

func (e *Enforcer) CheckPlatformStaff(ctx context.Context, localUserID int32) bool {
	if e.perms == nil {
		slog.Error("SpiceDB client unavailable - denying CheckPlatformStaff", "user_id", localUserID)
		return false
	}

	userIDStr := fmt.Sprintf("%d", localUserID)
	if !e.platformGrantCurrent(ctx, localUserID, false) {
		return false
	}
	isStaff, _ := e.perms.IsGlobalStaff(ctx, userIDStr)
	return isStaff || e.CheckPlatformAdmin(ctx, localUserID)
}

func (e *Enforcer) CheckOrgPermission(ctx context.Context, localUserID, orgID int32, permission string) bool {
	allowed, err := e.CheckOrgPermissionWithError(ctx, localUserID, orgID, permission)
	return err == nil && allowed
}

// CheckOrgPermissionWithError distinguishes revoked grants from retryable
// authorization failures for durable delivery workers.
func (e *Enforcer) CheckOrgPermissionWithError(ctx context.Context, localUserID, orgID int32, permission string) (bool, error) {
	if e.perms == nil {
		return false, fmt.Errorf("authorization service unavailable")
	}
	staff, err := e.checkPlatformStaffResult(ctx, localUserID)
	if err != nil || staff {
		return staff, err
	}
	current, err := e.orgGrantCurrentResult(ctx, localUserID, orgID, permission)
	if err != nil || !current {
		return false, err
	}
	return e.perms.CheckPermission(ctx, fmt.Sprint(localUserID), "organization", fmt.Sprint(orgID), permission)
}
