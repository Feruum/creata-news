package events

import (
	"connectrpc.com/connect"
	"context"
	"encoding/json"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
)

func TestEventTimeRangeRejectsMalformedAndReversedDates(t *testing.T) {
	for _, input := range [][2]string{{"invalid", "2026-10-02T12:00:00Z"}, {"2026-10-02T12:00:00Z", "invalid"}, {"2026-10-02T12:00:00Z", "2026-10-02T11:00:00Z"}, {"2026-10-02T12:00:00Z", "2026-10-02T12:00:00Z"}} {
		if _, _, err := parseEventTimeRange(input[0], input[1]); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("accepted %v: %v", input, err)
		}
	}
	if _, _, err := parseEventTimeRange("2026-10-02T12:00:00+05:00", "2026-10-02T08:00:00Z"); err != nil {
		t.Fatal(err)
	}
}

func TestCampusPublicationRequiresValidRoomID(t *testing.T) {
	svc := &Service{}
	for _, locationType := range []string{"campus_room", "hybrid"} {
		for _, room := range []string{"", " "} {
			event := &models.Event{Status: models.EventStatusDraft, StructuredLocation: json.RawMessage(`{"type":"` + locationType + `","room_id":"` + room + `"}`)}
			if err := svc.ensureRoomBookingReadyForPublish(context.Background(), event); connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("accepted %s room %q: %v", locationType, room, err)
			}
		}
	}
}

func TestCampusRoomCreationMustStartAsDraft(t *testing.T) {
	location := &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM}
	if _, err := initialEventStatus(nil, location); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("room bypass accepted: %v", err)
	}
	draft := eventsv1.EventStatus_EVENT_STATUS_DRAFT
	if _, err := initialEventStatus(&draft, location); err != nil {
		t.Fatal(err)
	}
	if _, err := initialEventStatus(nil, &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE}); err != nil {
		t.Fatal(err)
	}
}
