package events

import (
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/roomid"
)

func normalizeStructuredLocationRoomID(loc *eventsv1.EventLocation) *eventsv1.EventLocation {
	if loc == nil || loc.RoomId == nil {
		return loc
	}

	normalizedRoomID := roomid.Normalize(*loc.RoomId)
	if normalizedRoomID == "" {
		loc.RoomId = nil
		return loc
	}

	loc.RoomId = &normalizedRoomID
	return loc
}
