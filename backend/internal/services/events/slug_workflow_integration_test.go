//go:build integration

package events

import (
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestEventEditsKeepCanonicalSlugUntilTitleChanges(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "slug-editor")
	org := seedOrganization(t, db, "Slug editing club")
	event := seedDraftEvent(t, db, actor, org.ID, "Stable event link", "Hall")
	seedEventSlug(t, db, event.ID, "stable-event-link")
	ctx := authedContext("slug-editor")
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, domainevents.NewDurablePublisher(db, nil), "", "")
	assertSlugs := func(current string, count int) {
		t.Helper()
		var slugs []models.EventSlug
		if err := db.NewSelect().Model(&slugs).Where("event_id = ?", event.ID).Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if len(slugs) != count {
			t.Fatalf("unexpected alias count: got %d want %d; slugs=%+v", len(slugs), count, slugs)
		}
		currentCount := 0
		for _, alias := range slugs {
			if alias.IsCurrent {
				currentCount++
				if alias.Slug != current {
					t.Fatalf("canonical slug changed: got %q want %q", alias.Slug, current)
				}
			}
		}
		if currentCount != 1 {
			t.Fatalf("expected one canonical slug, got %d", currentCount)
		}
	}
	description, image := "Updated description", "https://example.test/cover.png"
	for _, input := range []*eventsv1.UpdateEventRequest{
		{Id: event.ID, Title: &event.Title, Description: &description},
		{Id: event.ID, Title: &event.Title, ImageUrl: &image},
	} {
		if _, err := svc.UpdateEvent(ctx, connect.NewRequest(input)); err != nil {
			t.Fatal(err)
		}
		assertSlugs("stable-event-link", 1)
	}
	var stored models.Event
	if err := db.NewSelect().Model(&stored).Where("id = ?", event.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if string(stored.Description) != `"Updated description"` || stored.ImageUrl == nil || *stored.ImageUrl != image {
		t.Fatalf("content edits did not persist: description=%s image=%v", stored.Description, stored.ImageUrl)
	}
	newTitle := "Renamed event link"
	if _, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, Title: &newTitle})); err != nil {
		t.Fatal(err)
	}
	assertSlugs("renamed-event-link", 2)
	for _, alias := range []string{"stable-event-link", "renamed-event-link"} {
		response, err := svc.GetEventBySlug(ctx, connect.NewRequest(&eventsv1.GetEventBySlugRequest{Slug: alias}))
		if err != nil || response.Msg.Event.Id != event.ID {
			t.Fatalf("alias %q no longer resolves to event: response=%v err=%v", alias, response, err)
		}
	}
}
