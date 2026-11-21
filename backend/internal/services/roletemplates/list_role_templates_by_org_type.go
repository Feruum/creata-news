package roletemplates

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) ListRoleTemplatesByOrgType(ctx context.Context, req *connect.Request[eventsv1.ListRoleTemplatesByOrgTypeRequest]) (*connect.Response[eventsv1.ListRoleTemplatesByOrgTypeResponse], error) {
	slog.Debug("ListRoleTemplatesByOrgType", "orgTypeId", req.Msg.OrganizationTypeId)

	var templates []models.RoleTemplate
	err := s.db.NewSelect().
		Model(&templates).
		Where("organization_type_id = ?", req.Msg.OrganizationTypeId).
		Where("organization_id IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTemplates := make([]*eventsv1.RoleTemplate, len(templates))
	for i, t := range templates {
		var caps []models.RoleTemplateCapability
		_ = s.db.NewSelect().Model(&caps).Where("role_template_id = ?", t.ID).Scan(ctx)
		protoTemplates[i] = dbRoleTemplateToProto(t, caps)
	}

	return connect.NewResponse(&eventsv1.ListRoleTemplatesByOrgTypeResponse{
		RoleTemplates: protoTemplates,
	}), nil
}
