//go:build integration

package events

import (
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestCommonCampusSpacesArePersistedEventLocationsWithBookingApproval(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "campus-spaces")
	organization := seedOrganization(t, db, "Campus Club")
	ctx := authedContext("campus-spaces")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, id := range []string{"ATRIUM", "COWORKING", "FULLGYM", "DININGHALL"} {
		room := &models.Room{}
		if err := db.NewSelect().Model(room).Where("id=?", id).Scan(ctx); err != nil {
			t.Fatalf("missing real room %s: %v", id, err)
		}
		if !room.IsActive || !room.IsBookable || room.Floor != 1 {
			t.Fatalf("venue %s cannot be selected for an event: %+v", id, room)
		}
		status := eventsv1.EventStatus_EVENT_STATUS_DRAFT
		floor := int32(1)
		response, err := svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{
			Title: "Campus gathering " + id, OrganizationId: organization.ID,
			StartTime: "2030-10-26T08:00:00Z", EndTime: "2030-10-26T09:00:00Z", Status: &status,
			StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: &id, RoomName: &room.Name, Floor: &floor},
		}))
		if err != nil {
			t.Fatalf("cannot create event in %s: %v", id, err)
		}
		if response.Msg.Event.GetStructuredLocation().GetRoomId() != id {
			t.Fatalf("event lost its venue: %v", response.Msg.Event)
		}
		readiness, err := svc.GetEventPublishReadiness(ctx, connect.NewRequest(&eventsv1.GetEventPublishReadinessRequest{EventId: response.Msg.Event.Id}))
		if err != nil || readiness.Msg.CanPublishNow || !readiness.Msg.RequiresRoomApproval {
			t.Fatalf("venue %s bypasses booking approval: %v, %v", id, readiness, err)
		}
	}
}
