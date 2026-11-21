package orgmembers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/samber/lo"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) ListOrganizationMembers(ctx context.Context, req *connect.Request[eventsv1.ListOrganizationMembersRequest]) (*connect.Response[eventsv1.ListOrganizationMembersResponse], error) {
	slog.Debug("ListOrganizationMembers", "orgId", req.Msg.OrganizationId)

	var members []organizationMemberRow
	err := s.db.NewSelect().
		Model(&members).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.organization_id = ?", req.Msg.OrganizationId).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total, err := s.db.NewSelect().
		Model((*models.OrganizationMember)(nil)).
		Where("organization_id = ?", req.Msg.OrganizationId).
		Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoMembers := lo.Map(members, func(m organizationMemberRow, _ int) *eventsv1.OrganizationMember {
		return dbOrganizationMemberToProto(m)
	})

	return connect.NewResponse(&eventsv1.ListOrganizationMembersResponse{
		Members: protoMembers,
		Total:   int32(total),
	}), nil
}
