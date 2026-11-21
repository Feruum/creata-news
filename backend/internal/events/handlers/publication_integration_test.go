//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestMapAnnouncementChecksCurrentPublicationTarget(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events(id integer PRIMARY KEY,status text,publish_on_map boolean);
	CREATE TABLE users(id integer,telegram_id bigint);
	CREATE TABLE organization_subscriptions(organization_id integer,user_id integer);
	INSERT INTO events VALUES(1,'published',false),(2,'draft',true),(3,'published',true);
	INSERT INTO users VALUES(1,12345);
	INSERT INTO organization_subscriptions VALUES(7,1);`); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"sent":1,"failed":0}`)
	}))
	defer server.Close()
	handler := handleEventPublished(HandlerDeps{DB: db, TelegramBotURL: server.URL, InternalNotificationSecret: "test-secret"})
	for _, id := range []int32{1, 2, 99, 3} {
		payload, _ := json.Marshal(events.EventPublishedPayload{EventID: id, OrganizationID: 7, Title: "Stale published payload"})
		if err := handler(message.NewMessage(fmt.Sprint(id), payload)); err != nil {
			t.Fatal(err)
		}
		want := 0
		if id == 3 {
			want = 1
		}
		if calls != want {
			t.Fatalf("event %d sent %d announcements, want %d", id, calls, want)
		}
	}
}
