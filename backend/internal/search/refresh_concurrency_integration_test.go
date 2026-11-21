//go:build integration

package search

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/registration"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

// Only the external indexing task is delayed; SQL and domain locks are real.
type delayedEventIndex struct {
	mu           sync.Mutex
	documents    map[int32]EventDocument
	calls        int
	firstStarted chan struct{}
	laterStarted chan struct{}
	releaseFirst chan struct{}
	releaseOnce  sync.Once
}

func newDelayedEventIndex() *delayedEventIndex {
	return &delayedEventIndex{
		documents: make(map[int32]EventDocument), firstStarted: make(chan struct{}),
		laterStarted: make(chan struct{}, 1), releaseFirst: make(chan struct{}),
	}
}

func (c *delayedEventIndex) release() {
	c.releaseOnce.Do(func() { close(c.releaseFirst) })
}

func (c *delayedEventIndex) write(ctx context.Context, documents []EventDocument, deleted []int32) error {
	c.mu.Lock()
	c.calls++
	call := c.calls
	c.mu.Unlock()
	if call == 1 {
		close(c.firstStarted)
		select {
		case <-c.releaseFirst:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case c.laterStarted <- struct{}{}:
		default:
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, document := range documents {
		c.documents[document.ID] = document
	}
	for _, id := range deleted {
		delete(c.documents, id)
	}
	return nil
}

func (c *delayedEventIndex) IndexEvent(ctx context.Context, document *EventDocument) error {
	return c.write(ctx, []EventDocument{*document}, nil)
}

func (c *delayedEventIndex) IndexEvents(ctx context.Context, documents []EventDocument) error {
	return c.write(ctx, documents, nil)
}

func (c *delayedEventIndex) DeleteDocument(ctx context.Context, _ string, id int32) error {
	return c.write(ctx, nil, []int32{id})
}

func (c *delayedEventIndex) DeleteDocuments(ctx context.Context, _ string, ids []int32) error {
	return c.write(ctx, nil, ids)
}

func eventRefreshDB(t *testing.T) *bun.DB {
	t.Helper()
	db := testsupport.IsolatedDB(t)
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE events (LIKE public.events INCLUDING ALL);
		CREATE TABLE organizations (LIKE public.organizations INCLUDING ALL);
		CREATE TABLE event_registrations (LIKE public.event_registrations INCLUDING ALL);
		INSERT INTO organizations(id,title,organization_type_id) VALUES(1,'current org',1);
		INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
		VALUES(1,'old title','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','refresh1','refresh1'),
		      (2,'second title','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','refresh2','refresh2');`); err != nil {
		t.Fatal(err)
	}
	return db
}

func runEventRefresh(ctx context.Context, db *bun.DB, client *delayedEventIndex, batch bool, ids []int32) error {
	if batch {
		_, err := RefreshEvents(ctx, db, client, ids)
		return err
	}
	_, err := RefreshEvent(ctx, db, client, ids[0])
	return err
}

func waitEventRefresh(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("refresh failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("refresh did not finish")
	}
}

func startDelayedEventRefresh(t *testing.T, db *bun.DB, client *delayedEventIndex, batch bool, ids []int32) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	done := make(chan error, 1)
	go func() { done <- runEventRefresh(ctx, db, client, batch, ids) }()
	t.Cleanup(func() {
		client.release()
		waitEventRefresh(t, done)
		cancel()
	})
	select {
	case <-client.firstStarted:
	case <-ctx.Done():
		t.Fatal("first indexing task did not start")
	}
	return ctx
}

func waitCompetingEventRefresh(t *testing.T, ctx context.Context, db *bun.DB, client *delayedEventIndex) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-client.laterStarted:
			t.Fatal("new refresh reached search before the old task finished")
		case <-ticker.C:
			var waiting bool
			if err := db.NewRaw(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database() AND wait_event_type='Lock'
				AND query LIKE '%pg_advisory_xact_lock%')`).Scan(ctx, &waiting); err != nil {
				t.Fatal(err)
			}
			if waiting {
				return
			}
		case <-deadline.C:
			t.Fatal("competing refresh did not wait on the search lock")
		}
	}
}

func TestDelayedEventRefreshDoesNotBlockDomainMutations(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, mutation := range []string{"registration", "edit"} {
			t.Run(fmt.Sprintf("batch=%t/%s", batch, mutation), func(t *testing.T) {
				db := eventRefreshDB(t)
				client := newDelayedEventIndex()
				startDelayedEventRefresh(t, db, client, batch, []int32{1})
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if mutation == "registration" {
					err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
						if _, err := registration.LockOpenEvent(ctx, tx, 1, time.Now); err != nil {
							return err
						}
						_, err := tx.ExecContext(ctx, "INSERT INTO event_registrations(event_id,user_id) VALUES(1,1)")
						return err
					})
					if err != nil {
						t.Fatalf("registration was blocked by the unfinished indexing task: %v", err)
					}
					var count int
					if err := db.NewRaw("SELECT count(*) FROM event_registrations WHERE event_id=1").Scan(ctx, &count); err != nil || count != 1 {
						t.Fatalf("registration did not commit: count=%d error=%v", count, err)
					}
				} else if _, err := db.ExecContext(ctx, "UPDATE events SET title='edited title' WHERE id=1"); err != nil {
					t.Fatalf("edit was blocked by the unfinished indexing task: %v", err)
				}
			})
		}
	}
}

func TestCompetingEventRefreshesKeepCurrentCommittedDocument(t *testing.T) {
	for _, firstBatch := range []bool{false, true} {
		for _, secondBatch := range []bool{false, true} {
			t.Run(fmt.Sprintf("firstBatch=%t/secondBatch=%t", firstBatch, secondBatch), func(t *testing.T) {
				db := eventRefreshDB(t)
				client := newDelayedEventIndex()
				ctx := startDelayedEventRefresh(t, db, client, firstBatch, []int32{1, 2, 1})
				editCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				if _, err := db.ExecContext(editCtx, "UPDATE events SET title='new committed title' WHERE id=1"); err != nil {
					t.Fatalf("edit could not commit during old search task: %v", err)
				}
				done := make(chan error, 1)
				ids := []int32{1}
				if secondBatch {
					ids = []int32{2, 1, 2}
				}
				go func() { done <- runEventRefresh(ctx, db, client, secondBatch, ids) }()
				finished := false
				t.Cleanup(func() {
					client.release()
					if !finished {
						waitEventRefresh(t, done)
					}
				})
				waitCompetingEventRefresh(t, ctx, db, client)
				client.release()
				waitEventRefresh(t, done)
				finished = true
				client.mu.Lock()
				defer client.mu.Unlock()
				if document := client.documents[1]; document.Title != "new committed title" || document.OrganizationTitle != "current org" {
					t.Fatalf("old refresh overwrote current data: %+v", document)
				}
			})
		}
	}
}

func TestCompetingEventRefreshRemovesPrivateAndDeletedDocuments(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, mutation := range []string{"draft", "archived", "deleted"} {
			t.Run(fmt.Sprintf("batch=%t/%s", batch, mutation), func(t *testing.T) {
				db := eventRefreshDB(t)
				client := newDelayedEventIndex()
				ctx := startDelayedEventRefresh(t, db, client, !batch, []int32{1})
				editCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				query := "UPDATE events SET status=? WHERE id=1"
				args := []any{mutation}
				if mutation == "deleted" {
					query, args = "DELETE FROM events WHERE id=1", nil
				}
				if _, err := db.ExecContext(editCtx, query, args...); err != nil {
					t.Fatalf("visibility change could not commit during old search task: %v", err)
				}
				done := make(chan error, 1)
				go func() { done <- runEventRefresh(ctx, db, client, batch, []int32{1}) }()
				finished := false
				t.Cleanup(func() {
					client.release()
					if !finished {
						waitEventRefresh(t, done)
					}
				})
				waitCompetingEventRefresh(t, ctx, db, client)
				client.release()
				waitEventRefresh(t, done)
				finished = true
				client.mu.Lock()
				defer client.mu.Unlock()
				if document, exists := client.documents[1]; exists {
					t.Fatalf("old refresh resurrected %s event: %+v", mutation, document)
				}
			})
		}
	}
}
