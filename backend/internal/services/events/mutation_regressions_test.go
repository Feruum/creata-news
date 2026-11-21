//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestUpcomingEventsAreFilteredBeforePagination(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO events (title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
		SELECT 'past','{}',1,1,NOW()-interval '2 days',NOW()-interval '1 day','{}','published',i::text,i::text FROM generate_series(1,150) i;
		INSERT INTO events (title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
		VALUES ('later','{}',1,1,NOW()+interval '2 hours',NOW()+interval '3 hours','{}','published','later','later'),
		('sooner','{}',1,1,NOW()+interval '1 hour',NOW()+interval '2 hours','{}','published','sooner','sooner');`); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, nil), nil, nil, "", "")
	upcoming := true
	for page, expected := range []string{"sooner", "later"} {
		resp, err := svc.ListEvents(ctx, connect.NewRequest(&eventsv1.ListEventsRequest{Page: int32(page + 1), Limit: 1, UpcomingOnly: &upcoming}))
		if err != nil {
			t.Fatal(err)
		}
		if resp.Msg.Total != 2 || len(resp.Msg.Events) != 1 || resp.Msg.Events[0].Title != expected {
			t.Fatalf("page%d upcoming selection incorrect: %v", page+1, resp.Msg)
		}
	}
}

func TestPublishedEventCannotBeMovedIntoUnbookedCampusRoom(t *testing.T) {
	db := newTestDB(t)
	ctx := authedContext("room-editor")
	actor := seedAuthUser(t, db, "room-editor")
	org := seedOrganization(t, db, "Room Organization")
	event := seedDraftEvent(t, db, actor, org.ID, "Published online", "Hall")
	if _, err := db.NewUpdate().Model((*models.Event)(nil)).Set("status='published'").Where("id=?", event.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, location := range []*eventsv1.EventLocation{{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM}, {Type: eventsv1.LocationType_LOCATION_TYPE_HYBRID, RoomId: ptrString("unknown-room")}} {
		if _, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, StructuredLocation: location})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("accepted unbooked published location: %v", err)
		}
	}
	var persisted models.Event
	if err := db.NewSelect().Model(&persisted).Where("id=?", event.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if string(persisted.StructuredLocation) != string(event.StructuredLocation) {
		t.Fatalf("location changed despite rejection: %s", persisted.StructuredLocation)
	}
}

func TestEventMutationsRollBackWhenOutboxInsertFails(t *testing.T) {
	for _, operation := range []string{"create", "update", "publish", "delete"} {
		t.Run(operation, func(t *testing.T) {
			db := newTestDB(t)
			actorID := seedAuthUser(t, db, "outbox-actor")
			org := seedOrganization(t, db, "Outbox Organization")
			event := seedDraftEvent(t, db, actorID, org.ID, "Original", "Hall")
			if _, err := db.ExecContext(context.Background(), "ALTER TABLE domain_event_outbox ADD CONSTRAINT injected_failure CHECK (false)"); err != nil {
				t.Fatal(err)
			}
			svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, domainevents.NewDurablePublisher(db, nil), "", "")
			ctx := context.WithValue(authedContext("outbox-actor"), auth.UserEmailKey, "outbox_actor@test.local")
			var err error
			switch operation {
			case "create":
				_, err = svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{Title: "Created", Description: `"description"`, OrganizationId: org.ID, StartTime: time.Now().Add(time.Hour).Format(time.RFC3339), EndTime: time.Now().Add(2 * time.Hour).Format(time.RFC3339)}))
			case "update":
				name := "Changed"
				_, err = svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, Title: &name}))
			case "publish":
				_, err = svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
			case "delete":
				_, err = svc.DeleteEvent(ctx, connect.NewRequest(&eventsv1.DeleteEventRequest{Id: event.ID}))
			}
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("enqueue failure returned success/wrong error: %v", err)
			}
			count, countErr := db.NewSelect().Model((*models.Event)(nil)).Count(context.Background())
			if countErr != nil || count != 1 {
				t.Fatalf("event mutation escaped rollback: %d %v", count, countErr)
			}
			var stored models.Event
			if err := db.NewSelect().Model(&stored).Where("id=?", event.ID).Scan(context.Background()); err != nil {
				t.Fatal(err)
			}
			if stored.Title != "Original" || stored.Status != models.EventStatusDraft {
				t.Fatalf("event changed after failure: title=%s status=%s", stored.Title, stored.Status)
			}
		})
	}
}
