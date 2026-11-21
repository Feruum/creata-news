package organizations

import (
	"context"
	"database/sql"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetOrganization(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationRequest]) (*connect.Response[eventsv1.GetOrganizationResponse], error) {
	slog.Debug("GetOrganization", "id", req.Msg.Id)

	var org models.Organization
	err := s.db.NewSelect().Model(&org).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &eventsv1.GetOrganizationResponse{
		Organization: modelOrganizationToProto(org),
	}

	if response.Organization != nil {
		var orgSlug models.OrganizationSlug
		slugErr := s.db.NewSelect().
			Model(&orgSlug).
			Column("slug").
			Where("organization_id = ?", org.ID).
			Where("is_current = true").
			Limit(1).
			Scan(ctx)
		if slugErr == nil {
			response.Organization.Slug = orgSlug.Slug
		}
	}

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err == nil {
		response.CanEdit = s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.Id, "manage_settings")
		response.CanManageMembers = s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.Id, "manage_members")
		response.CanCreateEvent = s.authz.CheckOrgPermission(ctx, localUserID, req.Msg.Id, "create_event")
	}

	return connect.NewResponse(response), nil
}
