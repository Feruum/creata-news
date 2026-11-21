//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	sundayopenv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/services"
	"github.com/studyverse/ems-backend/internal/services/sundayopen"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestSundayOpenEventAppearsInBothPublicCatalogs(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "shared-publication-admin")
	org := seedSundayOpenPublicationHost(t, db)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, "", "")
	start := time.Now().Add(time.Hour)
	created, err := svc.CreateEvent(authedContext("shared-publication-admin"), connect.NewRequest(&eventsv1.CreateEventRequest{
		Title: "Sunday shared publication", OrganizationId: org.ID,
		StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
		StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
	}))
	if err != nil {
		t.Fatal(err)
	}
	id := created.Msg.Event.Id
	ctx := context.Background()
	upcoming := true
	listed, err := svc.ListEvents(ctx, connect.NewRequest(&eventsv1.ListEventsRequest{UpcomingOnly: &upcoming}))
	if err != nil {
		t.Fatal(err)
	}
	if listed.Msg.Total != 1 || len(listed.Msg.Events) != 1 || listed.Msg.Events[0].Id != id {
		t.Fatalf("Sunday Open event missing from map Discover: %v", listed.Msg)
	}
	if _, err := svc.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: id})); err != nil {
		t.Fatalf("map event detail unavailable: %v", err)
	}
	if _, err := svc.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: "sunday-shared-publication"})); err != nil {
		t.Fatalf("map event slug unavailable: %v", err)
	}
	sunday := sundayopen.NewService(db, enforcer)
	sundayListed, err := sunday.ListPublicEvents(ctx, connect.NewRequest(&sundayopenv1.ListPublicEventsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if sundayListed.Msg.Total != 1 || len(sundayListed.Msg.Events) != 1 || sundayListed.Msg.Events[0].Id != id {
		t.Fatalf("event missing from Sunday Open catalog: %v", sundayListed.Msg)
	}
}

func TestAcceptedSundayOpenCohostAppearsInBothPublicCatalogs(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "shared-cohost-admin")
	club := seedOrganization(t, db, "Main club")
	sunday := seedSundayOpenPublicationHost(t, db)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, "", "")
	sundayService := sundayopen.NewService(db, enforcer)
	ctx := authedContext("shared-cohost-admin")
	start := time.Now().Add(time.Hour)
	created, err := svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{
		Title: "Accepted Sunday Open cohost", OrganizationId: club.ID,
		StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
		StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
	}))
	if err != nil {
		t.Fatal(err)
	}
	id := created.Msg.Event.Id
	if _, err := db.ExecContext(ctx, "INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered')", id, actor); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO external_registrations(event_id,full_name,email) VALUES(?,'Guest','guest@example.invalid')", id); err != nil {
		t.Fatal(err)
	}
	var original models.Event
	if err := db.NewSelect().Model(&original).Where("id=?", id).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	assertCatalogues := func(wantSunday int32) {
		t.Helper()
		mapList, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{}))
		if err != nil || mapList.Msg.Total != 1 || len(mapList.Msg.Events) != 1 || mapList.Msg.Events[0].Id != id {
			t.Fatalf("cohost mutation must retain map Discover: list=%v err=%v", mapList, err)
		}
		sundayList, err := sundayService.ListPublicEvents(context.Background(), connect.NewRequest(&sundayopenv1.ListPublicEventsRequest{}))
		if err != nil || sundayList.Msg.Total != wantSunday || len(sundayList.Msg.Events) != int(wantSunday) {
			t.Fatalf("wrong Sunday Open catalogue after cohost mutation: list=%v err=%v", sundayList, err)
		}
		if wantSunday == 1 && sundayList.Msg.Events[0].Id != id {
			t.Fatalf("wrong Sunday Open event: %v", sundayList.Msg.Events)
		}
	}
	assertCatalogues(0)
	invitation, err := svc.InviteOrganizationToCollaborate(ctx, connect.NewRequest(&eventsv1.InviteOrganizationToCollaborateRequest{EventId: id, OrganizationId: sunday.ID}))
	if err != nil {
		t.Fatal(err)
	}
	assertCatalogues(0)
	if _, err := svc.AcceptCollaborationInvite(ctx, connect.NewRequest(&eventsv1.AcceptCollaborationInviteRequest{Token: invitation.Msg.Invite.Token})); err != nil {
		t.Fatal(err)
	}
	assertCatalogues(1)
	assertCollaborationPublication(t, db, original, true, true)
	if _, err := svc.RemoveEventCollaborator(ctx, connect.NewRequest(&eventsv1.RemoveEventCollaboratorRequest{EventId: id, OrganizationId: sunday.ID})); err != nil {
		t.Fatal(err)
	}
	assertCatalogues(0)
	assertCollaborationPublication(t, db, original, true, false)
	for _, table := range []string{"event_registrations", "external_registrations"} {
		count, err := db.NewSelect().Table(table).Where("event_id=?", id).Count(ctx)
		if err != nil || count != 1 {
			t.Fatalf("cohost mutation changed %s: count=%d err=%v", table, count, err)
		}
	}
}

func TestPublicationTargetsSeparateMapCatalogAndRegistration(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "publication-admin")
	org := seedOrganization(t, db, "Publication organization")
	sundayOrg := seedSundayOpenPublicationHost(t, db)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, "", "")
	ctx := authedContext("publication-admin")
	on, off := true, false
	start := time.Now().Add(time.Hour)
	create := func(title string, orgID int32, onMap, onSunday *bool, status *eventsv1.EventStatus) (*connect.Response[eventsv1.CreateEventResponse], error) {
		return svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{
			Title: title, Description: "Publication integration test", OrganizationId: orgID,
			StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339),
			StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
			PublishOnMap:       onMap, PublishOnSundayOpen: onSunday, Status: status,
		}))
	}
	legacy, err := create("Legacy default map publication", org.ID, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.Msg.Event.PublishOnMap || legacy.Msg.Event.PublishOnSundayOpen {
		t.Fatal("legacy create defaults changed")
	}
	sunday, err := create("Sunday only publication", sundayOrg.ID, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Legacy flags must still control visibility independently of the host.
	if _, err := db.ExecContext(ctx, "UPDATE events SET publish_on_map=false WHERE id=?", sunday.Msg.Event.Id); err != nil {
		t.Fatal(err)
	}
	public, err := svc.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if public.Msg.Total != 1 || len(public.Msg.Events) != 1 || public.Msg.Events[0].Id != legacy.Msg.Event.Id {
		t.Fatalf("wrong map catalog: %v", public.Msg)
	}
	if _, err := svc.GetEvent(context.Background(), connect.NewRequest(&eventsv1.GetEventRequest{Id: sunday.Msg.Event.Id})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Sunday event exposed by ID: %v", err)
	}
	if _, err := svc.GetEventBySlug(context.Background(), connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: "sunday-only-publication"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Sunday event exposed by slug: %v", err)
	}
	if _, err := svc.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: sunday.Msg.Event.Id})); err != nil {
		t.Fatalf("editor preview denied: %v", err)
	}
	admin, err := svc.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(admin.Msg.Events) != 2 {
		t.Fatal("admin list lost a publication platform")
	}
	updated, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: sunday.Msg.Event.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.Event.PublishOnMap || !updated.Msg.Event.PublishOnSundayOpen {
		t.Fatal("omitted update changed flags")
	}
	updated, err = svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: sunday.Msg.Event.Id, PublishOnMap: &on, PublishOnSundayOpen: &off}))
	if err != nil || updated.Msg.Event.PublishOnMap || !updated.Msg.Event.PublishOnSundayOpen {
		t.Fatalf("client changed stored publication targets: response=%v err=%v", updated, err)
	}
	registrations := services.NewEventRegistrationsService(db, nil, enforcer)
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: sunday.Msg.Event.Id, UserId: actor})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("map registration accepted Sunday-only event: %v", err)
	}
	// An old registration must not reveal a newly Sunday-only event in map lists.
	if _, err := db.ExecContext(ctx, "INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered')", sunday.Msg.Event.Id, actor); err != nil {
		t.Fatal(err)
	}
	subscribed, err := svc.GetUserSubscribedEvents(ctx, connect.NewRequest(&eventsv1.GetUserSubscribedEventsRequest{UserId: actor}))
	if err != nil {
		t.Fatal(err)
	}
	if len(subscribed.Msg.Events) != 0 {
		t.Fatal("map subscribed events exposed a Sunday-only event")
	}
	automatic, err := create("Automatic publication target", org.ID, &off, &off, nil)
	if err != nil || !automatic.Msg.Event.PublishOnMap || automatic.Msg.Event.PublishOnSundayOpen {
		t.Fatalf("client flags prevented automatic map publication: response=%v err=%v", automatic, err)
	}
	draftStatus := eventsv1.EventStatus_EVENT_STATUS_DRAFT
	draft, err := create("Unpublished draft", org.ID, &off, &off, &draftStatus)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: draft.Msg.Event.Id})); err != nil {
		t.Fatalf("publish draft with automatic targets: %v", err)
	}
}
