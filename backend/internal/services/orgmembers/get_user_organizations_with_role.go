package orgmembers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetUserOrganizationsWithRole(ctx context.Context, req *connect.Request[eventsv1.GetUserOrganizationsWithRoleRequest]) (*connect.Response[eventsv1.GetUserOrganizationsWithRoleResponse], error) {
	slog.Debug("GetUserOrganizationsWithRole", "userId", req.Msg.UserId)

	var rows []userOrgRow
	err := s.db.NewSelect().
		ColumnExpr("om.organization_id").
		ColumnExpr("o.title AS organization_title").
		ColumnExpr("o.organization_type_id").
		ColumnExpr("o.status").
		ColumnExpr("o.created_at").
		ColumnExpr("o.updated_at").
		ColumnExpr("o.image_url").
		ColumnExpr("o.description").
		ColumnExpr("o.instagram").
		ColumnExpr("o.telegram_channel").
		ColumnExpr("o.telegram_chat").
		ColumnExpr("o.website").
		ColumnExpr("o.youtube").
		ColumnExpr("o.tiktok").
		ColumnExpr("o.linkedin").
		ColumnExpr("os.slug AS organization_slug").
		ColumnExpr("om.role_template_id").
		ColumnExpr("rt.key AS role_key").
		ColumnExpr("rt.display_name AS role_display_name").
		TableExpr("organization_members AS om").
		Join("JOIN organizations AS o ON o.id = om.organization_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Join("LEFT JOIN organization_slugs AS os ON os.organization_id = o.id AND os.is_current = true").
		Where("om.user_id = ?", req.Msg.UserId).
		Where("om.valid_until IS NULL OR om.valid_until > CURRENT_TIMESTAMP").
		Where("rt.is_active = true").
		Where("rt.organization_type_id = o.organization_type_id").
		Where("rt.organization_id IS NULL OR rt.organization_id = o.id").
		Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoOrgs := lo.Map(rows, func(o userOrgRow, _ int) *eventsv1.UserOrganizationWithRole {
		return &eventsv1.UserOrganizationWithRole{
			Organization:    dbUserOrgRowToProto(o),
			RoleTemplateId:  o.RoleTemplateID,
			RoleKey:         o.RoleKey,
			RoleDisplayName: o.RoleDisplayName,
		}
	})

	return connect.NewResponse(&eventsv1.GetUserOrganizationsWithRoleResponse{
		Organizations: protoOrgs,
	}), nil
}
