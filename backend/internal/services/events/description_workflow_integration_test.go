//go:build integration

package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestEventDescriptionEditingAcceptsTextClearAndRichContent(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "description-editor")
	org := seedOrganization(t, db, "Description club")
	event := seedDraftEvent(t, db, actor, org.ID, "Description editing", "Hall")
	ctx := authedContext("description-editor")
	// The current shared schema uses JSONB; the initial event SQL migration used TEXT.
	if _, err := db.ExecContext(ctx, "ALTER TABLE events ALTER COLUMN description TYPE jsonb USING description::jsonb"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, domainevents.NewDurablePublisher(db, nil), "", "")
	for _, test := range []struct{ name, input, expectedJSON string }{
		{"plain text", "Updated description\nSecond line", `"Updated description\nSecond line"`},
		{"clear", "", `""`},
		{"rich content", `[{"type":"p","children":[{"text":"Rich description"}]}]`, `[{"type":"p","children":[{"text":"Rich description"}]}]`},
		{"JSON string", `"Saved plain text"`, `"Saved plain text"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := svc.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, Description: &test.input}))
			if err != nil {
				t.Fatalf("description edit failed: %v", err)
			}
			var stored models.Event
			if err := db.NewSelect().Model(&stored).Where("id=?", event.ID).Scan(ctx); err != nil {
				t.Fatal(err)
			}
			var expected any
			if err := json.Unmarshal([]byte(test.expectedJSON), &expected); err != nil {
				t.Fatal(err)
			}
			for _, encoded := range []string{response.Msg.Event.Description, string(stored.Description)} {
				var actual any
				if err := json.Unmarshal([]byte(encoded), &actual); err != nil || !reflect.DeepEqual(actual, expected) {
					t.Fatalf("description changed or failed to persist: got=%s expected=%s err=%v", encoded, test.expectedJSON, err)
				}
			}
		})
	}
}

func TestCreateEventRollsBackWhenSlugInsertFails(t *testing.T) {
	db := newTestDB(t)
	seedAuthUser(t, db, "slug-failure-creator")
	org := seedOrganization(t, db, "Slug rollback club")
	ctx := authedContext("slug-failure-creator")
	if _, err := db.ExecContext(ctx, "ALTER TABLE event_slugs ADD CONSTRAINT injected_slug_failure CHECK (false) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil, domainevents.NewDurablePublisher(db, nil), "", "")
	_, err := svc.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{Title: "Must not survive", Description: "Description", OrganizationId: org.ID,
		StartTime: time.Now().Add(time.Hour).Format(time.RFC3339), EndTime: time.Now().Add(2 * time.Hour).Format(time.RFC3339)}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("expected failed slug insert, got %v", err)
	}
	for _, table := range []string{"events", "event_slugs", "domain_event_outbox"} {
		count, err := db.NewSelect().Table(table).Count(ctx)
		if err != nil || count != 0 {
			t.Fatalf("failed creation persisted %s rows: count=%d err=%v", table, count, err)
		}
	}
}
