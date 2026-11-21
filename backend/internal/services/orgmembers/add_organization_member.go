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
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) AddOrganizationMember(ctx context.Context, req *connect.Request[eventsv1.AddOrganizationMemberRequest]) (*connect.Response[eventsv1.AddOrganizationMemberResponse], error) {
	slog.Debug("AddOrganizationMember", "orgId", req.Msg.OrganizationId, "userId", req.Msg.UserId, "roleTemplateId", req.Msg.RoleTemplateId)

	actorID, err := s.authz.RequireOrgPermission(ctx, req.Msg.OrganizationId, "manage_members")
	if err != nil {
		return nil, err
	}

	member := models.OrganizationMember{
		OrganizationID: req.Msg.OrganizationId,
		UserID:         req.Msg.UserId,
		RoleTemplateID: req.Msg.RoleTemplateId,
		AssignedAt:     time.Now(),
	}

	member.AssignedBy = &actorID
	expiry, err := parseMembershipExpiry(req.Msg.ValidUntil, time.Now())
	if err != nil {
		return nil, err
	}
	member.ValidUntil = expiry
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := authz.LockOrganizationMember(ctx, tx, member.OrganizationID, member.UserID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := validateAssignmentInTransaction(ctx, tx, member.OrganizationID, member.RoleTemplateID, 0); err != nil {
		return nil, err
	}
	if _, err := tx.NewInsert().Model(&member).Returning("*").Exec(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.enqueueMembershipMutation(ctx, tx, member, actorID, "added"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.syncMemberCapabilitiesToSpiceDB(ctx, member.OrganizationID, member.UserID, member.RoleTemplateID); err != nil {
		slog.WarnContext(ctx, "Failed to sync member capabilities to SpiceDB", "error", err)
	}

	var memberWithRole organizationMemberRow
	err = s.db.NewSelect().
		Model(&memberWithRole).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.id = ?", member.ID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.AddOrganizationMemberResponse{
		Member: dbOrganizationMemberToProto(memberWithRole),
	}), nil
}
