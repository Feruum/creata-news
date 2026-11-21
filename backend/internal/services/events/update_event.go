package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/slug"
)

func (s *Service) updateEvent(ctx context.Context, req *connect.Request[eventsv1.UpdateEventRequest]) (*connect.Response[eventsv1.UpdateEventResponse], error) {
	slog.Debug("UpdateEvent", "id", req.Msg.Id)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.Id), "edit")
	if err != nil {
		return nil, err
	}

	tx := s.db

	event := &models.Event{ID: req.Msg.Id}
	err = tx.NewSelect().Model(event).WherePK().For("UPDATE").Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
	}
	if err := validateEventOwnershipUpdate(*event, req.Msg); err != nil {
		return nil, err
	}
	original := *event
	applyEventBadgeRequests(event, req.Msg.SocGpaRequested, req.Msg.IrosRequested)
	titleChanged := req.Msg.Title != nil && *req.Msg.Title != event.Title
	if req.Msg.Title != nil {
		event.Title = *req.Msg.Title
	}
	if req.Msg.Description != nil {
		event.Description = json.RawMessage(*req.Msg.Description)
		if !json.Valid(event.Description) {
			encoded, err := json.Marshal(*req.Msg.Description)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid description: %w", err))
			}
			event.Description = encoded
		}
	}
	if req.Msg.ImageUrl != nil {
		event.ImageUrl = req.Msg.ImageUrl
	}
	if req.Msg.UserId != nil {
		event.UserID = *req.Msg.UserId
	}
	if req.Msg.OrganizationId != nil {
		event.OrganizationID = *req.Msg.OrganizationId
	}
	if req.Msg.StartTime != nil {
		t, parseErr := time.Parse(time.RFC3339, *req.Msg.StartTime)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid start_time: %w", parseErr))
		}
		event.StartTime = t
	}
	if req.Msg.EndTime != nil {
		t, parseErr := time.Parse(time.RFC3339, *req.Msg.EndTime)
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid end_time: %w", parseErr))
		}
		event.EndTime = t
	}
	if req.Msg.CategoryId != nil {
		event.CategoryID = req.Msg.CategoryId
	}
	if req.Msg.Capacity != nil {
		if *req.Msg.Capacity == 0 {
			event.Capacity = nil
		} else {
			event.Capacity = req.Msg.Capacity
		}
	}
	if req.Msg.StructuredLocation != nil {
		normalizedLocation := normalizeStructuredLocationRoomID(req.Msg.StructuredLocation)
		event.StructuredLocation = serializeStructuredLocation(normalizedLocation)
		event.Format = deriveFormatFromLocationType(normalizedLocation.Type)
	}

	if !event.EndTime.After(event.StartTime) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("end_time must be after start_time"))
	}
	if event.Capacity != nil && *event.Capacity < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("capacity must be nonnegative"))
	}
	if event.Capacity != nil {
		var count int
		if err := tx.NewRaw("SELECT (SELECT count(*) FROM event_registrations WHERE event_id=? AND status='registered') + (SELECT count(*) FROM external_registrations WHERE event_id=? AND status='registered')", event.ID, event.ID).Scan(ctx, &count); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if count > int(*event.Capacity) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("capacity cannot be below existing registrations"))
		}
	}
	if event.Status == models.EventStatusPublished && bookingDetailsChanged(original, *event) {
		readinessEvent := *event
		readinessEvent.Status = models.EventStatusDraft
		if err := s.ensureRoomBookingReadyForPublish(ctx, &readinessEvent); err != nil {
			return nil, err
		}
		if err := s.releaseReplacedPrimaryBooking(ctx, original, *event); err != nil {
			return nil, err
		}
	}
	_, err = tx.NewUpdate().Model(event).WherePK().Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if titleChanged {
		baseSlug := slug.Slugify(*req.Msg.Title)
		eventSlug := baseSlug
		for i := 1; i <= 100; i++ {
			exists, checkErr := s.db.NewSelect().Model((*models.EventSlug)(nil)).Where("slug = ?", eventSlug).Exists(ctx)
			if checkErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check slug: %w", checkErr))
			}
			if !exists {
				break
			}
			eventSlug = slug.WithSuffix(baseSlug, i+1)
		}
		_, err = tx.NewUpdate().Model((*models.EventSlug)(nil)).Where("event_id = ?", event.ID).Set("is_current = ?", false).Exec(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to mark old slugs: %w", err))
		}
		newSlug := &models.EventSlug{
			EventID:   event.ID,
			Slug:      eventSlug,
			IsCurrent: true,
		}
		_, err = tx.NewInsert().Model(newSlug).Exec(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert new slug: %w", err))
		}
	}

	return connect.NewResponse(&eventsv1.UpdateEventResponse{
		Event: s.updatedEventProto(ctx, event, localUserID),
	}), nil
}

func (s *Service) updatedEventProto(ctx context.Context, event *models.Event, localUserID int32) *eventsv1.Event {
	org := &models.Organization{}
	_ = s.db.NewSelect().Model(org).Where("id = ?", event.OrganizationID).Scan(ctx)

	var category *models.Category
	var categoryID int32
	categoryName := ""
	if event.CategoryID != nil {
		categoryID = *event.CategoryID
		cat := &models.Category{}
		catErr := s.db.NewSelect().Model(cat).Where("id = ?", *event.CategoryID).Scan(ctx)
		if catErr == nil {
			category = cat
			categoryName = cat.Name
		}
	}

	location, _ := search.ExtractLocationFromStructured(event.StructuredLocation)
	eventSlug := ""
	eventSlugModel := &models.EventSlug{}
	if err := s.db.NewSelect().Model(eventSlugModel).Where("event_id = ? AND is_current = ?", event.ID, true).Scan(ctx); err == nil {
		eventSlug = eventSlugModel.Slug
	}

	if s.publisher != nil {
		if err := s.publisher.PublishEventUpdated(ctx, domainevents.EventUpdatedPayload{
			EventID:           event.ID,
			Title:             event.Title,
			OrganizationID:    event.OrganizationID,
			OrganizationTitle: org.Title,
			CategoryID:        categoryID,
			CategoryName:      categoryName,
			Status:            string(event.Status),
			StartTime:         event.StartTime.Format(time.RFC3339),
			EndTime:           event.EndTime.Format(time.RFC3339),
			Location:          location,
			EventSlug:         eventSlug,
			ActorID:           localUserID,
		}); err != nil {
			s.logPublishFailure(ctx, domainevents.TopicEventUpdated, err)
		}
	}

	return dbEventWithRelationsToProto(*event, *org, category)
}
