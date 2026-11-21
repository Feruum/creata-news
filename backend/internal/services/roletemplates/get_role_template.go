package roletemplates

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetRoleTemplate(ctx context.Context, req *connect.Request[eventsv1.GetRoleTemplateRequest]) (*connect.Response[eventsv1.GetRoleTemplateResponse], error) {
	slog.Debug("GetRoleTemplate", "id", req.Msg.Id)

	var template models.RoleTemplate
	err := s.db.NewSelect().Model(&template).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var caps []models.RoleTemplateCapability
	err = s.db.NewSelect().Model(&caps).Where("role_template_id = ?", template.ID).Scan(ctx)
	if err != nil {
		caps = []models.RoleTemplateCapability{}
	}

	return connect.NewResponse(&eventsv1.GetRoleTemplateResponse{
		RoleTemplate: dbRoleTemplateToProto(template, caps),
	}), nil
}
