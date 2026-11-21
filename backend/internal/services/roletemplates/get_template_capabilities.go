package roletemplates

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/samber/lo"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetTemplateCapabilities(ctx context.Context, req *connect.Request[eventsv1.GetTemplateCapabilitiesRequest]) (*connect.Response[eventsv1.GetTemplateCapabilitiesResponse], error) {
	slog.Debug("GetTemplateCapabilities", "templateId", req.Msg.RoleTemplateId)

	var caps []models.RoleTemplateCapability
	err := s.db.NewSelect().Model(&caps).Where("role_template_id = ?", req.Msg.RoleTemplateId).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoCaps := lo.Map(caps, func(c models.RoleTemplateCapability, _ int) eventsv1.Capability {
		return dbCapabilityToProto(c)
	})

	return connect.NewResponse(&eventsv1.GetTemplateCapabilitiesResponse{
		Capabilities: protoCaps,
	}), nil
}
