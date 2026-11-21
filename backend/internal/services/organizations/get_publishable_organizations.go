package organizations

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) GetPublishableOrganizations(ctx context.Context, req *connect.Request[eventsv1.GetPublishableOrganizationsRequest]) (*connect.Response[eventsv1.GetPublishableOrganizationsResponse], error) {
	// Get user from authenticated session instead of trusting client-provided userId
	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}

	slog.Debug("GetPublishableOrganizations", "userId", localUserID)

	userIDStr := fmt.Sprintf("%d", localUserID)

	if s.perms == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("permissions client not configured"))
	}

	orgIDStrs, err := s.perms.LookupResources(ctx, userIDStr, "organization", "create_event")
	if err != nil {
		slog.ErrorContext(ctx, "Failed to lookup publishable organizations from SpiceDB", "error", err, "userId", req.Msg.UserId)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(orgIDStrs) == 0 {
		return connect.NewResponse(&eventsv1.GetPublishableOrganizationsResponse{
			Organizations: []*eventsv1.Organization{},
		}), nil
	}

	orgIDs := s.authz.FilterOrganizations(ctx, localUserID, orgIDStrs, "create_event")

	var orgs []models.Organization
	err = s.db.NewSelect().Model(&orgs).Where("id IN (?)", bun.In(orgIDs)).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slugMap := make(map[int32]string)
	if len(orgIDs) > 0 {
		var slugs []models.OrganizationSlug
		if err := s.db.NewSelect().
			Model(&slugs).
			Where("organization_id IN (?)", bun.In(orgIDs)).
			Where("is_current = true").
			Scan(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		slugMap = make(map[int32]string, len(slugs))
		for _, row := range slugs {
			slugMap[row.OrganizationID] = row.Slug
		}
	}

	protoOrgs := lo.Map(orgs, func(o models.Organization, _ int) *eventsv1.Organization {
		protoOrg := modelOrganizationToProto(o)
		if slug, ok := slugMap[o.ID]; ok {
			protoOrg.Slug = slug
		}
		return protoOrg
	})

	return connect.NewResponse(&eventsv1.GetPublishableOrganizationsResponse{
		Organizations: protoOrgs,
	}), nil
}
