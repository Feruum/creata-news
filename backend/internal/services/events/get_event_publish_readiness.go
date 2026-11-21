package events

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetEventPublishReadiness(ctx context.Context, req *connect.Request[eventsv1.GetEventPublishReadinessRequest]) (*connect.Response[eventsv1.GetEventPublishReadinessResponse], error) {
	slog.Debug("GetEventPublishReadiness", "eventId", req.Msg.EventId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "edit")
	if err != nil {
		return nil, err
	}

	event := &models.Event{ID: req.Msg.EventId}
	err = s.db.NewSelect().Model(event).WherePK().Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch event: %w", err))
	}

	candidate, err := proposedPublicationEvent(*event, req.Msg)
	if err != nil {
		return nil, err
	}
	readiness, err := s.getPublishReadinessForEvent(ctx, &candidate)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&eventsv1.GetEventPublishReadinessResponse{
		CanPublishNow:        readiness.CanPublishNow,
		RequiresRoomApproval: readiness.RequiresRoomApproval,
		State:                readiness.State,
		BookingRequestId:     readiness.BookingRequestID,
		RejectionReason:      readiness.RejectionReason,
		Message:              readiness.Message,
	}), nil
}
