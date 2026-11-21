package events

import (
	"testing"
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestOnlyReplacedPrimaryReservationIsReleased(t *testing.T) {
	start := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)
	id := int32(10)
	original := models.Event{ID: id, StartTime: start, EndTime: start.Add(time.Hour), StructuredLocation: serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("C1.3.366L")})}
	candidate := original
	candidate.EndTime = start.Add(2 * time.Hour)
	old := models.RoomBooking{EventID: &id, RoomID: "С1.3.366L", StartTime: start, EndTime: original.EndTime}
	if !replacedPrimaryBooking(original, candidate, old) {
		t.Fatal("superseded primary reservation was retained")
	}
	replacement := old
	replacement.EndTime = candidate.EndTime
	if replacedPrimaryBooking(original, candidate, replacement) {
		t.Fatal("new covering reservation would be removed")
	}
	unrelated := old
	unrelated.RoomID = "C1.3.367L"
	if replacedPrimaryBooking(original, candidate, unrelated) {
		t.Fatal("unrelated secondary room would be removed")
	}
	if replacedPrimaryBooking(original, original, old) {
		t.Fatal("unchanged schedule would release its reservation")
	}
	otherID := int32(11)
	old.EventID = &otherID
	if replacedPrimaryBooking(original, candidate, old) {
		t.Fatal("another event's reservation would be removed")
	}
}
