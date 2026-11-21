package events

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
	"google.golang.org/protobuf/proto"
)

// Compare the effective scheduling values, not request presence or display labels.
func bookingDetailsChanged(original, candidate models.Event) bool {
	if !original.StartTime.Equal(candidate.StartTime) || !original.EndTime.Equal(candidate.EndTime) {
		return true
	}
	previous, next := parseStructuredLocation(original.StructuredLocation), parseStructuredLocation(candidate.StructuredLocation)
	if previous.GetType() != next.GetType() {
		return true
	}
	if next.GetType() != eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM && next.GetType() != eventsv1.LocationType_LOCATION_TYPE_HYBRID {
		return false
	}
	return roomid.Normalize(previous.GetRoomId()) != roomid.Normalize(next.GetRoomId())
}

func proposedPublicationEvent(original models.Event, request *eventsv1.GetEventPublishReadinessRequest) (models.Event, error) {
	if request.ProposedStartTime == nil && request.ProposedEndTime == nil && request.ProposedLocation == nil {
		return original, nil
	}
	if original.Status != models.EventStatusDraft && original.Status != models.EventStatusPublished {
		return original, connect.NewError(connect.CodeFailedPrecondition, errors.New("only draft or published events can have scheduling proposals"))
	}
	candidate := original
	var err error
	if request.ProposedStartTime != nil {
		candidate.StartTime, err = time.Parse(time.RFC3339, *request.ProposedStartTime)
		if err != nil {
			return original, connect.NewError(connect.CodeInvalidArgument, errors.New("proposed_start_time must be RFC3339"))
		}
	}
	if request.ProposedEndTime != nil {
		candidate.EndTime, err = time.Parse(time.RFC3339, *request.ProposedEndTime)
		if err != nil {
			return original, connect.NewError(connect.CodeInvalidArgument, errors.New("proposed_end_time must be RFC3339"))
		}
	}
	if !candidate.EndTime.After(candidate.StartTime) {
		return original, connect.NewError(connect.CodeInvalidArgument, errors.New("end_time must be after start_time"))
	}
	if request.ProposedLocation != nil {
		location := proto.Clone(request.ProposedLocation).(*eventsv1.EventLocation)
		candidate.StructuredLocation = serializeStructuredLocation(normalizeStructuredLocationRoomID(location))
	}
	// Published events are checked as candidates; the saved status remains unchanged.
	candidate.Status = models.EventStatusDraft
	return candidate, nil
}

// Requests arrive newest first. A new proposal must not hide a still-valid approval.
func selectCoveringBooking(event models.Event, requests []models.BookingRequest) *models.BookingRequest {
	roomID := roomid.Normalize(parseStructuredLocation(event.StructuredLocation).GetRoomId())
	var latest *models.BookingRequest
	for i := range requests {
		request := &requests[i]
		if roomid.Normalize(request.RoomID) != roomID || request.StartTime.After(event.StartTime) || request.EndTime.Before(event.EndTime) {
			continue
		}
		if request.Status == models.BookingRequestStatusApproved {
			return request
		}
		if latest == nil {
			latest = request
		}
	}
	return latest
}
