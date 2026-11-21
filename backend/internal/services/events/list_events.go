package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func (s *Service) ListEvents(ctx context.Context, req *connect.Request[eventsv1.ListEventsRequest]) (*connect.Response[eventsv1.ListEventsResponse], error) {
	slog.Debug("ListEvents", "page", req.Msg.Page, "limit", req.Msg.Limit)

	page := req.Msg.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	if limit > 200 {
		limit = 200
	}
	now := time.Now()
	requestedStatus := eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED
	if req.Msg.Status != nil {
		requestedStatus = *req.Msg.Status
	}
	isNonPublicStatusRequest := requestedStatus != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED &&
		requestedStatus != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED

	var localUserID int32
	isStaff := false
	if isNonPublicStatusRequest {
		var authErr error
		localUserID, authErr = s.authz.GetLocalUserID(ctx)
		if authErr != nil {
			return nil, authErr
		}
		isStaff = s.authz.CheckPlatformStaff(ctx, localUserID)
		if req.Msg.UserId != nil && *req.Msg.UserId != localUserID && !isStaff {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied for non-public event status listing"))
		}
	}

	query := s.db.NewSelect().Model((*models.Event)(nil))
	if !isNonPublicStatusRequest {
		query = query.Where("publish_on_map = TRUE")
	}

	if isNonPublicStatusRequest && !isStaff {
		query = query.Where("user_id = ?", localUserID)
	} else if req.Msg.UserId != nil {
		query = query.Where("user_id = ?", *req.Msg.UserId)
	}
	if req.Msg.OrganizationId != nil {
		query = filterParticipatingOrganization(query, *req.Msg.OrganizationId)
	}
	if req.Msg.CategoryId != nil {
		query = query.Where("category_id = ?", *req.Msg.CategoryId)
	}
	if requestedStatus != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		query = query.Where("status = ?", protoEventStatusToModel(requestedStatus))
	} else {
		query = query.Where("status = ?", models.EventStatusPublished)
	}

	order := "id"
	if req.Msg.GetUpcomingOnly() {
		query = query.Where("end_time >= ?", now)
		order = "start_time ASC, id ASC"
	}
	var events []models.Event
	err := query.OrderExpr(order).Limit(int(limit)).Offset(int((int64(page)-1)*int64(limit))).Scan(ctx, &events)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	countQuery := s.db.NewSelect().Model((*models.Event)(nil))
	if !isNonPublicStatusRequest {
		countQuery = countQuery.Where("publish_on_map = TRUE")
	}
	if isNonPublicStatusRequest && !isStaff {
		countQuery = countQuery.Where("user_id = ?", localUserID)
	} else if req.Msg.UserId != nil {
		countQuery = countQuery.Where("user_id = ?", *req.Msg.UserId)
	}
	if req.Msg.OrganizationId != nil {
		countQuery = filterParticipatingOrganization(countQuery, *req.Msg.OrganizationId)
	}
	if req.Msg.CategoryId != nil {
		countQuery = countQuery.Where("category_id = ?", *req.Msg.CategoryId)
	}
	if requestedStatus != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		countQuery = countQuery.Where("status = ?", protoEventStatusToModel(requestedStatus))
	} else {
		countQuery = countQuery.Where("status = ?", models.EventStatusPublished)
	}
	if req.Msg.GetUpcomingOnly() {
		countQuery = countQuery.Where("end_time >= ?", now)
	}
	total, err := countQuery.Count(ctx)
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

	// Load current slugs for events and organizations in one query each.
	// These are required for map deep links like /event/<slug> and /club/<slug>.
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

	if err := s.populateEventCounts(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.populateEventParticipants(ctx, protoEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.ListEventsResponse{
		Events: protoEvents,
		Total:  int32(total),
	}), nil
}

func filterParticipatingOrganization(query *bun.SelectQuery, organizationID int32) *bun.SelectQuery {
	return query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where("e.organization_id = ?", organizationID).
			WhereOr("EXISTS (SELECT 1 FROM event_collaborators AS participating_org WHERE participating_org.event_id = e.id AND participating_org.organization_id = ?)", organizationID)
	})
}
