package events

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/slug"
)

func (s *Service) createEvent(ctx context.Context, req *connect.Request[eventsv1.CreateEventRequest]) (*connect.Response[eventsv1.CreateEventResponse], error) {
	slog.Debug("CreateEvent", "title", req.Msg.Title)

	localUserID, err := s.authz.RequireOrgPermission(ctx, req.Msg.OrganizationId, "create_event")
	if err != nil {
		// Platform staff/admin can create/import events for any organization.
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			return nil, err
		}

		localUserID, err = s.authz.RequirePlatformStaff(ctx)
		if err != nil {
			return nil, err
		}
	}

	normalizedLocation := normalizeStructuredLocationRoomID(req.Msg.StructuredLocation)
	if normalizedLocation == nil {
		// Clients may omit location; default to online so create does not panic.
		normalizedLocation = &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE}
	}
	format := deriveFormatFromLocationType(normalizedLocation.Type)

	startTime, endTime, err := parseEventTimeRange(req.Msg.StartTime, req.Msg.EndTime)
	if err != nil {
		return nil, err
	}
	initialStatus, err := initialEventStatus(req.Msg.Status, normalizedLocation)
	if err != nil {
		return nil, err
	}

	baseSlug := slug.Slugify(req.Msg.Title)
	if baseSlug == "" {
		baseSlug = "event"
	}

	eventSlug := ""
	for i := 0; i < 100; i++ {
		candidateSlug := baseSlug
		if i > 0 {
			candidateSlug = slug.WithSuffix(baseSlug, i+1)
		}

		exists, err := s.db.NewSelect().Model((*models.EventSlug)(nil)).Where("slug = ?", candidateSlug).Exists(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check slug: %w", err))
		}
		if !exists {
			eventSlug = candidateSlug
			break
		}
	}
	if eventSlug == "" {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("failed to generate a unique event slug"))
	}

	tx := s.db

	descriptionJSON := json.RawMessage(req.Msg.Description)
	if len(descriptionJSON) == 0 {
		descriptionJSON = json.RawMessage(`""`)
	} else if !json.Valid(descriptionJSON) {
		// Plain text descriptions must be stored as JSON strings for JSONB.
		if encoded, encErr := json.Marshal(req.Msg.Description); encErr == nil {
			descriptionJSON = encoded
		} else {
			descriptionJSON = json.RawMessage(`""`)
		}
	}

	event := &models.Event{
		Title:              req.Msg.Title,
		Description:        descriptionJSON,
		UserID:             localUserID,
		OrganizationID:     req.Msg.OrganizationId,
		StartTime:          startTime,
		EndTime:            endTime,
		Format:             format,
		StructuredLocation: serializeStructuredLocation(normalizedLocation),
		Status:             initialStatus,
	}
	if err := assignPublicationTargetsFromHosts(ctx, s.db, event); err != nil {
		return nil, err
	}
	if req.Msg.Capacity != nil && *req.Msg.Capacity > 0 {
		event.Capacity = req.Msg.Capacity
	}
	applyEventBadgeRequests(event, req.Msg.SocGpaRequested, req.Msg.IrosRequested)
	if req.Msg.Status != nil && *req.Msg.Status == eventsv1.EventStatus_EVENT_STATUS_DRAFT {
		event.Status = models.EventStatusDraft
	}
	if event.Status == models.EventStatusPublished {
		now := time.Now()
		event.PublishedAt = &now
	}
	if req.Msg.ImageUrl != nil {
		event.ImageUrl = req.Msg.ImageUrl
	}
	if req.Msg.CategoryId != nil {
		event.CategoryID = req.Msg.CategoryId
	}

	// Generate check-in secret (32 bytes → hex) for rotating token derivation
	secretBytes := make([]byte, 32)
	if _, randErr := rand.Read(secretBytes); randErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate check-in secret: %w", randErr))
	}
	checkInSecret := hex.EncodeToString(secretBytes)
	event.CheckInSecret = &checkInSecret

	// Generate check-in token (8 bytes → base32 lowercase, truncated to 12 chars)
	tokenBytes := make([]byte, 8)
	if _, randErr := rand.Read(tokenBytes); randErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to generate check-in token: %w", randErr))
	}
	checkInToken := strings.ToLower(base32.StdEncoding.EncodeToString(tokenBytes)[:12])
	event.CheckInToken = &checkInToken

	_, err = tx.NewInsert().Model(event).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	eventSlugModel := &models.EventSlug{
		EventID:   event.ID,
		Slug:      eventSlug,
		IsCurrent: true,
	}
	_, err = tx.NewInsert().Model(eventSlugModel).Returning("*").Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert slug: %w", err))
	}

	if s.perms != nil {
		eventID := fmt.Sprintf("%d", event.ID)
		orgID := fmt.Sprintf("%d", req.Msg.OrganizationId)
		if err := s.perms.SetupEventRelationships(ctx, eventID, orgID, fmt.Sprintf("%d", localUserID)); err != nil {
			slog.WarnContext(ctx, "Failed to setup event relationship in SpiceDB", "error", err, "eventId", eventID)
		}
	}

	if s.publisher != nil {
		orgTitle := ""
		org := &models.Organization{}
		if err := s.db.NewSelect().Model(org).Where("id = ?", event.OrganizationID).Scan(ctx); err == nil {
			orgTitle = org.Title
		}

		var categoryID int32
		categoryName := ""
		if event.CategoryID != nil {
			categoryID = *event.CategoryID
			category := &models.Category{}
			if err := s.db.NewSelect().Model(category).Where("id = ?", *event.CategoryID).Scan(ctx); err == nil {
				categoryName = category.Name
			}
		}

		location, _ := search.ExtractLocationFromStructured(event.StructuredLocation)
		if err := s.publisher.PublishEventCreated(ctx, domainevents.EventCreatedPayload{
			EventID:           event.ID,
			Title:             event.Title,
			OrganizationID:    event.OrganizationID,
			OrganizationTitle: orgTitle,
			CategoryID:        categoryID,
			CategoryName:      categoryName,
			Status:            string(event.Status),
			StartTime:         event.StartTime.Format(time.RFC3339),
			EndTime:           event.EndTime.Format(time.RFC3339),
			Location:          location,
			EventSlug:         eventSlug,
			ActorID:           localUserID,
		}); err != nil {
			s.logPublishFailure(ctx, domainevents.TopicEventCreated, err)
		}
	}

	return connect.NewResponse(&eventsv1.CreateEventResponse{
		Event: dbEventToProto(*event, nil, nil),
	}), nil
}

func ptrString(s string) *string {
	return &s
}

func ptrStringIf(cond bool, s string) *string {
	if cond {
		return &s
	}
	return nil
}

func usernameFromEmail(email string) string {
	if atIdx := strings.Index(email, "@"); atIdx > 0 {
		return email[:atIdx]
	}
	return email
}

func (s *Service) GetEvent(ctx context.Context, req *connect.Request[eventsv1.GetEventRequest]) (*connect.Response[eventsv1.GetEventResponse], error) {
	slog.Debug("GetEvent", "id", req.Msg.Id)

	event := &models.Event{}
	err := s.db.NewSelect().Model(event).Where("id = ?", req.Msg.Id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.requireEventRead(ctx, *event); err != nil {
		return nil, err
	}

	org := &models.Organization{}
	err = s.db.NewSelect().Model(org).Where("id = ?", event.OrganizationID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get organization: %w", err))
	}

	var category *models.Category
	if event.CategoryID != nil {
		cat := &models.Category{}
		catErr := s.db.NewSelect().Model(cat).Where("id = ?", *event.CategoryID).Scan(ctx)
		if catErr == nil {
			category = cat
		}
	}

	response := &eventsv1.GetEventResponse{
		Event: dbEventWithRelationsToProto(*event, *org, category),
	}

	if err := s.populateEventCounts(ctx, []*eventsv1.Event{response.Event}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.populateEventParticipants(ctx, []*eventsv1.Event{response.Event}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	localUserID, authErr := s.authz.GetLocalUserID(ctx)
	if authErr == nil {
		response.CanEdit = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", event.ID), "edit")
		response.CanReviewBadges = s.authz.CheckPlatformAdmin(ctx, localUserID)
		response.CanDelete = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", event.ID), "delete")
		response.CanManageRegistrations = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", event.ID), "manage_registrations")
	}

	return connect.NewResponse(response), nil
}

func (s *Service) GetEventBySlug(ctx context.Context, req *connect.Request[eventsv1.GetEventBySlugRequest]) (*connect.Response[eventsv1.GetEventBySlugResponse], error) {
	slog.Debug("GetEventBySlug", "slug", req.Msg.Slug)

	var row eventSlugRow
	err := s.db.NewSelect().
		TableExpr("events AS e").
		ColumnExpr("e.*").
		ColumnExpr("es.slug").
		ColumnExpr("es.is_current").
		ColumnExpr("(SELECT slug FROM event_slugs WHERE event_id = e.id AND is_current = true) AS canonical_slug").
		Join("INNER JOIN event_slugs es ON es.event_id = e.id").
		Where("es.slug = ?", req.Msg.Slug).
		Scan(ctx, &row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.requireEventRead(ctx, row.Event); err != nil {
		return nil, err
	}

	org := &models.Organization{}
	err = s.db.NewSelect().Model(org).Where("id = ?", row.OrganizationID).Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get organization: %w", err))
	}

	var category *models.Category
	if row.CategoryID != nil {
		cat := &models.Category{}
		catErr := s.db.NewSelect().Model(cat).Where("id = ?", *row.CategoryID).Scan(ctx)
		if catErr == nil {
			category = cat
		}
	}

	response := &eventsv1.GetEventBySlugResponse{
		Event: dbEventWithRelationsToProto(row.Event, *org, category),
	}

	if !row.IsCurrent {
		response.CanonicalSlug = &row.CanonicalSlug
	}

	if err := s.populateEventCounts(ctx, []*eventsv1.Event{response.Event}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.populateEventParticipants(ctx, []*eventsv1.Event{response.Event}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	localUserID, authErr := s.authz.GetLocalUserID(ctx)
	if authErr == nil {
		response.CanEdit = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", row.ID), "edit")
		response.CanReviewBadges = s.authz.CheckPlatformAdmin(ctx, localUserID)
		response.CanDelete = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", row.ID), "delete")
		response.CanManageRegistrations = s.authz.CheckResourcePermission(ctx, localUserID, "event", fmt.Sprintf("%d", row.ID), "manage_registrations")
	}

	return connect.NewResponse(response), nil
}

func (s *Service) deleteEvent(ctx context.Context, req *connect.Request[eventsv1.DeleteEventRequest]) (*connect.Response[eventsv1.DeleteEventResponse], error) {
	slog.Debug("DeleteEvent", "id", req.Msg.Id)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.Id), "delete")
	if err != nil {
		return nil, err
	}

	event := &models.Event{}
	if err := s.db.NewSelect().Model(event).Where("id = ?", req.Msg.Id).For("UPDATE").Scan(ctx); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
	}
	if err := s.releaseDeletedEventBookings(ctx, event.ID); err != nil {
		return nil, err
	}

	_, err = s.db.NewDelete().Model((*models.Event)(nil)).Where("id = ?", req.Msg.Id).Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.publisher != nil {
		if err := s.publisher.PublishEventDeleted(ctx, domainevents.EventDeletedPayload{
			EventID:        event.ID,
			Title:          event.Title,
			OrganizationID: event.OrganizationID,
			ActorID:        localUserID,
		}); err != nil {
			s.logPublishFailure(ctx, domainevents.TopicEventDeleted, err)
		}
	}

	return connect.NewResponse(&eventsv1.DeleteEventResponse{
		Success: true,
	}), nil
}

// Direct upload credentials bypass byte validation and are no longer issued.
func (s *Service) GetEventImageUploadUrl(ctx context.Context, req *connect.Request[eventsv1.GetEventImageUploadUrlRequest]) (*connect.Response[eventsv1.GetEventImageUploadUrlResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("use UploadEventImage to upload a validated image"))
}
