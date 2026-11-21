package events

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func validateEventOwnershipUpdate(event models.Event, req *eventsv1.UpdateEventRequest) error {
	if (req.UserId != nil && *req.UserId != event.UserID) || (req.OrganizationId != nil && *req.OrganizationId != event.OrganizationID) {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("event ownership transfers are not supported"))
	}
	return nil
}
func (s *Service) requireEventRead(ctx context.Context, event models.Event) error {
	if event.Status == models.EventStatusPublished && event.PublishOnMap {
		return nil
	}
	id, err := s.authz.GetLocalUserID(ctx)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
	}
	if !s.authz.CheckResourcePermission(ctx, id, "event", fmt.Sprint(event.ID), "edit") {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("event not found"))
	}
	return nil
}
