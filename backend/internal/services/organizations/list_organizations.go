package organizations

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) ListOrganizations(ctx context.Context, req *connect.Request[eventsv1.ListOrganizationsRequest]) (*connect.Response[eventsv1.ListOrganizationsResponse], error) {
	slog.Debug("ListOrganizations", "page", req.Msg.Page, "limit", req.Msg.Limit)

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	var orgs []models.Organization
	err := s.db.NewSelect().Model(&orgs).Limit(int(limit)).Offset(int((page - 1) * limit)).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total, err := s.db.NewSelect().Model((*models.Organization)(nil)).Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	orgIDs := lo.Map(orgs, func(o models.Organization, _ int) int32 {
		return o.ID
	})

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

	return connect.NewResponse(&eventsv1.ListOrganizationsResponse{
		Organizations: protoOrgs,
		Total:         int32(total),
	}), nil
}
