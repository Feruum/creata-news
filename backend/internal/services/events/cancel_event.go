package events

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/search"
)

func (s *Service) CancelEvent(ctx context.Context, req *connect.Request[eventsv1.CancelEventRequest]) (*connect.Response[eventsv1.CancelEventResponse], error) {
	slog.Debug("CancelEvent", "eventId", req.Msg.EventId)

	localUserID, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer func() { _ = tx.Rollback() }()
	publisher := s.publisher.InTransaction(tx)

	event := &models.Event{ID: req.Msg.EventId}
	err = tx.NewSelect().Model(event).WherePK().For("UPDATE").Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch event: %w", err))
	}

	if event.Status != models.EventStatusDraft && event.Status != models.EventStatusPublished {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only draft or published events can be cancelled, current status: %s", event.Status))
	}

	now := time.Now()
	event.Status = models.EventStatusCancelled
	event.CancelledAt = &now
	event.CancelledBy = &localUserID
	if req.Msg.Reason != nil {
		event.CancelReason = req.Msg.Reason
	}

	_, err = tx.NewUpdate().Model(event).
		Column("status", "cancelled_at", "cancelled_by", "cancel_reason").
		WherePK().
		Exec(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel event: %w", err))
	}

	org := &models.Organization{}
	_ = tx.NewSelect().Model(org).Where("id = ?", event.OrganizationID).Scan(ctx)

	var category *models.Category
	var categoryID int32
	categoryName := ""
	if event.CategoryID != nil {
		categoryID = *event.CategoryID
		cat := &models.Category{}
		if catErr := tx.NewSelect().Model(cat).Where("id = ?", *event.CategoryID).Scan(ctx); catErr == nil {
			category = cat
			categoryName = cat.Name
		}
	}

	eventSlug := ""
	eventSlugModel := &models.EventSlug{}
	if err := tx.NewSelect().Model(eventSlugModel).Where("event_id = ? AND is_current = ?", event.ID, true).Scan(ctx); err == nil {
		eventSlug = eventSlugModel.Slug
	}
	location, _ := search.ExtractLocationFromStructured(event.StructuredLocation)

	if publisher != nil {
		if err := publisher.PublishEventUpdated(ctx, domainevents.EventUpdatedPayload{
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
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.CancelEventResponse{
		Event: dbEventWithRelationsToProto(*event, *org, category),
	}), nil
}
