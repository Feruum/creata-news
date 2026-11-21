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

func (s *Service) DeleteRoleTemplate(ctx context.Context, req *connect.Request[eventsv1.DeleteRoleTemplateRequest]) (*connect.Response[eventsv1.DeleteRoleTemplateResponse], error) {
	slog.Debug("DeleteRoleTemplate", "id", req.Msg.Id)

	var existing models.RoleTemplate
	err := s.db.NewSelect().Model(&existing).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if existing.OrganizationID != nil {
		if _, permErr := s.authz.RequireOrgPermission(ctx, *existing.OrganizationID, "manage_settings"); permErr != nil {
			return nil, permErr
		}
	} else {
		if _, permErr := s.authz.RequirePlatformStaff(ctx); permErr != nil {
			return nil, permErr
		}
	}

	_, err = s.db.NewDelete().Model((*models.RoleTemplate)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.DeleteRoleTemplateResponse{
		Success: true,
	}), nil
}
