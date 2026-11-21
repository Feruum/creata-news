//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestPublishedMetadataEditDoesNotRequireNewBooking(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "published-metadata-editor")
	org := seedOrganization(t, db, "Published metadata club")
	event := seedDraftEvent(t, db, actor, org.ID, "Original metadata", "Hall")
	ctx := authedContext("published-metadata-editor")
	location := &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("C1.3.366L")}
	if _, err := db.NewUpdate().Model((*models.Event)(nil)).Set("status = 'published'").Set("structured_location = ?::jsonb", string(serializeStructuredLocation(location))).Where("id = ?", event.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	description := "Updated description without changing the room or time"
	// Both clients may send unchanged schedule/location fields along with metadata.
	location.RoomId = ptrString(" с1.3.366l ")
	start, end := event.StartTime.Format(time.RFC3339), event.EndTime.Format(time.RFC3339)
	response, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, Description: &description, StructuredLocation: location, StartTime: &start, EndTime: &end}))
	if err != nil {
		t.Fatalf("metadata-only save unexpectedly required booking: %v", err)
	}
	if response.Msg.Event.Status != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED {
		t.Fatalf("metadata save changed publication status: %v", response.Msg.Event.Status)
	}
	var stored models.Event
	if err := db.NewSelect().Model(&stored).Where("id = ?", event.ID).Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !stored.StartTime.Equal(event.StartTime) || !stored.EndTime.Equal(event.EndTime) {
		t.Fatal("metadata save changed the event time")
	}
}

func TestPublishedScheduleProposalWaitsForApprovalAndRechecksOnSave(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "published-schedule-editor")
	org := seedOrganization(t, db, "Published schedule club")
	event := seedDraftEvent(t, db, actor, org.ID, "Published schedule", "Hall")
	ctx := authedContext("published-schedule-editor")
	if _, err := db.NewUpdate().Model((*models.Event)(nil)).Set("status = 'published'").Where("id = ?", event.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	start, end := event.StartTime.Add(3*time.Hour), event.EndTime.Add(3*time.Hour)
	startText, endText := start.Format(time.RFC3339), end.Format(time.RFC3339)
	location := &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: ptrString("с1.3.366l")}
	request := &eventsv1.GetEventPublishReadinessRequest{EventId: event.ID, ProposedStartTime: &startText, ProposedEndTime: &endText, ProposedLocation: location}
	update := &eventsv1.UpdateEventRequest{Id: event.ID, StartTime: &startText, EndTime: &endText, StructuredLocation: location}
	assertUnchanged := func() {
		t.Helper()
		var stored models.Event
		if err := db.NewSelect().Model(&stored).Where("id = ?", event.ID).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if stored.Status != models.EventStatusPublished || !stored.StartTime.Equal(event.StartTime) || string(stored.StructuredLocation) != string(event.StructuredLocation) {
			t.Fatalf("unapproved proposal modified public data: %+v", stored)
		}
	}
	check := func(want eventsv1.EventPublishReadinessState) {
		t.Helper()
		response, err := svc.GetEventPublishReadiness(ctx, connect.NewRequest(request))
		if err != nil || response.Msg.State != want {
			t.Fatalf("proposal readiness = %v, error = %v; want %v", response, err, want)
		}
		assertUnchanged()
	}
	check(eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_MISSING)
	booking := models.BookingRequest{RoomID: "C1.3.366L", EventID: &event.ID, OrganizationID: &org.ID, RequesterID: actor, Priority: models.BookingPriorityIndividual, Title: event.Title, StartTime: start, EndTime: end, Status: models.BookingRequestStatusPending}
	if _, err := db.NewInsert().Model(&booking).Returning("*").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	for _, state := range []struct {
		status models.BookingRequestStatus
		want   eventsv1.EventPublishReadinessState
	}{
		{models.BookingRequestStatusPending, eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_WAITING_ROOM_APPROVAL},
		{models.BookingRequestStatusRejected, eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_REJECTED},
		{models.BookingRequestStatusCancelled, eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_ROOM_BOOKING_CANCELLED},
	} {
		if _, err := db.NewUpdate().Model((*models.BookingRequest)(nil)).Set("status = ?", state.status).Where("id = ?", booking.ID).Exec(ctx); err != nil {
			t.Fatal(err)
		}
		check(state.want)
		if _, err := svc.UpdateEvent(ctx, connect.NewRequest(update)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("unapproved save accepted: %v", err)
		}
		assertUnchanged()
	}
	setStatus := func(status models.BookingRequestStatus) {
		t.Helper()
		if _, err := db.NewUpdate().Model((*models.BookingRequest)(nil)).Set("status = ?", status).Where("id = ?", booking.ID).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	setStatus(models.BookingRequestStatusApproved)
	check(eventsv1.EventPublishReadinessState_EVENT_PUBLISH_READINESS_STATE_READY)
	// Approval can be revoked between the read-only check and the final mutation.
	setStatus(models.BookingRequestStatusCancelled)
	if _, err := svc.UpdateEvent(ctx, connect.NewRequest(update)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("save used stale approval: %v", err)
	}
	assertUnchanged()
	setStatus(models.BookingRequestStatusApproved)
	response, err := svc.UpdateEvent(ctx, connect.NewRequest(update))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Event.Status != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED || response.Msg.Event.StructuredLocation.GetRoomId() != "C1.3.366L" || response.Msg.Event.StartTime != startText {
		t.Fatalf("approved scheduling changes were not saved: %v", response.Msg.Event)
	}
}
