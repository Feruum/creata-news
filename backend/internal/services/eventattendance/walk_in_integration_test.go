//go:build integration

package eventattendance

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/studyverse/ems-backend/gen/events/v1"
)

func TestWalkInConcurrentScansCreateOneRegistrationAndAttendance(t *testing.T) {
	for _, sunday := range []bool{false, true} {
		t.Run(fmt.Sprint(sunday), func(t *testing.T) {
			db, s, ctx := guardedAttendanceFixture(t)
			if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=? WHERE id=1`, sunday); err != nil {
				t.Fatal(err)
			}
			results := make(chan error, 8)
			for range 8 {
				go func() {
					response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
					if err == nil && !response.Msg.Success {
						err = fmt.Errorf("scan rejected: %s", response.Msg.Message)
					}
					results <- err
				}()
			}
			for range 8 {
				if err := <-results; err != nil {
					t.Error(err)
				}
			}
			var registrations, attendance int
			if err := db.NewRaw(`SELECT (SELECT count(*) FROM event_registrations),(SELECT count(*) FROM event_attendance)`).Scan(ctx, &registrations, &attendance); err != nil || registrations != 1 || attendance != 1 {
				t.Fatalf("duplicate walk-in rows: %d %d %v", registrations, attendance, err)
			}
			totals, err := s.GetEventAttendance(ctx, connect.NewRequest(&pb.GetEventAttendanceRequest{EventId: 1}))
			if err != nil || totals.Msg.TotalRegistered != 1 || totals.Msg.TotalAttended != 1 || totals.Msg.TotalCheckedIn != 1 {
				t.Fatalf("walk-in totals: %v %v", totals, err)
			}
		})
	}
}

func TestWalkInPreservesAttendanceHistoryAndCommitsReactivation(t *testing.T) {
	for _, status := range []string{"checked_in", "attended", "no_show"} {
		t.Run(status, func(t *testing.T) {
			db, s, ctx := guardedAttendanceFixture(t)
			first := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Microsecond)
			if _, err := db.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status,cancelled_at) VALUES(1,1,1,'cancelled',NOW()); INSERT INTO event_attendance(registration_id,status,checked_in_at,method) VALUES(1,?,?,'self')`, status, first); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
				if err != nil || !response.Msg.Success {
					t.Fatalf("repeat scan: %v %v", response, err)
				}
			}
			var savedStatus, savedRegistration string
			var checked time.Time
			if err := db.NewRaw(`SELECT ea.status,ea.checked_in_at,er.status FROM event_attendance ea JOIN event_registrations er ON er.id=ea.registration_id WHERE er.id=1 AND er.cancelled_at IS NULL`).Scan(ctx, &savedStatus, &checked, &savedRegistration); err != nil {
				t.Fatal(err)
			}
			want := "checked_in"
			if status == "attended" {
				want = status
			}
			if savedStatus != want || savedRegistration != "registered" || !checked.Equal(first) {
				t.Fatalf("attendance history changed or reactivation rolled back: %s %s %v", savedStatus, savedRegistration, checked)
			}
		})
	}
}

func TestWalkInAttendanceFailureRollsBackRegistration(t *testing.T) {
	for _, status := range []string{"none", "cancelled", "pending_approval", "rejected", "waitlist"} {
		t.Run(status, func(t *testing.T) {
			db, s, ctx := guardedAttendanceFixture(t)
			if status != "none" {
				if _, err := db.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,?)`, status); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.ExecContext(ctx, `ALTER TABLE event_attendance ADD CONSTRAINT injected_failure CHECK (false)`); err != nil {
				t.Fatal(err)
			}
			response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
			if connect.CodeOf(err) != connect.CodeInternal {
				t.Fatalf("attendance failure not reported: %v %v", response, err)
			}
			var saved string
			if err := db.NewRaw(`SELECT COALESCE(max(status::text),'none') FROM event_registrations`).Scan(ctx, &saved); err != nil || saved != status {
				t.Fatalf("failed scan changed registration: %s %v", saved, err)
			}
		})
	}
}

func TestWalkInInvalidQRDoesNotCreateRegistration(t *testing.T) {
	for _, mutation := range []string{"check_in_token='revoked'", "check_in_secret='revoked'", "status='cancelled'", "completed_at=NOW()", "start_time=NOW()+interval '1 day',end_time=NOW()+interval '2 days'", "start_time=NOW()-interval '5 hours',end_time=NOW()-interval '4 hours'"} {
		t.Run(mutation, func(t *testing.T) {
			db, s, ctx := guardedAttendanceFixture(t)
			if _, err := db.ExecContext(ctx, "UPDATE events SET "+mutation+" WHERE id=1"); err != nil {
				t.Fatal(err)
			}
			response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
			if err == nil && response.Msg.Success {
				t.Fatal("invalid QR accepted")
			}
			var count int
			if err := db.NewRaw(`SELECT count(*) FROM event_registrations`).Scan(ctx, &count); err != nil || count != 0 {
				t.Fatalf("invalid QR created registration: %d %v", count, err)
			}
		})
	}
}

func TestWalkInRepeatedScanReportsAlreadyCheckedIn(t *testing.T) {
	_, s, ctx := guardedAttendanceFixture(t)
	for i := range 2 {
		response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
		if err != nil || !response.Msg.Success || (i == 1 && !strings.Contains(response.Msg.Message, "already")) {
			t.Fatalf("repeat scan response: %v %v", response, err)
		}
	}
}
