package events

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestBookingDetailsCompareEffectiveRoomAndInstants(t *testing.T) {
	start := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)
	original := models.Event{StartTime: start, EndTime: start.Add(time.Hour), StructuredLocation: serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("C1.3.366L")})}
	unchanged := original
	unchanged.StructuredLocation = serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString(" с1.3.366l "), RoomName: ptrString("New display label")})
	unchanged.StartTime = start.In(time.FixedZone("+05", 5*3600))
	if bookingDetailsChanged(original, unchanged) {
		t.Fatal("normalized aliases, labels, and equal instants must not request a new approval")
	}
	for _, change := range []func(*models.Event){
		func(e *models.Event) { e.StartTime = start.Add(time.Minute) },
		func(e *models.Event) { e.EndTime = e.EndTime.Add(time.Minute) },
		func(e *models.Event) {
			e.StructuredLocation = serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("C1.3.367L")})
		},
		func(e *models.Event) {
			e.StructuredLocation = serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_HYBRID, RoomId: ptrString("C1.3.366L")})
		},
	} {
		candidate := original
		change(&candidate)
		if !bookingDetailsChanged(original, candidate) {
			t.Fatal("changed room, format, or time was ignored")
		}
	}
}

func TestProposalDoesNotMutatePublishedEventOrRequest(t *testing.T) {
	start := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)
	original := models.Event{Status: models.EventStatusPublished, StartTime: start, EndTime: start.Add(time.Hour)}
	nextStart, nextEnd := start.Add(time.Hour).Format(time.RFC3339), start.Add(2*time.Hour).Format(time.RFC3339)
	request := &eventsv1.GetEventPublishReadinessRequest{ProposedStartTime: &nextStart, ProposedEndTime: &nextEnd, ProposedLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("с1.3.366l")}}
	candidate, err := proposedPublicationEvent(original, request)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Status != models.EventStatusDraft || !candidate.StartTime.Equal(start.Add(time.Hour)) || parseStructuredLocation(candidate.StructuredLocation).GetRoomId() != "C1.3.366L" {
		t.Fatalf("incorrect proposal: %+v", candidate)
	}
	if original.Status != models.EventStatusPublished || !original.StartTime.Equal(start) || request.ProposedLocation.GetRoomId() != "с1.3.366l" {
		t.Fatal("read-only proposal mutated stored state or request")
	}
	invalid := "invalid date"
	request.ProposedStartTime = &invalid
	if _, err := proposedPublicationEvent(original, request); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("malformed proposal accepted: %v", err)
	}
}

func TestBookingSelectionKeepsCoveringApprovalWhileNewProposalWaits(t *testing.T) {
	start := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)
	event := models.Event{StartTime: start, EndTime: start.Add(time.Hour), StructuredLocation: serializeStructuredLocation(&eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("C1.3.366L")})}
	requests := []models.BookingRequest{
		{ID: 3, RoomID: "C1.3.366L", Status: models.BookingRequestStatusPending, StartTime: start.Add(-time.Hour), EndTime: start.Add(2 * time.Hour)},
		{ID: 2, RoomID: "С1.3.366L", Status: models.BookingRequestStatusApproved, StartTime: start, EndTime: start.Add(time.Hour)},
	}
	if booking := selectCoveringBooking(event, requests); booking == nil || booking.ID != 2 {
		t.Fatalf("new proposal hid the existing approval: %+v", booking)
	}
	event.EndTime = start.Add(2 * time.Hour)
	if booking := selectCoveringBooking(event, requests); booking == nil || booking.ID != 3 {
		t.Fatalf("new interval used an approval that does not cover it: %+v", booking)
	}
	requests[0].RoomID = "C1.3.367L"
	if booking := selectCoveringBooking(event, requests); booking != nil {
		t.Fatalf("another room or a partial interval approved the proposal: %+v", booking)
	}
}
