//go:build integration

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestFinalizationFailureRollsBackAndRetriesAttendance(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, table := range []string{"events", "event_organizers", "event_registrations", "event_attendance"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_grace_minutes,check_in_secret,check_in_token)
		VALUES (1,'expired','{}',1,1,NOW()-interval '5 hours',NOW()-interval '4 hours','{}','published',0,'secret','token');
		INSERT INTO event_organizers (event_id,user_id,organization_id,added_at) VALUES (1,1,1,NOW()),(1,2,1,NOW());
		INSERT INTO event_registrations (id,event_id,user_id,status,registered_at) VALUES (1,1,1,'registered',NOW()),(2,1,2,'registered',NOW());
		INSERT INTO event_attendance (registration_id,status,method) VALUES (2,'no_show','manual');
		ALTER TABLE event_attendance ADD CONSTRAINT injected_failure CHECK (status <> 'attended');
	`); err != nil {
		t.Fatal(err)
	}
	w := NewFinalizationWorker(db)
	now := time.Now()
	count, err := w.finalizeEvents(ctx, now)
	if err == nil || count != 0 {
		t.Fatalf("expected retryable failure and no completed events, got %d %v", count, err)
	}
	var completed bool
	if err := db.NewRaw("SELECT completed_at IS NOT NULL FROM events WHERE id=1").Scan(ctx, &completed); err != nil {
		t.Fatal(err)
	}
	if completed {
		t.Fatal("failed auto-attendance permanently marked event completed")
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE event_attendance DROP CONSTRAINT injected_failure"); err != nil {
		t.Fatal(err)
	}
	count, err = w.finalizeEvents(ctx, now)
	if err != nil || count != 1 {
		t.Fatalf("retry failed: %d %v", count, err)
	}
	var attendanceCount int
	if err := db.NewRaw("SELECT count(*) FROM event_attendance WHERE registration_id=1 AND status='attended' AND method='auto'").Scan(ctx, &attendanceCount); err != nil || attendanceCount != 1 {
		t.Fatalf("auto attendance: %d %v", attendanceCount, err)
	}
	if err := db.NewRaw("SELECT count(*) FROM event_attendance WHERE registration_id=2 AND status='no_show' AND method='manual'").Scan(ctx, &attendanceCount); err != nil || attendanceCount != 1 {
		t.Fatalf("existing attendance overwritten: %d %v", attendanceCount, err)
	}
	count, err = w.finalizeEvents(ctx, now)
	if err != nil || count != 0 {
		t.Fatalf("repeat finalization not idempotent: %d %v", count, err)
	}
}

func TestArchivingAndSearchCleanupCommitTogether(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE events (LIKE public.events INCLUDING ALL);
	CREATE TABLE domain_event_outbox (LIKE public.domain_event_outbox INCLUDING ALL);
	INSERT INTO events (id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,completed_at,check_in_secret,check_in_token)
	VALUES (1,'old','{}',1,1,NOW()-interval '1 year',NOW()-interval '1 year','{}','published',NOW()-interval '1 year','a','a');
	ALTER TABLE domain_event_outbox ADD CONSTRAINT injected_failure CHECK (topic <> 'event.archived');`); err != nil {
		t.Fatal(err)
	}
	w := NewFinalizationWorker(db)
	count, err := w.archiveOldEvents(ctx, time.Now())
	if err == nil || count != 0 {
		t.Fatalf("expected failed atomic archive: %d %v", count, err)
	}
	var status string
	if err := db.NewRaw("SELECT status FROM events WHERE id=1").Scan(ctx, &status); err != nil || status != "published" {
		t.Fatalf("archive survived failed cleanup enqueue: %s %v", status, err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE domain_event_outbox DROP CONSTRAINT injected_failure"); err != nil {
		t.Fatal(err)
	}
	count, err = w.archiveOldEvents(ctx, time.Now())
	if err != nil || count != 1 {
		t.Fatalf("retry: %d %v", count, err)
	}
	var jobs int
	if err := db.NewRaw("SELECT count(*) FROM domain_event_outbox WHERE topic='event.archived' AND payload->>'event_id'='1'").Scan(ctx, &jobs); err != nil || jobs != 1 {
		t.Fatalf("search cleanup outbox missing: %d %v", jobs, err)
	}
	count, err = w.archiveOldEvents(ctx, time.Now())
	if err != nil || count != 0 {
		t.Fatalf("archive repeat: %d %v", count, err)
	}
}
