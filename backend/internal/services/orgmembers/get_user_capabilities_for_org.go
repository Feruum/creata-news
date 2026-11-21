package orgmembers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetUserCapabilitiesForOrg(ctx context.Context, req *connect.Request[eventsv1.GetUserCapabilitiesForOrgRequest]) (*connect.Response[eventsv1.GetUserCapabilitiesForOrgResponse], error) {
	slog.Debug("GetUserCapabilitiesForOrg", "userId", req.Msg.UserId, "orgId", req.Msg.OrganizationId)

	var caps []models.RoleTemplateCapability
	err := s.db.NewSelect().
		Model(&caps).
		Join("JOIN organization_members AS om ON om.role_template_id = rc.role_template_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Join("JOIN organizations AS o ON o.id = om.organization_id").
		Where("om.valid_until IS NULL OR om.valid_until > CURRENT_TIMESTAMP").
		Where("rt.is_active = true").
		Where("rt.organization_type_id = o.organization_type_id").
		Where("rt.organization_id IS NULL OR rt.organization_id = o.id").
		Where("om.user_id = ?", req.Msg.UserId).
		Where("om.organization_id = ?", req.Msg.OrganizationId).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoCaps := lo.Map(caps, func(c models.RoleTemplateCapability, _ int) eventsv1.Capability {
		return DBCapabilityToProto(c.Capability)
	})

	return connect.NewResponse(&eventsv1.GetUserCapabilitiesForOrgResponse{
		Capabilities: protoCaps,
	}), nil
}
