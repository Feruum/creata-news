//go:build integration

package events

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestEventReadsReturnRegistrationAndAttendanceCounts(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "counts-admin")
	org := seedOrganization(t, db, "Counts organization")
	event := seedDraftEvent(t, db, actor, org.ID, "Counts event", "Hall")
	empty := seedDraftEvent(t, db, actor, org.ID, "Empty event", "Hall")
	ctx := authedContext("counts-admin")
	for _, status := range []string{"registered", "registered", "cancelled"} {
		if _, err := db.ExecContext(ctx, "INSERT INTO external_registrations(event_id,full_name,email,status) VALUES(?,?,?,?)", event.ID, "Sunday participant", "participant@example.test", status); err != nil {
			t.Fatal(err)
		}
	}
	for i, state := range []struct{ registration, attendance string }{
		{"registered", "attended"},
		{"registered", "checked_in"},
		{"cancelled", "attended"},
		{"cancelled", "checked_in"},
		{"cancelled", "no_show"},
		{"waitlist", "attended"},
		{"waitlist", "checked_in"},
		{"waitlist", "no_show"},
	} {
		user := seedAuthUser(t, db, fmt.Sprintf("counts-user-%d", i))
		var id int32
		if err := db.NewRaw("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,?) RETURNING id", event.ID, user, state.registration).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO event_attendance(registration_id,status) VALUES(?,?)", id, state.attendance); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	detail, err := svc.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: event.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if detail.Msg.Event.TotalRegistrations != 2 || detail.Msg.Event.TotalAttendees != 2 {
		t.Fatalf("wrong detail counts: %v", detail.Msg.Event)
	}
	if detail.Msg.Event.ExternalRegistrationCount != 2 || detail.Msg.Event.AllocatedSeatCount != 4 {
		t.Fatalf("wrong combined capacity counts: %v", detail.Msg.Event)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO event_slugs(event_id,slug,is_current) VALUES(?,'counts-event',true)", event.ID); err != nil {
		t.Fatal(err)
	}
	slugDetail, err := svc.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: "counts-event"}))
	if err != nil {
		t.Fatal(err)
	}
	if slugDetail.Msg.Event.TotalRegistrations != 2 || slugDetail.Msg.Event.TotalAttendees != 2 {
		t.Fatal("slug detail has inconsistent counts")
	}
	list, err := svc.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int32]bool{}
	for _, got := range list.Msg.Events {
		seen[got.Id] = true
		if got.Id == event.ID && (got.TotalRegistrations != 2 || got.TotalAttendees != 2) {
			t.Fatalf("wrong list counts: %v", got)
		}
		if got.Id == empty.ID && (got.TotalRegistrations != 0 || got.TotalAttendees != 0) {
			t.Fatal("empty event has counts")
		}
	}
	if !seen[event.ID] || !seen[empty.ID] {
		t.Fatal("events missing from admin list")
	}
}

func TestEventReadsPropagateAttendanceCountFailure(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "counts-storage-admin")
	org := seedOrganization(t, db, "Counts storage organization")
	event := seedDraftEvent(t, db, actor, org.ID, "Counts storage event", "Hall")
	ctx := authedContext("counts-storage-admin")
	seedEventSlug(t, db, event.ID, "counts-storage-event")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	if _, err := db.ExecContext(ctx, "ALTER TABLE event_attendance RENAME TO unavailable_event_attendance"); err != nil {
		t.Fatal(err)
	}
	for name, read := range map[string]func() error{
		"detail": func() error {
			_, err := svc.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: event.ID}))
			return err
		},
		"slug detail": func() error {
			_, err := svc.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: "counts-storage-event"}))
			return err
		},
		"admin list": func() error {
			_, err := svc.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{Limit: 100}))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := read(); connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("attendance count failure should be internal, got %v", err)
			}
		})
	}
}
