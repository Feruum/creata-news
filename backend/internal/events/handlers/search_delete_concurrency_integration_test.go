//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

type delayedDeletedEventIndex struct {
	fakeSearchClient
	mu            sync.Mutex
	document      *search.EventDocument
	indexStarted  chan struct{}
	deleteStarted chan struct{}
	releaseIndex  chan struct{}
	releaseOnce   sync.Once
}

func (c *delayedDeletedEventIndex) release() {
	c.releaseOnce.Do(func() { close(c.releaseIndex) })
}

func (c *delayedDeletedEventIndex) IndexEvent(ctx context.Context, document *search.EventDocument) error {
	close(c.indexStarted)
	select {
	case <-c.releaseIndex:
	case <-ctx.Done():
		return ctx.Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.document = document
	return nil
}

func (c *delayedDeletedEventIndex) DeleteDocument(_ context.Context, _ string, _ int32) error {
	close(c.deleteStarted)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.document = nil
	return nil
}

func TestDeletedEventHandlerCannotBeOverwrittenByInFlightRefresh(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE events (LIKE public.events INCLUDING ALL);
		CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
		INSERT INTO organizations(id,title,organization_type_id) VALUES(1,'org',1);
		INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
		VALUES(1,'deleted event','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','deleted1','deleted1');`); err != nil {
		t.Fatal(err)
	}
	client := &delayedDeletedEventIndex{
		indexStarted: make(chan struct{}), deleteStarted: make(chan struct{}), releaseIndex: make(chan struct{}),
	}
	deps := HandlerDeps{DB: db, SearchClient: client}
	makeMessage := func(payload any) *message.Message {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		msg := message.NewMessage("search-concurrent-delete", body)
		msg.SetContext(ctx)
		return msg
	}
	updated := makeMessage(events.EventUpdatedPayload{EventID: 1, Status: "published"})
	deleted := makeMessage(events.EventDeletedPayload{EventID: 1})
	oldDone := make(chan error, 1)
	go func() { oldDone <- handleEventUpdated(deps)(updated) }()
	oldFinished := false
	wait := func(done <-chan error) {
		t.Helper()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("search handler failed: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("search handler did not finish")
		}
	}
	t.Cleanup(func() {
		client.release()
		if !oldFinished {
			wait(oldDone)
		}
	})
	select {
	case <-client.indexStarted:
	case <-ctx.Done():
		t.Fatal("index task did not start")
	}
	deleteCtx, stopDelete := context.WithTimeout(ctx, 2*time.Second)
	defer stopDelete()
	if _, err := db.ExecContext(deleteCtx, "DELETE FROM events WHERE id=1"); err != nil {
		t.Fatalf("domain deletion was blocked by search task: %v", err)
	}
	newDone := make(chan error, 1)
	go func() { newDone <- handleEventDeleted(deps)(deleted) }()
	newFinished := false
	t.Cleanup(func() {
		client.release()
		if !newFinished {
			wait(newDone)
		}
	})
	// Observe either a real PostgreSQL search-lock wait or the competing write.
	// A direct delete completes first and is then overwritten by the old index.
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
waitForDelete:
	for {
		select {
		case <-client.deleteStarted:
			wait(newDone)
			newFinished = true
			break waitForDelete
		case <-ticker.C:
			var waiting bool
			if err := db.NewRaw(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database() AND wait_event_type='Lock'
				AND query LIKE '%pg_advisory_xact_lock%')`).Scan(ctx, &waiting); err != nil {
				t.Fatal(err)
			}
			if waiting {
				break waitForDelete
			}
		case <-deadline.C:
			t.Fatal("delete handler neither wrote nor waited for the current refresh")
		}
	}
	client.release()
	wait(oldDone)
	oldFinished = true
	if !newFinished {
		wait(newDone)
		newFinished = true
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.document != nil {
		t.Fatalf("in-flight index task resurrected the deleted event: %+v", client.document)
	}
}
