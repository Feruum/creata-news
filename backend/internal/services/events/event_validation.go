package events

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func parseEventTimeRange(start, end string) (time.Time, time.Time, error) {
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, errors.New("start_time must be RFC3339"))
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, errors.New("end_time must be RFC3339"))
	}
	if !endTime.After(startTime) {
		return time.Time{}, time.Time{}, connect.NewError(connect.CodeInvalidArgument, errors.New("end_time must be after start_time"))
	}
	return startTime, endTime, nil
}

func initialEventStatus(requested *eventsv1.EventStatus, location *eventsv1.EventLocation) (models.EventStatus, error) {
	if requested != nil && *requested == eventsv1.EventStatus_EVENT_STATUS_DRAFT {
		return models.EventStatusDraft, nil
	}
	if requested != nil && *requested != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED && *requested != eventsv1.EventStatus_EVENT_STATUS_UNSPECIFIED {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("new events must be draft or published"))
	}
	if location != nil && (location.Type == eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM || location.Type == eventsv1.LocationType_LOCATION_TYPE_HYBRID) {
		return "", connect.NewError(connect.CodeFailedPrecondition, errors.New("create campus room events as drafts, then publish after booking approval"))
	}
	return models.EventStatusPublished, nil
}
