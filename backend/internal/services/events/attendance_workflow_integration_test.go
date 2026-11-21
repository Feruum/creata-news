//go:build integration

package events

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/services"
	"github.com/studyverse/ems-backend/internal/services/eventattendance"
	"github.com/studyverse/ems-backend/internal/services/statistics"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestSelfCheckInUpdatesAttendanceTotalsImmediately(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "attendance-count-user")
	org := seedOrganization(t, db, "Attendance totals workflow")
	event := seedDraftEvent(t, db, actor, org.ID, "Live attendance totals", "Hall")
	ctx := authedContext("attendance-count-user")
	if _, err := db.NewUpdate().Model(&event).Set("status = ?", models.EventStatusPublished).
		Set("start_time = ?", time.Now().Add(-time.Minute)).Set("end_time = ?", time.Now().Add(time.Hour)).WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	registrations := services.NewEventRegistrationsService(db, nil, enforcer)
	events := NewService(db, nil, enforcer, nil, nil, "", "")
	attendance := eventattendance.NewService(db, enforcer, "")
	analytics := statistics.NewService(db, enforcer)
	assertCounts := func(t *testing.T, registered, attendees, checkedIn, noShow int32) {
		t.Helper()
		detail, err := events.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: event.ID}))
		if err != nil {
			t.Fatal(err)
		}
		if detail.Msg.Event.TotalRegistrations != registered || detail.Msg.Event.TotalAttendees != attendees {
			t.Errorf("detail counts: want registered=%d attendees=%d, got %v", registered, attendees, detail.Msg.Event)
		}
		public, err := events.ListEvents(ctx, connect.NewRequest(&eventsv1.ListEventsRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if len(public.Msg.Events) != 1 || public.Msg.Events[0].TotalRegistrations != registered || public.Msg.Events[0].TotalAttendees != attendees {
			t.Errorf("public list counts: want registered=%d attendees=%d, got %v", registered, attendees, public.Msg.Events)
		}
		admin, err := events.ListEventsForAdmin(ctx, connect.NewRequest(&eventsv1.ListEventsForAdminRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if len(admin.Msg.Events) != 1 || admin.Msg.Events[0].TotalRegistrations != registered || admin.Msg.Events[0].TotalAttendees != attendees {
			t.Errorf("admin list counts: want registered=%d attendees=%d, got %v", registered, attendees, admin.Msg.Events)
		}
		rows, err := attendance.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: event.ID}))
		if err != nil {
			t.Fatal(err)
		}
		if rows.Msg.TotalRegistered != registered || rows.Msg.TotalAttended != attendees || rows.Msg.TotalCheckedIn != checkedIn || rows.Msg.TotalNoShow != noShow {
			t.Errorf("attendance totals: want registered=%d attendees=%d checked_in=%d no_show=%d, got %v", registered, attendees, checkedIn, noShow, rows.Msg)
		}
		stats, err := analytics.GetEventStatistics(ctx, connect.NewRequest(&eventsv1.GetEventStatisticsRequest{EventId: event.ID}))
		if err != nil {
			t.Fatal(err)
		}
		wantRate := float64(0)
		if registered > 0 {
			wantRate = float64(attendees) / float64(registered) * 100
		}
		if stats.Msg.TotalRegistrations != registered || stats.Msg.TotalAttendees != attendees || stats.Msg.CheckedIn != checkedIn || stats.Msg.NoShow != noShow || stats.Msg.AttendanceRate != wantRate {
			t.Errorf("event statistics: want registered=%d attendees=%d checked_in=%d no_show=%d rate=%v, got %v", registered, attendees, checkedIn, noShow, wantRate, stats.Msg)
		}
	}
	t.Run("before walk-in", func(t *testing.T) { assertCounts(t, 0, 0, 0, 0) })
	info, err := attendance.GetCheckInInfo(ctx, connect.NewRequest(&eventsv1.GetCheckInInfoRequest{EventId: event.ID}))
	if err != nil || info.Msg.CheckInToken == nil || info.Msg.CurrentRotatingToken == nil {
		t.Fatalf("get live QR code: response=%v error=%v", info, err)
	}
	for _, stage := range []string{"first QR scan", "repeated QR scan"} {
		checked, err := attendance.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{
			Token: *info.Msg.CheckInToken, RotatingToken: *info.Msg.CurrentRotatingToken,
		}))
		if err != nil || !checked.Msg.Success {
			t.Fatalf("self check-in: response=%v error=%v", checked, err)
		}
		t.Run(stage, func(t *testing.T) { assertCounts(t, 1, 1, 1, 0) })
	}
	rows, err := attendance.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: event.ID}))
	if err != nil || len(rows.Msg.Attendance) != 1 {
		t.Fatalf("QR retry must preserve one attendance row: response=%v error=%v", rows, err)
	}
	if rows.Msg.Attendance[0].Status != eventsv1.AttendanceStatus_ATTENDANCE_STATUS_CHECKED_IN || rows.Msg.Attendance[0].GetMethod() != eventsv1.CheckInMethod_CHECK_IN_METHOD_SELF {
		t.Fatalf("QR scan must preserve checked-in status and self method: %v", rows.Msg.Attendance[0])
	}
	registrationID := rows.Msg.Attendance[0].RegistrationId
	for _, stage := range []struct {
		name      string
		status    eventsv1.AttendanceStatus
		attendees int32
		noShow    int32
	}{
		{"confirmed attendance", eventsv1.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED, 1, 0},
		{"marked no-show", eventsv1.AttendanceStatus_ATTENDANCE_STATUS_NO_SHOW, 0, 1},
	} {
		if _, err := attendance.MarkAttendance(ctx, connect.NewRequest(&eventsv1.MarkAttendanceRequest{
			RegistrationId: registrationID, Status: stage.status,
		})); err != nil {
			t.Fatal(err)
		}
		t.Run(stage.name, func(t *testing.T) { assertCounts(t, 1, stage.attendees, 0, stage.noShow) })
	}
	if _, err := registrations.CancelRegistration(ctx, connect.NewRequest(&eventsv1.CancelRegistrationRequest{
		RegistrationId: &registrationID,
	})); err != nil {
		t.Fatal(err)
	}
	t.Run("cancelled registration", func(t *testing.T) { assertCounts(t, 0, 0, 0, 0) })
	rows, err = attendance.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: event.ID}))
	if err != nil || len(rows.Msg.Attendance) != 1 || rows.Msg.Attendance[0].Status != eventsv1.AttendanceStatus_ATTENDANCE_STATUS_NO_SHOW {
		t.Fatalf("cancellation must retain attendance history: response=%v error=%v", rows, err)
	}
}

func TestSelfCheckInPropagatesAttendanceWriteFailure(t *testing.T) {
	db := newTestDB(t)
	actor := seedAuthUser(t, db, "attendance-write-user")
	org := seedOrganization(t, db, "Attendance workflow")
	event := seedDraftEvent(t, db, actor, org.ID, "Attendance storage outage", "Hall")
	ctx := authedContext("attendance-write-user")
	if _, err := db.NewUpdate().Model(&event).Set("status = ?", models.EventStatusPublished).
		Set("start_time = ?", time.Now().Add(-time.Minute)).Set("end_time = ?", time.Now().Add(time.Hour)).WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	registrations := services.NewEventRegistrationsService(db, nil, enforcer)
	if _, err := registrations.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{EventId: event.ID, UserId: actor})); err != nil {
		t.Fatal(err)
	}
	attendance := eventattendance.NewService(db, enforcer, "")
	info, err := attendance.GetCheckInInfo(ctx, connect.NewRequest(&eventsv1.GetCheckInInfoRequest{EventId: event.ID}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE event_attendance ADD CONSTRAINT injected_write_failure CHECK (false)"); err != nil {
		t.Fatal(err)
	}
	response, err := attendance.SelfCheckIn(ctx, connect.NewRequest(&eventsv1.SelfCheckInRequest{Token: *info.Msg.CheckInToken, RotatingToken: *info.Msg.CurrentRotatingToken}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("attendance write failed but check-in did not report internal error: response=%v err=%v", response, err)
	}
	count, err := db.NewSelect().Model((*models.EventAttendance)(nil)).Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("failed check-in must not persist attendance: count=%d err=%v", count, err)
	}
}
