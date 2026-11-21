package events

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) ListEventsForAdmin(ctx context.Context, req *connect.Request[eventsv1.ListEventsForAdminRequest]) (*connect.Response[eventsv1.ListEventsForAdminResponse], error) {
	slog.Debug("ListEventsForAdmin", "page", req.Msg.Page, "limit", req.Msg.Limit)

	localUserID, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return nil, err
	}
	isStaff := s.authz.CheckPlatformStaff(ctx, localUserID)
	var editableOrgIDs []int32
	if !isStaff {
		editableOrgIDs, err = s.editableOrganizationIDs(ctx, localUserID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	if limit > 1000 {
		limit = 1000
	}

	query := s.db.NewSelect().Model((*models.Event)(nil)).Column(
		"id", "title", "description", "image_url", "user_id", "organization_id", "start_time", "end_time",
		"format", "created_at", "updated_at", "category_id", "capacity", "structured_location", "status",
		"check_in_grace_minutes", "check_in_secret", "check_in_token", "published_at", "completed_at", "cancelled_at",
		"cancelled_by", "cancel_reason", "archived_at", "publish_on_map", "publish_on_sunday_open",
		"soc_gpa_badge_status", "soc_gpa_badge_reviewed_by", "soc_gpa_badge_reviewed_at",
		"iros_badge_status", "iros_badge_reviewed_by", "iros_badge_reviewed_at",
	)
	if !isStaff {
		query = restrictEditableEvents(query, localUserID, editableOrgIDs)
	}
	if req.Msg.UserId != nil {
		query = query.Where("user_id = ?", *req.Msg.UserId)
	}
	if req.Msg.OrganizationId != nil {
		query = query.Where("organization_id = ?", *req.Msg.OrganizationId)
	}
	if req.Msg.Status != nil && *req.Msg.Status != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		query = query.Where("status = ?", protoEventStatusToModel(*req.Msg.Status))
	}

	var events []models.Event
	err = query.OrderExpr("id").Limit(int(limit)).Offset(int((int64(page)-1)*int64(limit))).Scan(ctx, &events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	eventIDs := lo.Map(events, func(e models.Event, _ int) int32 {
		return e.ID
	})
	orgIDs := lo.Map(events, func(e models.Event, _ int) int32 {
		return e.OrganizationID
	})
	catIDs := lo.FilterMap(events, func(e models.Event, _ int) (int32, bool) {
		if e.CategoryID == nil {
			return 0, false
		}
		return *e.CategoryID, true
	})

	eventSlugMap := make(map[int32]string)
	if len(eventIDs) > 0 {
		var slugs []models.EventSlug
		if err := s.db.NewSelect().
			Model(&slugs).
			Where("event_id IN (?)", bun.In(eventIDs)).
			Where("is_current = true").
			Scan(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		eventSlugMap = make(map[int32]string, len(slugs))
		for _, row := range slugs {
			eventSlugMap[row.EventID] = row.Slug
		}
	}

	orgSlugMap := make(map[int32]string)
	if len(orgIDs) > 0 {
		var slugs []models.OrganizationSlug
		if err := s.db.NewSelect().
			Model(&slugs).
			Where("organization_id IN (?)", bun.In(orgIDs)).
			Where("is_current = true").
			Scan(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		orgSlugMap = make(map[int32]string, len(slugs))
		for _, row := range slugs {
			orgSlugMap[row.OrganizationID] = row.Slug
		}
	}

	var orgs []models.Organization
	if len(orgIDs) > 0 {
		_ = s.db.NewSelect().Model(&orgs).Where("id IN (?)", bun.In(orgIDs)).Scan(ctx)
	}
	orgMap := make(map[int32]models.Organization, len(orgs))
	for _, o := range orgs {
		orgMap[o.ID] = o
	}

	var cats []models.Category
	if len(catIDs) > 0 {
		_ = s.db.NewSelect().Model(&cats).Where("id IN (?)", bun.In(catIDs)).Scan(ctx)
	}
	catMap := make(map[int32]models.Category, len(cats))
	for _, c := range cats {
		catMap[c.ID] = c
	}

	protoEvents := make([]*eventsv1.Event, len(events))
	for i, e := range events {
		org := orgMap[e.OrganizationID]
		var category *models.Category
		if e.CategoryID != nil {
			if cat, ok := catMap[*e.CategoryID]; ok {
				category = &cat
			}
		}
		protoEvent := dbEventWithRelationsToProto(e, org, category)
		if slug, ok := eventSlugMap[e.ID]; ok {
			protoEvent.Slug = slug
		}
		if protoEvent.Organization != nil {
			if slug, ok := orgSlugMap[e.OrganizationID]; ok {
				protoEvent.Organization.Slug = slug
			}
		}
		protoEvents[i] = protoEvent
	}

	countQuery := s.db.NewSelect().Model((*models.Event)(nil))
	if !isStaff {
		countQuery = restrictEditableEvents(countQuery, localUserID, editableOrgIDs)
	}
	if req.Msg.UserId != nil {
		countQuery = countQuery.Where("user_id = ?", *req.Msg.UserId)
	}
	if req.Msg.OrganizationId != nil {
		countQuery = countQuery.Where("organization_id = ?", *req.Msg.OrganizationId)
	}
	if req.Msg.Status != nil && *req.Msg.Status != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		countQuery = countQuery.Where("status = ?", protoEventStatusToModel(*req.Msg.Status))
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.populateEventCounts(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.populateEventParticipants(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.ListEventsForAdminResponse{
		Events: protoEvents,
		Total:  int32(total),
	}), nil
}
