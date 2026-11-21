package organizations

import (
	"context"
	"database/sql"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetOrganizationBySlug(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationBySlugRequest]) (*connect.Response[eventsv1.GetOrganizationBySlugResponse], error) {
	slog.Debug("GetOrganizationBySlug", "slug", req.Msg.Slug)

	var result organizationWithSlug
	err := s.db.NewSelect().
		TableExpr("organizations AS o").
		ColumnExpr("o.*").
		ColumnExpr("os.is_current").
		ColumnExpr("(SELECT os2.slug FROM organization_slugs os2 WHERE os2.organization_id = o.id AND os2.is_current = true LIMIT 1) AS canonical_slug").
		Join("JOIN organization_slugs AS os ON os.organization_id = o.id").
		Where("lower(os.slug) = lower(?)", req.Msg.Slug).
		OrderExpr("os.is_current DESC").
		OrderExpr("os.created_at DESC").
		Limit(1).
		Scan(ctx, &result)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	org := organizationRowToOrganization(result)

	protoOrg := modelOrganizationToProto(org)
	// Always return the current/canonical slug in the organization object for stable deep links.
	if result.CanonicalSlug != "" {
		protoOrg.Slug = result.CanonicalSlug
	} else {
		protoOrg.Slug = req.Msg.Slug
	}

	response := &eventsv1.GetOrganizationBySlugResponse{
		Organization: protoOrg,
	}

	if !result.IsCurrent {
		response.CanonicalSlug = &result.CanonicalSlug
	}

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err == nil {
		response.CanEdit = s.authz.CheckOrgPermission(ctx, localUserID, org.ID, "manage_settings")
		response.CanManageMembers = s.authz.CheckOrgPermission(ctx, localUserID, org.ID, "manage_members")
		response.CanCreateEvent = s.authz.CheckOrgPermission(ctx, localUserID, org.ID, "create_event")
	}

	return connect.NewResponse(response), nil
}
