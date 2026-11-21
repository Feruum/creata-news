//go:build integration

package rooms

import (
	"encoding/json"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	eventservice "github.com/studyverse/ems-backend/internal/services/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestRoomApprovalPreservesEventLinkAndPublicationWorkflow(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "room-workflow-admin")
	org := seedOrganization(t, db, "Room workflow organization")
	room := seedRoom(t, db, "A-101", "Workshop room")
	event := seedEvent(t, db, actor, org.ID, "Campus workshop")
	ctx := authedContext("room-workflow-admin")
	start := time.Now().Add(time.Hour)
	if _, err := db.NewUpdate().Model(&event).
		Set("status = ?", models.EventStatusDraft).
		Set("start_time = ?", start).Set("end_time = ?", start.Add(time.Hour)).
		Set("structured_location = ?", `{"type":"campus_room","room_id":"A-101"}`).WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	bookingReq := seedBookingRequest(t, db, actor, org.ID, event.ID, room.ID, event.Title)
	if _, err := db.NewUpdate().Model(&bookingReq).Set("start_time = ?", start).
		Set("end_time = ?", start.Add(time.Hour)).WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	pub := domainevents.NewDurablePublisher(db, nil)
	events := eventservice.NewService(db, nil, enforcer, nil, pub, "", "")
	svc := NewService(db, nil, enforcer, pub)
	if _, err := events.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("pending room approval must block publication: %v", err)
	}
	approved, err := svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: bookingReq.ID}))
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Msg.Booking.EventId == nil || *approved.Msg.Booking.EventId != event.ID {
		t.Errorf("approved room booking lost linked event: %v", approved.Msg.Booking)
	}
	var payload struct {
		EventID int32 `json:"event_id"`
	}
	var rawPayload []byte
	if err := db.NewRaw("SELECT payload FROM domain_event_outbox WHERE topic = ?", domainevents.TopicBookingApproved).Scan(ctx, &rawPayload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.EventID != event.ID {
		t.Errorf("booking approval notification lost linked event: got %d, want %d", payload.EventID, event.ID)
	}
	if _, err := svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: bookingReq.ID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("duplicate approval must be rejected: %v", err)
	}
	if _, err := events.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID})); err != nil {
		t.Fatalf("publish approved campus event: %v", err)
	}
}

func TestCreateBookingRequestRejectsReversedTimeRange(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "booking-time-user")
	room := seedRoom(t, db, "A-102", "Second workshop room")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil)
	start := time.Now().Add(time.Hour)
	_, err := svc.CreateBookingRequest(authedContext("booking-time-user"), connect.NewRequest(&roomsv1.CreateBookingRequestRequest{
		RoomId: room.ID, Title: "Reversed booking", StartTime: start.Format(time.RFC3339), EndTime: start.Add(-time.Hour).Format(time.RFC3339),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("reversed booking times must be invalid_argument, got %v", err)
	}
}
