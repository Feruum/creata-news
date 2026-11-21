//go:build integration

package sundayopen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/studyverse/ems-backend/gen/sundayopen/v1/sundayopenv1connect"
	"github.com/uptrace/bun"
)

type publicHostQueryCounter struct{ queries atomic.Int32 }

func (c *publicHostQueryCounter) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (c *publicHostQueryCounter) AfterQuery(_ context.Context, event *bun.QueryEvent) {
	if strings.Contains(event.Query, "event_collaborators AS ec") {
		c.queries.Add(1)
	}
}

func TestPublicHostsRoundTripIncludesOnlyAcceptedOrganizations(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE event_collaboration_invites (LIKE public.event_collaboration_invites INCLUDING ALL)`,
		`CREATE TABLE event_organizers (LIKE public.event_organizers INCLUDING ALL)`,
		`INSERT INTO organizations(id,title,image_url,organization_type_id) VALUES(3,'Pending private club',NULL,1),(4,'AITU Gaming','https://example.test/gaming.png',1)`,
		`INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(4,'aitu-gaming',true),(4,'old-gaming',false)`,
		`INSERT INTO event_collaborators(event_id,organization_id,added_by,added_at) VALUES(1,2,42,'2026-01-01'),(1,4,42,'2026-01-02'),(1,1,42,'2026-01-03')`,
		`INSERT INTO event_collaboration_invites(event_id,organization_id,invited_by_user_id,token,status,expires_at,message) VALUES(1,3,42,'private-invitation-token','pending',NOW()+INTERVAL '1 day','Private invitation message')`,
		`INSERT INTO event_organizers(event_id,user_id,organization_id,added_by) VALUES(1,42,1,42)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	counter := &publicHostQueryCounter{}
	db.AddQueryHook(counter)
	_, handler := sundayopenv1connect.NewSundayOpenServiceHandler(NewService(db, nil))
	server := httptest.NewServer(handler)
	defer server.Close()

	request := func(method, body string) map[string]any {
		t.Helper()
		response, err := http.Post(server.URL+"/sundayopen.v1.SundayOpenService/"+method, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s failed: %v", method, payload)
		}
		return payload
	}

	feed := request("ListPublicEvents", `{}`)
	events := feed["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("unexpected public events: %v", events)
	}
	first := events[0].(map[string]any)
	if first["organizationId"] != float64(1) || first["organizationTitle"] != "Sunday Open" || first["organizationSlug"] != "sunday-open" {
		t.Fatalf("main host identity changed or missing: %v", first)
	}
	want := []any{
		map[string]any{"organizationId": float64(2), "organizationTitle": "Other club", "organizationSlug": "other-club"},
		map[string]any{"organizationId": float64(4), "organizationTitle": "AITU Gaming", "organizationImageUrl": "https://example.test/gaming.png", "organizationSlug": "aitu-gaming"},
	}
	if !reflect.DeepEqual(first["collaborators"], want) {
		t.Fatalf("accepted public collaborators: got %v, want %v", first["collaborators"], want)
	}
	second := events[1].(map[string]any)
	if collaborators, ok := second["collaborators"].([]any); ok && len(collaborators) != 0 {
		t.Fatalf("unrelated event inherited collaborators: %v", second)
	}
	if counter.queries.Load() != 1 {
		t.Fatalf("feed must batch hosts in one query, got %d queries", counter.queries.Load())
	}
	detail := request("GetEventBySlug", `{"slug":"event-1"}`)["event"].(map[string]any)
	if !reflect.DeepEqual(detail, first) {
		t.Fatalf("detail and feed public identities differ: %v / %v", detail, first)
	}
	withoutCollaborators := request("GetEventBySlug", `{"slug":"event-2"}`)["event"].(map[string]any)
	if !reflect.DeepEqual(withoutCollaborators, second) {
		t.Fatalf("event without collaborators changed between list and detail: %v / %v", withoutCollaborators, second)
	}
	serialized, _ := json.Marshal(detail)
	for _, private := range []string{"Pending private club", "private-invitation-token", "Private invitation message", "addedBy", "userId", "organizers"} {
		if strings.Contains(string(serialized), private) {
			t.Fatalf("private collaboration data leaked: %s", private)
		}
	}
}
