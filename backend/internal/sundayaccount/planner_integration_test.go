//go:build integration

package sundayaccount

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPlannerRescheduleLocationAndReminderIdempotency(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, id := loginAccount(t, h, m, "planner@example.test")
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Workshop','{}',1,1,NOW()+INTERVAL '24 hours'-INTERVAL '30 seconds',NOW()+INTERVAL '25 hours','{"type":"external","external_address":"First address"}','published',true,'a','b')`); err != nil {
		t.Fatal(err)
	}
	w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if _, err := h.db.ExecContext(ctx, `UPDATE external_registrations SET registered_at=NOW()-INTERVAL '2 hours' WHERE sunday_open_account_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := h.planNotifications(ctx); err != nil {
		t.Fatal(err)
	}
	if err := h.planNotifications(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_notification_deliveries WHERE kind='reminder_24h'`).Scan(ctx, &count); err != nil || count != 1 {
		t.Fatalf("reminder dedupe %d %v", count, err)
	}
	if _, err := h.db.ExecContext(ctx, `UPDATE events SET structured_location='{"type":"external","external_address":"Second address"}',updated_at=NOW() WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	// Running without a configured bot still refreshes account snapshots.
	h.tick(ctx)
	var payload string
	if err := h.db.NewRaw(`SELECT snapshot::text FROM external_registrations WHERE sunday_open_account_id=?`, id).Scan(ctx, &payload); err != nil {
		t.Fatal(err)
	}
	var snapshot eventSnapshot
	_ = json.Unmarshal([]byte(payload), &snapshot)
	if snapshot.LocationText != "Second address" {
		t.Fatalf("location not refreshed %s", payload)
	}
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_notification_deliveries WHERE kind='event_changed'`).Scan(ctx, &count); err != nil || count != 1 {
		t.Fatalf("location notification %d %v", count, err)
	}
	// Forward reschedule invalidates the previous queued reminder at send time.
	if _, err := h.db.ExecContext(ctx, `UPDATE events SET start_time=start_time+INTERVAL '1 hour',end_time=end_time+INTERVAL '1 hour',updated_at=NOW() WHERE id=1; UPDATE sunday_open_accounts SET telegram_chat_id=111,telegram_user_id=111 WHERE id=?; UPDATE sunday_open_notification_deliveries SET sent_at=NOW() WHERE kind<>'reminder_24h'`, id); err != nil {
		t.Fatal(err)
	}
	var received []map[string]any
	fakeTelegram(h, `{"ok":true}`, &received)
	if !h.deliverOne(ctx) {
		t.Fatal("no reminder claimed")
	}
	if len(received) != 0 {
		t.Fatal("stale reminder sent after reschedule")
	}
}

func TestPlannerVisitsRowsBeyondFirstThousand(t *testing.T) {
	h, m := accountFixture(t)
	_, _, id := loginAccount(t, h, m, "bulk@example.test")
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) SELECT n,'Event '||n,'{}',1,1,NOW()+INTERVAL '2 days',NOW()+INTERVAL '3 days','{"type":"external","external_address":"First"}','published',true,'s'||n,'t'||n FROM generate_series(1,1001) n; INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,status,snapshot) SELECT e.id,?,'Visitor','bulk@example.test','registered',`+snapshotSQL+` FROM events e; UPDATE events SET structured_location='{"type":"external","external_address":"Next"}',updated_at=NOW()`, id); err != nil {
		t.Fatal(err)
	}
	if err := h.planNotifications(ctx); err != nil {
		t.Fatal(err)
	}
	if err := h.planNotifications(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM sunday_open_notification_deliveries WHERE kind='event_changed'`).Scan(ctx, &count); err != nil || count != 1001 {
		t.Fatalf("later registrations starved %d %v", count, err)
	}
}

func TestExpiredReminderAndRestoredRegistrationJobsAreDiscarded(t *testing.T) {
	h, m := accountFixture(t)
	cookie, csrf, id := loginAccount(t, h, m, "oldreminder@example.test")
	ctx := context.Background()
	if _, err := h.db.ExecContext(ctx, `INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,publish_on_sunday_open,check_in_secret,check_in_token) VALUES(1,'Event','{}',1,1,NOW()+INTERVAL '22 hours',NOW()+INTERVAL '23 hours','{}','published',true,'s','t')`); err != nil {
		t.Fatal(err)
	}
	w := callAccount(t, h, "POST", "registrations", map[string]int{"eventId": 1}, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if _, err := h.db.ExecContext(ctx, `UPDATE external_registrations SET registered_at=NOW()-INTERVAL '3 days' WHERE sunday_open_account_id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := h.planNotifications(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	_ = h.db.NewRaw(`SELECT count(*) FROM sunday_open_notification_deliveries WHERE kind='reminder_24h'`).Scan(ctx, &count)
	if count != 0 {
		t.Fatal("enqueued late 24h reminder")
	}
	// A queue item from an earlier incarnation cannot notify a restored booking.
	if _, err := h.db.ExecContext(ctx, `UPDATE events SET start_time=NOW()+INTERVAL '1 hour'-INTERVAL '20 seconds',end_time=NOW()+INTERVAL '2 hours'; UPDATE external_registrations SET snapshot=`+snapshotSQL+` FROM events e WHERE e.id=external_registrations.event_id; INSERT INTO sunday_open_notification_deliveries(delivery_key,account_id,registration_id,event_id,kind,payload,created_at) SELECT 'old',sunday_open_account_id,id,event_id,'reminder_1h',snapshot,NOW()-INTERVAL '1 day' FROM external_registrations; UPDATE external_registrations SET registered_at=NOW(); UPDATE sunday_open_accounts SET telegram_chat_id=111,telegram_user_id=111 WHERE id=?; UPDATE sunday_open_notification_deliveries SET sent_at=NOW() WHERE kind<>'reminder_1h'`, id); err != nil {
		t.Fatal(err)
	}
	var received []map[string]any
	fakeTelegram(h, `{"ok":true}`, &received)
	h.deliverOne(ctx)
	if len(received) > 0 {
		t.Fatal("old booking reminder delivered after restore")
	}
}
