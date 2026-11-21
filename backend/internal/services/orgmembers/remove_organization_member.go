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

func (s *Service) RemoveOrganizationMember(ctx context.Context, req *connect.Request[eventsv1.RemoveOrganizationMemberRequest]) (*connect.Response[eventsv1.RemoveOrganizationMemberResponse], error) {
	slog.Debug("RemoveOrganizationMember", "id", req.Msg.Id)

	var member organizationMemberRow
	err := s.db.NewSelect().
		Model(&member).
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

	actorID, permErr := s.authz.RequireOrgPermission(ctx, member.OrganizationID, "manage_members")
	if permErr != nil {
		return nil, permErr
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
		Where("id = ?", req.Msg.Id).
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
	if err := s.removeMemberFromSpiceDB(ctx, member.OrganizationID, member.UserID); err != nil {
		slog.WarnContext(ctx, "Failed to remove member from SpiceDB", "error", err)
	}

	return connect.NewResponse(&eventsv1.RemoveOrganizationMemberResponse{
		Success: true,
	}), nil
}
