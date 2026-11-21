package orgmembers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) RemoveOrganizationMemberByUserAndOrg(ctx context.Context, req *connect.Request[eventsv1.RemoveOrganizationMemberByUserAndOrgRequest]) (*connect.Response[eventsv1.RemoveOrganizationMemberByUserAndOrgResponse], error) {
	slog.Debug("RemoveOrganizationMemberByUserAndOrg", "userId", req.Msg.UserId, "orgId", req.Msg.OrganizationId)

	actorID, err := s.authz.RequireOrgPermission(ctx, req.Msg.OrganizationId, "manage_members")
	if err != nil {
		return nil, err
	}

	var member organizationMemberRow
	err = s.db.NewSelect().
		Model(&member).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.user_id = ?", req.Msg.UserId).
		Where("om.organization_id = ?", req.Msg.OrganizationId).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := authz.LockOrganizationMember(ctx, tx, member.OrganizationID, member.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.NewSelect().Model(&member.OrganizationMember).Where("id=?", member.ID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := tx.NewDelete().
		Model((*models.OrganizationMember)(nil)).
		Where("user_id = ?", req.Msg.UserId).
		Where("organization_id = ?", req.Msg.OrganizationId).
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if rowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}

	if err := s.enqueueMembershipMutation(ctx, tx, member.OrganizationMember, actorID, "removed"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.removeMemberFromSpiceDB(ctx, req.Msg.OrganizationId, req.Msg.UserId); err != nil {
		slog.WarnContext(ctx, "Failed to remove member from SpiceDB", "error", err)
	}

	// Publish domain event
	return connect.NewResponse(&eventsv1.RemoveOrganizationMemberByUserAndOrgResponse{
		Success: true,
	}), nil
}
