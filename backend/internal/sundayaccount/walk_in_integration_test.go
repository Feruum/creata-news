//go:build integration

package sundayaccount

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGuestWalkInConcurrentScansNotifyOnce(t *testing.T) {
	h, _ := accountFixture(t)
	seedCheckIn(t, h)
	cookie, csrf, account := startGuest(t, h, "parallel-walk-in@example.test", "")
	ticket := guestTicket(t, h)
	results := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		go func() {
			results <- callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
		}()
	}
	for range 8 {
		response := <-results
		if response.Code != 200 {
			t.Errorf("concurrent guest scan: %d %s", response.Code, response.Body)
		}
	}
	var registrations, checkedIn, notifications int
	if err := h.db.NewRaw(`SELECT count(*),count(checked_in_at),(SELECT count(*) FROM sunday_open_notification_deliveries WHERE account_id=?) FROM external_registrations WHERE event_id=1 AND sunday_open_account_id=? AND status='registered'`, account.ID, account.ID).Scan(context.Background(), &registrations, &checkedIn, &notifications); err != nil || registrations != 1 || checkedIn != 1 || notifications != 1 {
		t.Fatalf("concurrent scans duplicated registration or notification: %d %d %d %v", registrations, checkedIn, notifications, err)
	}
}

func TestGuestWalkInNotificationFailureRollsBackAttendanceAndRegistration(t *testing.T) {
	for _, status := range []string{"none", "cancelled", "pending_approval", "rejected"} {
		t.Run(status, func(t *testing.T) {
			h, _ := accountFixture(t)
			seedCheckIn(t, h)
			cookie, csrf, account := startGuest(t, h, "rollback-walk-in@example.test", "")
			if status != "none" {
				if _, err := h.db.ExecContext(context.Background(), `INSERT INTO external_registrations(event_id,sunday_open_account_id,full_name,email,status) VALUES(1,?,'Walk In','rollback-walk-in@example.test',?)`, account.ID, status); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := h.db.ExecContext(context.Background(), `ALTER TABLE sunday_open_notification_deliveries ADD CONSTRAINT injected_failure CHECK (false)`); err != nil {
				t.Fatal(err)
			}
			response := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": guestTicket(t, h)}, cookie, csrf)
			if response.Code != 503 {
				t.Fatalf("storage failure not reported: %d %s", response.Code, response.Body)
			}
			var saved string
			var checked int
			if err := h.db.NewRaw(`SELECT COALESCE(max(status::text),'none'),count(checked_in_at) FROM external_registrations WHERE sunday_open_account_id=?`, account.ID).Scan(context.Background(), &saved, &checked); err != nil || saved != status || checked != 0 {
				t.Fatalf("failed scan persisted registration/attendance: %s %d %v", saved, checked, err)
			}
		})
	}
}

func TestGuestWalkInReactivationNotifiesOnceAndKeepsFirstAttendance(t *testing.T) {
	for _, status := range []string{"cancelled", "pending_approval", "rejected"} {
		t.Run(status, func(t *testing.T) {
			h, _ := accountFixture(t)
			seedCheckIn(t, h)
			cookie, csrf, account := startGuest(t, h, "reactivate-walk-in@example.test", "")
			ticket := guestTicket(t, h)
			response := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
			if response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			first := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Microsecond)
			if _, err := h.db.ExecContext(context.Background(), `UPDATE external_registrations SET status=?,checked_in_at=?,cancelled_at=NOW() WHERE sunday_open_account_id=?`, status, first, account.ID); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				response = callAccount(t, h, "POST", "check-in", map[string]string{"ticket": ticket}, cookie, csrf)
				if response.Code != 200 {
					t.Fatalf("reactivation failed: %d %s", response.Code, response.Body)
				}
			}
			var checked time.Time
			var notifications int
			if err := h.db.NewRaw(`SELECT checked_in_at,(SELECT count(*) FROM sunday_open_notification_deliveries WHERE account_id=?) FROM external_registrations WHERE sunday_open_account_id=? AND status='registered' AND cancelled_at IS NULL`, account.ID, account.ID).Scan(context.Background(), &checked, &notifications); err != nil || !checked.Equal(first) || notifications != 2 {
				t.Fatalf("reactivation changed history or duplicated/missed notification: %v %d %v", checked, notifications, err)
			}
		})
	}
}

func TestGuestWalkInRetainsContactConflictChecks(t *testing.T) {
	h, m := accountFixture(t)
	seedCheckIn(t, h)
	cookie, csrf, id := loginAccount(t, h, m, "contact-conflict@example.test")
	if _, err := h.db.ExecContext(context.Background(), `INSERT INTO external_registrations(event_id,full_name,email,phone,status) SELECT 1,'Other','other@example.test',phone,'registered' FROM sunday_open_accounts WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	response := callAccount(t, h, "POST", "check-in", map[string]string{"ticket": guestTicket(t, h)}, cookie, csrf)
	if response.Code != 409 {
		t.Fatalf("contact conflict accepted: %d %s", response.Code, response.Body)
	}
	var count int
	if err := h.db.NewRaw(`SELECT count(*) FROM external_registrations WHERE sunday_open_account_id=?`, id).Scan(context.Background(), &count); err != nil || count != 0 {
		t.Fatalf("contact conflict allocated registration: %d %v", count, err)
	}
}
