package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
	"github.com/studyverse/ems-backend/internal/search"
)

func (s *Service) publishEvent(ctx context.Context, req *connect.Request[eventsv1.PublishEventRequest]) (*connect.Response[eventsv1.PublishEventResponse], error) {
	slog.Debug("PublishEvent", "eventId", req.Msg.EventId)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	event := &models.Event{ID: req.Msg.EventId}
	err = s.db.NewSelect().Model(event).WherePK().For("UPDATE").Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch event: %w", err))
	}

	if event.Status != models.EventStatusDraft {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only draft events can be published, current status: %s", event.Status))
	}

	if err := assignPublicationTargetsFromHosts(ctx, s.db, event); err != nil {
		return nil, err
	}
	if err := s.ensureRoomBookingReadyForPublish(ctx, event); err != nil {
		return nil, err
	}

	now := time.Now()
	event.Status = models.EventStatusPublished
	event.PublishedAt = &now

	_, err = s.db.NewUpdate().Model(event).
		Column("status", "published_at", "publish_on_map", "publish_on_sunday_open").
		WherePK().
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to publish event: %w", err))
	}

	org := &models.Organization{}
	_ = s.db.NewSelect().Model(org).Where("id = ?", event.OrganizationID).Scan(ctx)

	// Fetch current event slug for notification URL
	eventSlug := &models.EventSlug{}
	_ = s.db.NewSelect().Model(eventSlug).Where("event_id = ? AND is_current = ?", event.ID, true).Scan(ctx)
	location, _ := search.ExtractLocationFromStructured(event.StructuredLocation)
	if s.publisher != nil {
		if err := s.publisher.PublishEventPublished(ctx, domainevents.EventPublishedPayload{
			EventID:          event.ID,
			Title:            event.Title,
			OrganizationID:   event.OrganizationID,
			OrganizationName: org.Title,
			StartTime:        event.StartTime.Format(time.RFC3339),
			Location:         location,
			EventSlug:        eventSlug.Slug,
			MapAppURL:        s.mapAppURL,
			ActorID:          localUserID,
		}); err != nil {
			s.logPublishFailure(ctx, domainevents.TopicEventPublished, err)
		}
	}

	var category *models.Category
	if event.CategoryID != nil {
		cat := &models.Category{}
		if catErr := s.db.NewSelect().Model(cat).Where("id = ?", *event.CategoryID).Scan(ctx); catErr == nil {
			category = cat
		}
	}

	return connect.NewResponse(&eventsv1.PublishEventResponse{
		Event: dbEventWithRelationsToProto(*event, *org, category),
	}), nil
}

type publishReadinessResult struct {
	CanPublishNow        bool
	RequiresRoomApproval bool
	State                eventsv1.EventPublishReadinessState
	BookingRequestID     *int32
	RejectionReason      *string
	Message              string
}

func (s *Service) getPublishReadinessForEvent(ctx context.Context, event *models.Event) (publishReadinessResult, error) {
	notDraftResult := publishReadinessResult{
		CanPublishNow:        false,
		RequiresRoomApproval: false,
		State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_NOT_DRAFT,
		Message:              fmt.Sprintf("only draft events can be published, current status: %s", event.Status),
	}

	if event.Status != models.EventStatusDraft {
		return notDraftResult, nil
	}

	loc := parseStructuredLocation(event.StructuredLocation)
	if loc != nil && (loc.Type == eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM || loc.Type == eventsv1.LocationType_LOCATION_TYPE_HYBRID) && (loc.RoomId == nil || roomid.Normalize(*loc.RoomId) == "") {
		return publishReadinessResult{RequiresRoomApproval: true, State: eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_MISSING, Message: "campus room events require a valid room ID and approved booking"}, nil
	}
	if loc == nil || loc.RoomId == nil || strings.TrimSpace(*loc.RoomId) == "" {
		return publishReadinessResult{
			CanPublishNow:        true,
			RequiresRoomApproval: false,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_NO_ROOM_APPROVAL_REQUIRED,
			Message:              "no campus room approval required",
		}, nil
	}

	if loc.Type != eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM && loc.Type != eventsv1.LocationType_LOCATION_TYPE_HYBRID {
		return publishReadinessResult{
			CanPublishNow:        true,
			RequiresRoomApproval: false,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_NO_ROOM_APPROVAL_REQUIRED,
			Message:              "no campus room approval required",
		}, nil
	}

	normalizedRoomID := roomid.Normalize(*loc.RoomId)
	if normalizedRoomID == "" {
		return publishReadinessResult{
			CanPublishNow:        true,
			RequiresRoomApproval: false,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_NO_ROOM_APPROVAL_REQUIRED,
			Message:              "no campus room approval required",
		}, nil
	}

	var requests []models.BookingRequest
	err := s.db.NewSelect().
		Model(&requests).
		Where("event_id = ?", event.ID).
		Where("start_time <= ? AND end_time >= ?", event.StartTime, event.EndTime).
		OrderExpr("created_at DESC, id DESC").
		For("SHARE").
		Scan(ctx)
	if err != nil {
		return publishReadinessResult{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check booking request status: %w", err))
	}
	latest := selectCoveringBooking(*event, requests)
	if latest == nil {
		return publishReadinessResult{
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_MISSING,
			Message:              "cannot publish: no approved booking covers this room and event time",
		}, nil
	}

	switch latest.Status {
	case models.BookingRequestStatusApproved:
		if latest.StartTime.After(event.StartTime) || latest.EndTime.Before(event.EndTime) {
			return publishReadinessResult{RequiresRoomApproval: true, State: eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_MISSING, Message: "approved booking does not cover this event's time range"}, nil
		}
		return publishReadinessResult{
			CanPublishNow:        true,
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_READY,
			BookingRequestID:     &latest.ID,
			Message:              "room booking approved",
		}, nil
	case models.BookingRequestStatusPending:
		return publishReadinessResult{
			CanPublishNow:        false,
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_WAITING_ROOM_APPROVAL,
			BookingRequestID:     &latest.ID,
			Message:              "cannot publish: room booking request is still pending approval",
		}, nil
	case models.BookingRequestStatusRejected:
		return publishReadinessResult{
			CanPublishNow:        false,
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_REJECTED,
			BookingRequestID:     &latest.ID,
			RejectionReason:      latest.RejectionReason,
			Message:              "cannot publish: room booking request was rejected",
		}, nil
	case models.BookingRequestStatusCancelled:
		return publishReadinessResult{
			CanPublishNow:        false,
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_CANCELLED,
			BookingRequestID:     &latest.ID,
			Message:              "cannot publish: room booking request was cancelled",
		}, nil
	default:
		return publishReadinessResult{
			CanPublishNow:        false,
			RequiresRoomApproval: true,
			State:                eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_WAITING_ROOM_APPROVAL,
			BookingRequestID:     &latest.ID,
			Message:              "cannot publish: room booking request is not approved",
		}, nil
	}
}

func (s *Service) ensureRoomBookingReadyForPublish(ctx context.Context, event *models.Event) error {
	readiness, err := s.getPublishReadinessForEvent(ctx, event)
	if err != nil {
		return err
	}
	if readiness.CanPublishNow {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, errors.New(readiness.Message))
}
