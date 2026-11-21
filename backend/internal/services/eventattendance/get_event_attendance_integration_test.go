//go:build integration

package eventattendance

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func attendanceCountsFixture(t *testing.T) (*bun.DB, *Service, context.Context, int32, int32) {
	t.Helper()
	db := testsupport.ClonedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "attendance-count-admin")
	if _, err := db.ExecContext(ctx, "TRUNCATE events, organizations, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatal(err)
	}
	insert := func(query string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(query, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	actor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('attendance-count-admin','attendance-count-admin','attendance-count-admin@test.local','test') RETURNING id")
	typ := insert("INSERT INTO organization_types(title) VALUES('Attendance counts type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Attendance counts org',?) RETURNING id", typ)
	event := insert("INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token) VALUES('Attendance counts','{}',?,?,NOW(),NOW()+interval '1 hour','{}','published','counts-secret','counts-token') RETURNING id", actor, org)
	empty := insert("INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token) VALUES('Empty attendance counts','{}',?,?,NOW(),NOW()+interval '1 hour','{}','published','empty-secret','empty-token') RETURNING id", actor, org)
	return db, NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), ""), ctx, event, empty
}

func TestGetEventAttendanceCountsActiveRegistrationsAndRetainsHistory(t *testing.T) {
	db, svc, ctx, event, empty := attendanceCountsFixture(t)
	for i, state := range []struct{ registration, attendance string }{
		{"registered", "checked_in"},
		{"registered", "attended"},
		{"registered", "no_show"},
		{"registered", ""},
		{"cancelled", "checked_in"},
		{"cancelled", "attended"},
		{"cancelled", "no_show"},
		{"waitlist", "checked_in"},
		{"waitlist", "attended"},
		{"waitlist", "no_show"},
	} {
		name := fmt.Sprintf("attendance-user-%d", i)
		var user, registration int32
		if err := db.NewRaw("INSERT INTO users(kratos_id,username,email,password) VALUES(?,?,?,'test') RETURNING id", name, name, name+"@test.local").Scan(ctx, &user); err != nil {
			t.Fatal(err)
		}
		if err := db.NewRaw("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,?) RETURNING id", event, user, state.registration).Scan(ctx, &registration); err != nil {
			t.Fatal(err)
		}
		if state.attendance != "" {
			if _, err := db.ExecContext(ctx, "INSERT INTO event_attendance(registration_id,status,method) VALUES(?,?,'self')", registration, state.attendance); err != nil {
				t.Fatal(err)
			}
		}
	}
	response, err := svc.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: event}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalRegistered != 4 || response.Msg.TotalAttended != 2 || response.Msg.TotalCheckedIn != 1 || response.Msg.TotalNoShow != 1 {
		t.Errorf("only active registrations belong in totals: %v", response.Msg)
	}
	if len(response.Msg.Attendance) != 9 {
		t.Fatalf("inactive attendance history must remain in rows: %v", response.Msg.Attendance)
	}
	statuses := map[eventsv1.AttendanceStatus]int{}
	for _, row := range response.Msg.Attendance {
		statuses[row.Status]++
		if row.GetMethod() != eventsv1.CheckInMethod_CHECK_IN_METHOD_SELF {
			t.Errorf("counting must preserve check-in methods: %v", row)
		}
	}
	for _, status := range []eventsv1.AttendanceStatus{
		eventsv1.AttendanceStatus_ATTENDANCE_STATUS_CHECKED_IN,
		eventsv1.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED,
		eventsv1.AttendanceStatus_ATTENDANCE_STATUS_NO_SHOW,
	} {
		if statuses[status] != 3 {
			t.Errorf("history should preserve three %v rows, got %d", status, statuses[status])
		}
	}
	response, err = svc.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: empty}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalRegistered != 0 || response.Msg.TotalAttended != 0 || response.Msg.TotalCheckedIn != 0 || response.Msg.TotalNoShow != 0 || len(response.Msg.Attendance) != 0 {
		t.Fatalf("empty event should have no attendance totals or rows: %v", response.Msg)
	}
}

func TestGetEventAttendancePropagatesStorageFailure(t *testing.T) {
	db, svc, ctx, event, _ := attendanceCountsFixture(t)
	if _, err := db.ExecContext(ctx, "ALTER TABLE event_attendance RENAME TO unavailable_event_attendance"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetEventAttendance(ctx, connect.NewRequest(&eventsv1.GetEventAttendanceRequest{EventId: event})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("attendance storage failure should be internal, got %v", err)
	}
}
