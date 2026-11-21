//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"testing"
)

func TestOutOfOrderSearchEventsUseCurrentCommittedVisibility(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
	CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
	INSERT INTO organizations (id,title,organization_type_id) VALUES (1,'org',1);
	INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
	VALUES (1,'current title','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','a','a');`); err != nil {
		t.Fatal(err)
	}
	client := &fakeSearchClient{}
	deps := HandlerDeps{DB: db, SearchClient: client}
	payload, _ := json.Marshal(events.EventUpdatedPayload{EventID: 1, Title: "outdated title", Status: "published"})
	msg := message.NewMessage("late-old-update", payload)
	msg.SetContext(ctx)
	if err := handleEventUpdated(deps)(msg); err != nil {
		t.Fatal(err)
	}
	if client.indexedEvt == nil || client.indexedEvt.Title != "current title" {
		t.Fatalf("stale title indexed: %+v", client)
	}
	client.indexedEvt = nil
	if _, err := db.ExecContext(ctx, "UPDATE events SET status='draft' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if err := handleEventUpdated(deps)(msg); err != nil {
		t.Fatal(err)
	}
	if client.indexedEvt != nil || client.deletedIdx != search.IndexEvents || client.deletedID != 1 {
		t.Fatalf("old message resurrected private event: %+v", client)
	}
	if _, err := db.ExecContext(ctx, "UPDATE events SET status='published' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	published, _ := json.Marshal(events.EventPublishedPayload{EventID: 1})
	pubMsg := message.NewMessage("published", published)
	pubMsg.SetContext(ctx)
	if err := handleEventPublishedSearch(deps)(pubMsg); err != nil {
		t.Fatal(err)
	}
	if client.indexedEvt == nil || client.indexedEvt.ID != 1 {
		t.Fatal("published event missing from index")
	}
}
