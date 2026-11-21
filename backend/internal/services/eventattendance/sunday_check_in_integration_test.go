//go:build integration

package eventattendance

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestSundayStudentScanRegistersAndChecksInOnce(t *testing.T) {
	db, service, ctx, event, _ := attendanceCountsFixture(t)
	if _, err := db.ExecContext(ctx, "UPDATE events SET publish_on_sunday_open=true,capacity=1 WHERE id=?", event); err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: "counts-token", RotatingToken: computeRotatingToken("counts-secret", getCurrentWindow())})
	for i := 0; i < 2; i++ {
		response, err := service.SelfCheckIn(ctx, request)
		if err != nil || !response.Msg.Success {
			t.Fatalf("student check-in: %v %v", response, err)
		}
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM event_attendance ea JOIN event_registrations er ON er.id=ea.registration_id WHERE er.event_id=? AND er.status='registered' AND ea.status='checked_in'", event).Scan(context.Background(), &count); err != nil || count != 1 {
		t.Fatalf("student attendance %d %v", count, err)
	}
	var checked time.Time
	_ = db.NewRaw("SELECT checked_in_at FROM event_attendance LIMIT 1").Scan(ctx, &checked)
	response, err := service.SelfCheckIn(ctx, request)
	if err != nil || !response.Msg.Success {
		t.Fatal("repeat failed")
	}
	var after time.Time
	_ = db.NewRaw("SELECT checked_in_at FROM event_attendance LIMIT 1").Scan(ctx, &after)
	if !after.Equal(checked) {
		t.Fatal("repeat changed original check-in time")
	}
}

func TestNonSundayEventRegistersWalkIn(t *testing.T) {
	_, service, ctx, _, _ := attendanceCountsFixture(t)
	response, err := service.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: "counts-token", RotatingToken: computeRotatingToken("counts-secret", getCurrentWindow())}))
	if err != nil || !response.Msg.Success {
		t.Fatalf("ordinary walk-in failed: %v %v", response, err)
	}
}

func TestSundayStudentCheckInRechecksTokenRevocationAfterLock(t *testing.T) {
	db, service, ctx, eventID, _ := attendanceCountsFixture(t)
	if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=true WHERE id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	var event models.Event
	if err := db.NewSelect().Model(&event).Where("id = ?", eventID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	userID, err := service.authz.GetLocalUserID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The scan loaded the event before an organizer revoked the QR base token.
	if _, err := db.ExecContext(ctx, `UPDATE events SET check_in_token='revoked-token' WHERE id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	_, err = service.selfCheckInEvent(ctx, &event, userID, computeRotatingToken("counts-secret", getCurrentWindow()))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("revoked base token accepted after event lock: %v", err)
	}
	var count int
	if err := db.NewRaw(`SELECT count(*) FROM event_registrations WHERE event_id=?`, eventID).Scan(ctx, &count); err != nil || count != 0 {
		t.Fatalf("revoked QR allocated a seat: %d %v", count, err)
	}
}

func TestSundayStudentCheckInAllowsWalkInWhenExternalRegistrationsFillCapacity(t *testing.T) {
	db, service, ctx, eventID, _ := attendanceCountsFixture(t)
	if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=true,capacity=1 WHERE id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO external_registrations(event_id,full_name,email,phone,status) VALUES(?,'External visitor','external@example.test','+77001111111','registered')`, eventID); err != nil {
		t.Fatal(err)
	}
	response, err := service.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: "counts-token", RotatingToken: computeRotatingToken("counts-secret", getCurrentWindow())}))
	if err != nil || !response.Msg.Success {
		t.Fatalf("full event blocked QR attendance: %v %v", response, err)
	}
}

func TestSundayStudentCheckInPreservesFirstTimestampAfterAttendanceCorrection(t *testing.T) {
	db, service, ctx, eventID, _ := attendanceCountsFixture(t)
	if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=true WHERE id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: "counts-token", RotatingToken: computeRotatingToken("counts-secret", getCurrentWindow())})
	if _, err := service.SelfCheckIn(ctx, request); err != nil {
		t.Fatal(err)
	}
	first := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Microsecond)
	if _, err := db.ExecContext(ctx, `UPDATE event_attendance SET status='no_show',checked_in_at=? WHERE registration_id IN(SELECT id FROM event_registrations WHERE event_id=?)`, first, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SelfCheckIn(ctx, request); err != nil {
		t.Fatal(err)
	}
	var checked time.Time
	if err := db.NewRaw(`SELECT checked_in_at FROM event_attendance WHERE registration_id IN(SELECT id FROM event_registrations WHERE event_id=?)`, eventID).Scan(ctx, &checked); err != nil || !checked.Equal(first) {
		t.Fatalf("repeat scan replaced the first attendance timestamp: %v %v", checked, err)
	}
}
