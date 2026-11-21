//go:build integration

package eventattendance

import (
	"context"
	"os"
	"strconv"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

func guardedAttendanceFixture(t *testing.T) (*bun.DB, *Service, context.Context) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("disposable DATABASE_URL required")
	}
	db := testsupport.IsolatedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "guarded-attendance")
	for _, table := range []string{"users", "events", "event_registrations", "external_registrations", "event_attendance", "event_registration_forms", "event_registration_form_versions"} {
		if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (LIKE public."+table+" INCLUDING ALL)"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users(id,kratos_id,username,email,password) VALUES(1,'guarded-attendance','Guarded','guarded@example.test','test'); INSERT INTO events(id,title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token,publish_on_sunday_open) VALUES(1,'Guarded','{}',1,1,NOW(),NOW()+interval '1 hour','{}','published','guarded-secret','guarded-token',true)`); err != nil {
		t.Fatal(err)
	}
	return db, NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), ""), ctx
}

func TestStudentWalkInBypassesFormAndApproval(t *testing.T) {
	for _, sunday := range []bool{false, true} {
		for _, configured := range []bool{false, true} {
			for _, status := range []string{"none", "cancelled", "registered", "pending_approval", "rejected", "waitlist"} {
				t.Run(status+"/configured="+strconv.FormatBool(configured)+"/sunday="+strconv.FormatBool(sunday), func(t *testing.T) {
					db, s, ctx := guardedAttendanceFixture(t)
					if _, err := db.ExecContext(ctx, `UPDATE events SET publish_on_sunday_open=?,capacity=1 WHERE id=1; INSERT INTO external_registrations(event_id,full_name,email,status) VALUES(1,'Existing guest','existing@example.test','registered')`, sunday); err != nil {
						t.Fatal(err)
					}
					s.WithForms(registrationforms.NewService(db, s.authz, nil))
					if configured {
						version := uuid.NewString()
						if _, err := db.ExecContext(ctx, `INSERT INTO event_registration_form_versions(id,event_id,definition) VALUES(?::uuid,1,?::jsonb); INSERT INTO event_registration_forms(event_id,draft,published_version_id) VALUES(1,'{}',?::uuid)`, version, `{"eventId":1,"versionId":"`+version+`","approvalMode":"APPROVAL_MODE_MANUAL"}`, version); err != nil {
							t.Fatal(err)
						}
					}
					if status != "none" {
						if _, err := db.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,?)`, status); err != nil {
							t.Fatal(err)
						}
					}
					response, err := s.SelfCheckIn(ctx, connect.NewRequest(&pb.SelfCheckInRequest{Token: "guarded-token", RotatingToken: computeRotatingToken("guarded-secret", getCurrentWindow())}))
					if err != nil || !response.Msg.Success {
						t.Fatalf("valid QR check-in blocked: %v %v", response, err)
					}
					var count int
					if err := db.NewRaw(`SELECT COUNT(*) FROM event_attendance`).Scan(ctx, &count); err != nil {
						t.Fatal(err)
					}
					if count != 1 {
						t.Fatalf("walk-in attendance count: %d", count)
					}
					var active int
					if err := db.NewRaw(`SELECT count(*) FROM event_registrations WHERE event_id=1 AND user_id=1 AND status='registered' AND cancelled_at IS NULL`).Scan(ctx, &active); err != nil || active != 1 {
						t.Fatalf("walk-in registration count: %d %v", active, err)
					}
				})
			}
		}
	}
}

func TestUnconfirmedRegistrationCannotCheckInThroughPrivilegedPaths(t *testing.T) {
	for _, status := range []string{"pending_approval", "rejected", "waitlist", "cancelled"} {
		for _, path := range []string{"mark", "late", "override", "manual"} {
			t.Run(status+"/"+path, func(t *testing.T) {
				db, s, ctx := guardedAttendanceFixture(t)
				if _, err := db.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,?)`, status); err != nil {
					t.Fatal(err)
				}
				var err error
				switch path {
				case "mark":
					_, err = s.MarkAttendance(ctx, connect.NewRequest(&pb.MarkAttendanceRequest{RegistrationId: 1, Status: pb.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED}))
				case "late":
					_, err = s.LateCheckIn(ctx, connect.NewRequest(&pb.LateCheckInRequest{RegistrationId: 1, Reason: "Organizer correction"}))
				case "override":
					_, err = s.StaffOverrideCheckIn(ctx, connect.NewRequest(&pb.StaffOverrideCheckInRequest{RegistrationId: 1}))
				case "manual":
					_, err = s.CheckInAttendee(ctx, connect.NewRequest(&pb.CheckInAttendeeRequest{RegistrationId: 1, CheckedInBy: 1}))
				}
				if connect.CodeOf(err) != connect.CodeFailedPrecondition {
					t.Errorf("%s %s bypassed confirmation: %v", status, path, err)
				}
				var saved string
				if err := db.NewRaw(`SELECT status FROM event_registrations WHERE id=1`).Scan(ctx, &saved); err != nil {
					t.Fatal(err)
				}
				if saved != status {
					t.Errorf("registration status changed to %s", saved)
				}
				var count int
				if err := db.NewRaw(`SELECT COUNT(*) FROM event_attendance`).Scan(ctx, &count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Error("unconfirmed attendance created")
				}
			})
		}
	}
}

func TestConfirmedRegistrationStillSupportsPrivilegedAttendance(t *testing.T) {
	for _, path := range []string{"mark", "late", "override", "manual"} {
		t.Run(path, func(t *testing.T) {
			db, s, ctx := guardedAttendanceFixture(t)
			if _, err := db.ExecContext(ctx, `INSERT INTO event_registrations(id,event_id,user_id,status) VALUES(1,1,1,'registered')`); err != nil {
				t.Fatal(err)
			}
			var err error
			switch path {
			case "mark":
				_, err = s.MarkAttendance(ctx, connect.NewRequest(&pb.MarkAttendanceRequest{RegistrationId: 1, Status: pb.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED}))
			case "late":
				_, err = s.LateCheckIn(ctx, connect.NewRequest(&pb.LateCheckInRequest{RegistrationId: 1, Reason: "Organizer correction"}))
			case "override":
				_, err = s.StaffOverrideCheckIn(ctx, connect.NewRequest(&pb.StaffOverrideCheckInRequest{RegistrationId: 1}))
			case "manual":
				_, err = s.CheckInAttendee(ctx, connect.NewRequest(&pb.CheckInAttendeeRequest{RegistrationId: 1, CheckedInBy: 1}))
			}
			if err != nil {
				t.Fatal(err)
			}
			var count int
			if err := db.NewRaw(`SELECT COUNT(*) FROM event_attendance WHERE registration_id=1 AND status IN ('checked_in','attended')`).Scan(ctx, &count); err != nil || count != 1 {
				t.Fatalf("confirmed attendance missing: %d %v", count, err)
			}
		})
	}
}
