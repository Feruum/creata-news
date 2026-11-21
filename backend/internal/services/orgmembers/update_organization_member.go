package orgmembers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) UpdateOrganizationMember(ctx context.Context, req *connect.Request[eventsv1.UpdateOrganizationMemberRequest]) (*connect.Response[eventsv1.UpdateOrganizationMemberResponse], error) {
	slog.Debug("UpdateOrganizationMember", "id", req.Msg.Id)

	var existingMember organizationMemberRow
	err := s.db.NewSelect().
		Model(&existingMember).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.id = ?", req.Msg.Id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	actorID, err := s.authz.RequireOrgPermission(ctx, existingMember.OrganizationID, "manage_members")
	if err != nil {
		return nil, err
	}

	roleID := existingMember.RoleTemplateID
	if req.Msg.RoleTemplateId != nil {
		roleID = *req.Msg.RoleTemplateId
	}
	var expiry *time.Time
	if req.Msg.ValidUntil != nil {
		expiry, err = parseMembershipExpiry(req.Msg.ValidUntil, time.Now())
		if err != nil {
			return nil, err
		}
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := authz.LockOrganizationMember(ctx, tx, existingMember.OrganizationID, existingMember.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.NewSelect().Model(&existingMember.OrganizationMember).Where("id=?", existingMember.ID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if req.Msg.RoleTemplateId == nil {
		roleID = existingMember.RoleTemplateID
	}
	if err := validateAssignmentInTransaction(ctx, tx, existingMember.OrganizationID, roleID, existingMember.ID); err != nil {
		return nil, err
	}
	query := tx.NewUpdate().Model((*models.OrganizationMember)(nil)).Where("id = ?", req.Msg.Id).Set("role_template_id = ?", roleID)
	if req.Msg.ValidUntil != nil {
		query = query.Set("valid_until = ?", expiry)
	}
	result, err := query.Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if affected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	updatedMember := existingMember.OrganizationMember
	updatedMember.RoleTemplateID = roleID
	if req.Msg.RoleTemplateId != nil && roleID != existingMember.RoleTemplateID {
		if err := s.enqueueMembershipMutation(ctx, tx, updatedMember, actorID, "updated"); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else if err := events.EnqueueMemberSync(ctx, tx, existingMember.OrganizationID, existingMember.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.authz.SyncOrganizationMember(ctx, existingMember.OrganizationID, existingMember.UserID); err != nil {
		slog.WarnContext(ctx, "Failed to reconcile updated member", "error", err)
	}
	var memberWithRole organizationMemberRow
	err = s.db.NewSelect().
		Model(&memberWithRole).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.id = ?", req.Msg.Id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.UpdateOrganizationMemberResponse{
		Member: dbOrganizationMemberToProto(memberWithRole),
	}), nil
}
