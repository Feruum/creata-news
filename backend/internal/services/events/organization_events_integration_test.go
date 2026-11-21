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

func TestOrganizationEventsIncludeAcceptedCollaborations(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "organization-events")
	host := seedOrganization(t, db, "Host")
	guest := seedOrganization(t, db, "Collaborator")
	other := seedOrganization(t, db, "Other collaborator")
	shared := seedDraftEvent(t, db, actor, host.ID, "Shared upcoming", "Atrium")
	past := seedDraftEvent(t, db, actor, host.ID, "Shared past", "Atrium")
	draft := seedDraftEvent(t, db, actor, host.ID, "Shared draft", "Atrium")
	hidden := seedDraftEvent(t, db, actor, host.ID, "Not on map", "Atrium")
	pending := seedDraftEvent(t, db, actor, host.ID, "Pending collaboration", "Atrium")
	if _, err := db.ExecContext(context.Background(), "UPDATE events SET status='published' WHERE id IN (?,?,?,?)", shared.ID, past.ID, hidden.ID, pending.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE events SET start_time=NOW()+interval '1 day',end_time=NOW()+interval '2 days' WHERE id=?", shared.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "UPDATE events SET publish_on_map=false WHERE id=?", hidden.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int32{shared.ID, past.ID, draft.ID, hidden.ID} {
		row := models.EventCollaborator{EventID: id, OrganizationID: guest.ID, AddedAt: time.Now()}
		if _, err := db.NewInsert().Model(&row).Exec(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	extra := models.EventCollaborator{EventID: shared.ID, OrganizationID: other.ID, AddedAt: time.Now()}
	if _, err := db.NewInsert().Model(&extra).Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	for _, orgID := range []int32{host.ID, guest.ID} {
		first, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{OrganizationId: &orgID, Page: 1, Limit: 1}))
		if err != nil {
			t.Fatal(err)
		}
		wantTotal := int32(2)
		if orgID == host.ID {
			wantTotal++ // The host also sees its published event with an unaccepted invite.
		}
		if first.Msg.Total != wantTotal || len(first.Msg.Events) != 1 || first.Msg.Events[0].Id != shared.ID {
			t.Fatalf("organization %d: wrong first page or total: %v", orgID, first.Msg)
		}
		second, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{OrganizationId: &orgID, Page: 2, Limit: 1}))
		if err != nil || len(second.Msg.Events) != 1 || second.Msg.Events[0].Id != past.ID {
			t.Fatalf("wrong second page: %v, %v", second, err)
		}
	}
	upcoming := true
	upcomingEvents, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{OrganizationId: &guest.ID, UpcomingOnly: &upcoming}))
	if err != nil || upcomingEvents.Msg.Total != 1 || len(upcomingEvents.Msg.Events) != 1 || upcomingEvents.Msg.Events[0].Id != shared.ID {
		t.Fatalf("wrong upcoming events: %v, %v", upcomingEvents, err)
	}
	if _, err := db.NewDelete().Model((*models.EventCollaborator)(nil)).Where("event_id=? AND organization_id=?", shared.ID, guest.ID).Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	remaining, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{OrganizationId: &guest.ID}))
	if err != nil || remaining.Msg.Total != 1 || len(remaining.Msg.Events) != 1 || remaining.Msg.Events[0].Id != past.ID {
		t.Fatalf("removed collaboration is still listed: %v, %v", remaining, err)
	}
}

func TestEventReadsPopulateParticipants(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "participants")
	host := seedOrganization(t, db, "Host")
	guest := seedOrganization(t, db, "Collaborator")
	event := seedDraftEvent(t, db, actor, host.ID, "Shared event", "Atrium")
	seedEventSlug(t, db, event.ID, "participant-event")
	if _, err := db.ExecContext(context.Background(), "UPDATE events SET status='published' WHERE id=?", event.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), "INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(?,'collaborator',true)", guest.ID); err != nil {
		t.Fatal(err)
	}
	collaborator := models.EventCollaborator{EventID: event.ID, OrganizationID: guest.ID, AddedAt: time.Now()}
	organizer := models.EventOrganizer{EventID: event.ID, OrganizationID: guest.ID, UserID: actor, AddedAt: time.Now()}
	for _, row := range []any{&collaborator, &organizer} {
		if _, err := db.NewInsert().Model(row).Exec(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, nil, "", "")
	detail, err := svc.GetEvent(context.Background(), connect.NewRequest(&eventsv1.GetEventRequest{Id: event.ID}))
	if err != nil {
		t.Fatal(err)
	}
	slug, err := svc.GetEventBySlug(context.Background(), connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: "participant-event"}))
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{}))
	if err != nil || len(list.Msg.Events) != 1 {
		t.Fatalf("list events: %v, %v", list, err)
	}
	for _, got := range []*eventsv1.Event{detail.Msg.Event, slug.Msg.Event, list.Msg.Events[0]} {
		if len(got.Collaborators) != 1 || got.Collaborators[0].GetOrganizationSlug() != "collaborator" {
			t.Fatalf("missing collaborator: %v", got)
		}
		if len(got.Organizers) != 1 || got.Organizers[0].UserId != actor {
			t.Fatalf("missing organizer: %v", got)
		}
		if got.Organizers[0].Email != nil || got.Organizers[0].Username != nil {
			t.Fatal("public event participants include unnecessary personal details")
		}
	}
}
