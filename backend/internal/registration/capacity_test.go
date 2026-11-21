//go:build integration

package registration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func TestConcurrentInternalAndExternalRegistrationsShareCapacity(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, table := range []string{"events", "event_registrations", "external_registrations"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,capacity,check_in_secret,check_in_token)
		VALUES (1,'capacity','{}',1,1,NOW()+interval '1 hour',NOW()+interval '2 hours','{}','published',2,'secret','token');
		INSERT INTO external_registrations (event_id,full_name,phone,email,status) VALUES (1,'existing','existing','existing@test.local','registered');`); err != nil {
		t.Fatal(err)
	}
	var succeeded atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				event, err := LockOpenEvent(ctx, tx, 1, time.Now)
				if err != nil {
					return err
				}
				if err := EnsureCapacity(ctx, tx, event); err != nil {
					return err
				}
				if i%2 == 0 {
					_, err = tx.ExecContext(ctx, "INSERT INTO event_registrations (event_id,user_id,status) VALUES (1,?,'registered')", i+1)
				} else {
					_, err = tx.ExecContext(ctx, "INSERT INTO external_registrations (event_id,full_name,phone,email,status) VALUES (1,'new',?,'new@test.local','registered')", fmt.Sprint(i))
				}
				return err
			})
			if err == nil {
				succeeded.Add(1)
			} else if connect.CodeOf(err) != connect.CodeResourceExhausted {
				t.Errorf("unexpected registration failure: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if succeeded.Load() != 1 {
		t.Fatalf("capacity exceeded: %d new registrations, want1", succeeded.Load())
	}
	if _, err := db.ExecContext(ctx, "UPDATE events SET end_time=NOW()-interval '1 hour' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error { _, err := LockOpenEvent(ctx, tx, 1, time.Now); return err })
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("ended event accepted: %v", err)
	}
}

func TestRegistrationChecksDeadlineAfterWaitingForEventLock(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
	INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token)
	VALUES (1,'closing','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','a','a');`); err != nil {
		t.Fatal(err)
	}
	lock, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback()
	var id int32
	if err := lock.NewRaw("SELECT id FROM events WHERE id=1 FOR UPDATE").Scan(ctx, &id); err != nil {
		t.Fatal(err)
	}
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	called := make(chan struct{}, 1)
	result := make(chan error, 1)
	go func() {
		result <- db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			_, err := LockOpenEvent(ctx, tx, 1, func() time.Time { called <- struct{}{}; return time.Unix(0, clock.Load()) })
			return err
		})
	}()
	select {
	case <-called:
		t.Fatal("deadline sampled before event lock acquired")
	case <-time.After(50 * time.Millisecond):
	}
	clock.Store(time.Now().Add(2 * time.Hour).UnixNano())
	if err := lock.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("accepted registration after deadline: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("waiting registration did not finish")
	}
}
