//go:build integration

package sundayopen

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func publicEventsFixture(t *testing.T) *bun.DB {
	t.Helper()
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, table := range []string{"events", "event_slugs", "organizations", "organization_slugs", "categories", "event_registrations", "external_registrations", "users", "platform_user_roles", "organization_members", "event_collaborators"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	statements := []string{
		`INSERT INTO organizations(id,title,organization_type_id) VALUES(1,'Sunday Open',1),(2,'Other club',1)`,
		`INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(1,'sunday-open',true),(2,'other-club',true),(2,'sunday-open-old',false)`,
		`INSERT INTO categories(id,name,color) VALUES(1,'Workshops','#ff5000'),(2,'Talks','#333333')`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []struct {
		id, org                   int
		title, start, end, status string
		completed                 bool
		category                  int
	}{
		{1, 1, "Current workshop", "-1 hour", "1 hour", "published", false, 1},
		{2, 1, "Future workshop", "2 hour", "3 hour", "published", false, 1},
		{3, 1, "Finished talk", "-3 hour", "-2 hour", "published", false, 2},
		{4, 1, "Completed workshop", "4 hour", "5 hour", "published", true, 1},
		{5, 1, "Draft", "2 hour", "3 hour", "draft", false, 1},
		{6, 1, "Cancelled", "2 hour", "3 hour", "cancelled", false, 1},
		{7, 1, "Archived", "2 hour", "3 hour", "archived", false, 1},
		{8, 2, "Foreign workshop", "2 hour", "3 hour", "published", false, 1},
	} {
		completed := "NULL"
		if e.completed {
			completed = "NOW()"
		}
		_, err := db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,category_id,completed_at,check_in_secret,check_in_token) VALUES(?,?,?,1,?,NOW()+?::interval,NOW()+?::interval,'{}',?,?,`+completed+`,?,?)`, e.id, e.title, `"Workshop description"`, e.org, e.start, e.end, e.status, e.category, fmt.Sprint(e.id), fmt.Sprint(e.id))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO event_slugs(event_id,slug,is_current) VALUES(?,?,true)`, e.id, fmt.Sprintf("event-%d", e.id)); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=? WHERE id=?`, e.org == 1, e.id); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestPublicEventFeedScopesFiltersAndPaginates(t *testing.T) {
	db := publicEventsFixture(t)
	s := NewService(db, nil)
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		request *pbv1.ListPublicEventsRequest
		ids     []int32
		total   int32
	}{
		{"upcoming includes ongoing", &pbv1.ListPublicEventsRequest{}, []int32{1, 2}, 2},
		{"past includes explicitly completed", &pbv1.ListPublicEventsRequest{Period: pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_PAST}, []int32{4, 3}, 2},
		{"search", &pbv1.ListPublicEventsRequest{Search: " FUTURE "}, []int32{2}, 1},
		{"description search", &pbv1.ListPublicEventsRequest{Search: "description"}, []int32{1, 2}, 2},
		{"literal wildcard search", &pbv1.ListPublicEventsRequest{Search: "%"}, nil, 0},
		{"category", &pbv1.ListPublicEventsRequest{CategoryId: int32ptr(2)}, nil, 0},
		{"second page", &pbv1.ListPublicEventsRequest{Page: int32ptr(2), Limit: int32ptr(1)}, []int32{2}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := s.ListPublicEvents(ctx, connect.NewRequest(tc.request))
			if err != nil {
				t.Fatal(err)
			}
			if response.Msg.Total != tc.total || len(response.Msg.Events) != len(tc.ids) {
				t.Fatalf("unexpected response: %v", response.Msg)
			}
			for i, id := range tc.ids {
				if response.Msg.Events[i].Id != id {
					t.Fatalf("event order/scoping mismatch: %v", response.Msg.Events)
				}
			}
		})
	}
	legacy, err := s.GetUpcomingEvents(ctx, connect.NewRequest(&pbv1.GetUpcomingEventsRequest{}))
	if err != nil || len(legacy.Msg.Events) != 2 || legacy.Msg.Events[0].Id != 1 {
		t.Fatalf("legacy feed missing ongoing event: %v %v", legacy, err)
	}
	for _, id := range []int{5, 6, 7, 8} {
		_, err := s.GetEventBySlug(ctx, connect.NewRequest(&pbv1.GetEventBySlugRequest{Slug: fmt.Sprintf("event-%d", id)}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("private/foreign event %d exposed: %v", id, err)
		}
	}
}

func TestExternalRegistrationRejectsOtherOrganizationsAndClosedEvents(t *testing.T) {
	db := publicEventsFixture(t)
	s := NewService(db, nil, strings.Repeat("s", 32))
	for _, id := range []int32{3, 4, 5, 6, 7, 8} {
		_, err := s.RegisterExternally(context.Background(), connect.NewRequest(&pbv1.RegisterExternallyRequest{EventId: id, FullName: "Visitor", Phone: "+77000000000", Email: "visitor@example.test"}))
		if err == nil {
			t.Fatalf("event %d accepted external registration", id)
		}
		if id == 8 && connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("foreign event should be hidden: %v", err)
		}
	}
}

func TestPublicSearchMatchesVisibleDescriptionsWithoutSerializerMetadata(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	for id, description := range map[int]string{
		1: `[{"type":"p","children":[{"text":"Learn "},{"text":"together","bold":true}]}]`,
		2: `{"type":"doc","content":[{"type":"paragraph","attrs":{"id":"private-marker"},"content":[{"type":"text","text":"Workshop skills"}]}]}`,
	} {
		if _, err := db.ExecContext(ctx, `UPDATE events SET description=?::jsonb WHERE id=?`, description, id); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(db, nil)
	for _, tc := range []struct {
		search string
		ids    []int32
	}{
		{"text", nil}, {"type", nil}, {"children", nil}, {"content", nil}, {"paragraph", nil}, {"private-marker", nil}, {"bold", nil},
		{"Learn together", []int32{1}}, {"skills", []int32{2}},
	} {
		response, err := s.ListPublicEvents(ctx, connect.NewRequest(&pbv1.ListPublicEventsRequest{Search: tc.search, Limit: int32ptr(1)}))
		if err != nil {
			t.Fatal(err)
		}
		if response.Msg.Total != int32(len(tc.ids)) || len(response.Msg.Events) != len(tc.ids) {
			t.Fatalf("search %q matched metadata or missed visible text: %v", tc.search, response.Msg)
		}
		if len(tc.ids) > 0 && response.Msg.Events[0].Id != tc.ids[0] {
			t.Fatalf("unexpected visible search result: %v", response.Msg)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE events SET description='"Plain strings"' WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	response, err := s.ListPublicEvents(ctx, connect.NewRequest(&pbv1.ListPublicEventsRequest{Search: "Plain strings"}))
	if err != nil || response.Msg.Total != 1 || response.Msg.Events[0].Id != 2 {
		t.Fatalf("plain description not searchable: %v %v", response, err)
	}
}

func TestPublicCategoriesIncludeEntirePublishedOrganizationCatalogue(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO categories(id,name,color) VALUES(3,'Draft only','#333333'),(4,'Foreign only','#333333'),(5,'Later page','#333333')`,
		`UPDATE events SET category_id=3 WHERE id=5`,
		`UPDATE events SET category_id=4 WHERE id=8`,
		`UPDATE events SET category_id=5 WHERE id=2`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(db, nil)
	for _, request := range []*pbv1.ListPublicEventsRequest{
		{Limit: int32ptr(1)},
		{Page: int32ptr(2), Limit: int32ptr(1)},
		{Period: pbv1.PublicEventPeriod_PUBLIC_EVENT_PERIOD_PAST},
		{Search: "no matching event"},
		{CategoryId: int32ptr(1)},
	} {
		response, err := s.ListPublicEvents(ctx, connect.NewRequest(request))
		if err != nil {
			t.Fatal(err)
		}
		categories := response.Msg.Categories
		if len(categories) != 3 || categories[0].Id != 5 || categories[0].Name != "Later page" || categories[1].Id != 2 || categories[1].Name != "Talks" || categories[2].Id != 1 || categories[2].Name != "Workshops" {
			t.Fatalf("incomplete or leaked categories for %v: %v", request, categories)
		}
	}
	// Regression: deriving categories from a maximum-size event page silently
	// omits categories once the organization has more than 200 published events.
	for _, statement := range []string{
		`INSERT INTO categories(id,name,color) SELECT n,'Extra '||n,'#333333' FROM generate_series(1000,1200) AS n`,
		`INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,category_id,check_in_secret,check_in_token) SELECT n,'Extra '||n,'"Description"',1,1,NOW()+INTERVAL '1 day',NOW()+INTERVAL '2 days','{}','published',n,'secret-'||n,'token-'||n FROM generate_series(1000,1200) AS n`,
		`INSERT INTO event_slugs(event_id,slug,is_current) SELECT n,'extra-'||n,true FROM generate_series(1000,1200) AS n`,
		`UPDATE events SET publish_on_sunday_open=true WHERE id BETWEEN 1000 AND 1200`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	response, err := s.ListPublicEvents(ctx, connect.NewRequest(&pbv1.ListPublicEventsRequest{Limit: int32ptr(1)}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Events) != 1 || len(response.Msg.Categories) != 204 {
		t.Fatalf("categories were limited by the event page: %v", response.Msg)
	}
}

func TestEventCreatorCanReadExternalVisitorsButStrangerCannot(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO users(id,username,email,password,kratos_id) VALUES(1,'creator','creator@test.local','test','creator'),(2,'stranger','stranger@test.local','test','stranger')`,
		`INSERT INTO platform_user_roles(user_id,role) VALUES(1,'none'),(2,'none')`,
		`INSERT INTO external_registrations(event_id,full_name,phone,email,status) VALUES(1,'External visitor','+77000000000','visitor@test.local','registered')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)))
	creator := context.WithValue(ctx, auth.UserIDKey, "creator")
	response, err := s.GetExternalRegistrations(creator, connect.NewRequest(&pbv1.GetExternalRegistrationsRequest{EventId: 1}))
	if err != nil || response.Msg.Total != 1 {
		t.Fatalf("creator cannot manage external visitors: %v %v", response, err)
	}
	stranger := context.WithValue(ctx, auth.UserIDKey, "stranger")
	_, err = s.GetExternalRegistrations(stranger, connect.NewRequest(&pbv1.GetExternalRegistrationsRequest{EventId: 1}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("stranger read private visitor list: %v", err)
	}
}

func int32ptr(value int32) *int32 { return &value }

func TestPublicationTargetOverridesOrganizationSlug(t *testing.T) {
	db := publicEventsFixture(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=false WHERE id IN (1,2); UPDATE events SET publish_on_sunday_open=true WHERE id=8`); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, nil, strings.Repeat("s", 32))
	response, err := s.ListPublicEvents(ctx, connect.NewRequest(&pbv1.ListPublicEventsRequest{}))
	if err != nil || len(response.Msg.Events) != 1 || response.Msg.Events[0].Id != 8 {
		t.Fatalf("organization still controls publication: %v %v", response, err)
	}
	if _, err := s.GetEventBySlug(ctx, connect.NewRequest(&pbv1.GetEventBySlugRequest{Slug: "event-8"})); err != nil {
		t.Fatal("published target hidden by organization", err)
	}
	if _, err := s.GetEventBySlug(ctx, connect.NewRequest(&pbv1.GetEventBySlugRequest{Slug: "event-1"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatal("unpublished target exposed by organization")
	}
	if _, err := s.RegisterExternally(ctx, connect.NewRequest(&pbv1.RegisterExternallyRequest{EventId: 8, FullName: "Visitor", Phone: "test", Email: "visitor@example.test"})); err != nil {
		t.Fatal("other organization Sunday target could not register", err)
	}
}
