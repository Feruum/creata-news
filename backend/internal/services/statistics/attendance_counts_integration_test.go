//go:build integration

package statistics

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

type statisticsAttendanceFixture struct {
	db          *bun.DB
	service     *Service
	ctx         context.Context
	org         int32
	emptyOrg    int32
	firstEvent  int32
	secondEvent int32
	emptyEvent  int32
	onceUser    int32
}

func newStatisticsAttendanceFixture(t *testing.T) statisticsAttendanceFixture {
	t.Helper()
	db := testsupport.ClonedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "attendance-statistics-admin")
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	actor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('attendance-statistics-admin','attendance-statistics-admin','attendance-statistics-admin@test.local','test') RETURNING id")
	exec("INSERT INTO platform_user_roles(user_id,role) VALUES(?,'staff')", actor)
	typ := insert("INSERT INTO organization_types(title) VALUES('Attendance statistics type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Attendance statistics org',?) RETURNING id", typ)
	emptyOrg := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Empty statistics org',?) RETURNING id", typ)
	now := time.Now().UTC()
	seedEvent := func(title, token string, start time.Time) int32 {
		t.Helper()
		return insert("INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token) VALUES(?,'{}',?,?,?,?,'{}','published',?,?) RETURNING id", title, actor, org, start, start.Add(time.Hour), token, token)
	}
	fixture := statisticsAttendanceFixture{
		db: db, service: NewService(db, authz.NewEnforcer(db, testsupport.AllowedPermissions(t))), ctx: ctx,
		org: org, emptyOrg: emptyOrg,
		firstEvent:  seedEvent("First attendance event", "stats-first", now.Add(-24*time.Hour)),
		secondEvent: seedEvent("Second attendance event", "stats-second", now.Add(time.Hour)),
		emptyEvent:  seedEvent("Empty attendance event", "stats-empty", now.Add(-72*time.Hour)),
	}
	var repeatUser, cancelledUser int32
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
		name := fmt.Sprintf("attendance-statistics-user-%d", i)
		user := insert("INSERT INTO users(kratos_id,username,email,password) VALUES(?,?,?,'test') RETURNING id", name, name, name+"@test.local")
		if i == 0 {
			repeatUser = user
		} else if i == 1 {
			fixture.onceUser = user
		} else if i == 5 {
			cancelledUser = user
		}
		registration := insert("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,?) RETURNING id", fixture.firstEvent, user, state.registration)
		if state.attendance != "" {
			exec("INSERT INTO event_attendance(registration_id,status) VALUES(?,?)", registration, state.attendance)
		}
	}
	registration := insert("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered') RETURNING id", fixture.secondEvent, repeatUser)
	exec("INSERT INTO event_attendance(registration_id,status) VALUES(?,'checked_in')", registration)
	cancelledRegistration := insert("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'cancelled') RETURNING id", fixture.secondEvent, cancelledUser)
	exec("INSERT INTO event_attendance(registration_id,status) VALUES(?,'attended')", cancelledRegistration)
	return fixture
}

func assertAttendanceRate(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > 0.000001 {
		t.Errorf("attendance rate: want %v, got %v", want, got)
	}
}

func TestStatisticsUsesActiveCheckedInAndAttendedRegistrations(t *testing.T) {
	f := newStatisticsAttendanceFixture(t)
	t.Run("per event counts and conversion", func(t *testing.T) {
		for _, want := range []struct {
			event, registered, attendees, checkedIn, noShow int32
			rate                                            float64
		}{
			{f.firstEvent, 4, 2, 1, 1, 50},
			{f.secondEvent, 1, 1, 1, 0, 100},
			{f.emptyEvent, 0, 0, 0, 0, 0},
		} {
			response, err := f.service.GetEventStatistics(f.ctx, connect.NewRequest(&eventsv1.GetEventStatisticsRequest{EventId: want.event}))
			if err != nil {
				t.Fatal(err)
			}
			if response.Msg.TotalRegistrations != want.registered || response.Msg.TotalAttendees != want.attendees || response.Msg.CheckedIn != want.checkedIn || response.Msg.NoShow != want.noShow {
				t.Errorf("event %d counts: want registered=%d attendees=%d checked_in=%d no_show=%d, got %v", want.event, want.registered, want.attendees, want.checkedIn, want.noShow, response.Msg)
			}
			assertAttendanceRate(t, response.Msg.AttendanceRate, want.rate)
		}
	})
	t.Run("dashboard and recent events", func(t *testing.T) {
		response, err := f.service.GetDashboardStatistics(f.ctx, connect.NewRequest(&eventsv1.GetDashboardStatisticsRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		stats := response.Msg.Statistics
		if stats.TotalEvents != 3 || stats.TotalRegistrations != 5 || stats.TotalAttendees != 3 || stats.UpcomingEvents != 1 || stats.PastEvents != 2 {
			t.Errorf("dashboard attendance counts: %v", stats)
		}
		if len(stats.RecentEvents) != 3 {
			t.Fatalf("dashboard should include all three recent events: %v", stats.RecentEvents)
		}
		for _, event := range stats.RecentEvents {
			var registered, attendees int32
			var rate float64
			switch event.EventId {
			case f.firstEvent:
				registered, attendees, rate = 4, 2, 50
			case f.secondEvent:
				registered, attendees, rate = 1, 1, 100
			case f.emptyEvent:
			default:
				t.Errorf("unexpected recent event: %v", event)
			}
			if event.Registrations != registered || event.Attendees != attendees {
				t.Errorf("recent event attendance counts: %v", event)
			}
			assertAttendanceRate(t, event.AttendanceRate, rate)
		}
	})
	t.Run("overall average includes empty event", func(t *testing.T) {
		response, err := f.service.GetOverallStatistics(f.ctx, connect.NewRequest(&eventsv1.GetOverallStatisticsRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if response.Msg.TotalEvents != 3 || response.Msg.TotalRegistrations != 5 {
			t.Errorf("overall active registration counts: %v", response.Msg)
		}
		assertAttendanceRate(t, response.Msg.AverageAttendanceRate, 50)
	})
	t.Run("top events", func(t *testing.T) {
		response, err := f.service.GetTopPerformingEvents(f.ctx, connect.NewRequest(&eventsv1.GetTopPerformingEventsRequest{Limit: 10, Days: 90}))
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Msg.Events) != 3 {
			t.Fatalf("expected three top event rows: %v", response.Msg.Events)
		}
		for _, event := range response.Msg.Events {
			var registered, attendees int32
			var rate float64
			switch event.Id {
			case f.firstEvent:
				registered, attendees, rate = 4, 2, 50
			case f.secondEvent:
				registered, attendees, rate = 1, 1, 100
			case f.emptyEvent:
			default:
				t.Errorf("unexpected top event: %v", event)
			}
			if event.TotalRegistrations != registered || event.TotalAttendees != attendees {
				t.Errorf("top event attendance counts: %v", event)
			}
			assertAttendanceRate(t, event.AttendanceRate, rate)
		}
	})
	t.Run("club conversion", func(t *testing.T) {
		response, err := f.service.GetTopPerformingClubs(f.ctx, connect.NewRequest(&eventsv1.GetTopPerformingClubsRequest{Limit: 10, Days: 90}))
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Msg.Clubs) != 1 {
			t.Fatalf("expected one club with events: %v", response.Msg.Clubs)
		}
		club := response.Msg.Clubs[0]
		if club.OrganizationId != f.org || club.TotalEvents != 3 || club.TotalRegistrations != 5 || club.TotalAttendees != 3 {
			t.Errorf("club attendance counts: %v", club)
		}
		assertAttendanceRate(t, club.AverageAttendanceRate, 60)
	})
	t.Run("organization average attendance", func(t *testing.T) {
		response, err := f.service.GetOrganizationActivity(f.ctx, connect.NewRequest(&eventsv1.GetOrganizationActivityRequest{Limit: 10}))
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Msg.Organizations) != 2 {
			t.Fatalf("expected both organizations: %v", response.Msg.Organizations)
		}
		for _, org := range response.Msg.Organizations {
			if org.Id == f.org {
				if org.TotalEvents != 3 {
					t.Errorf("organization event count: %v", org)
				}
				assertAttendanceRate(t, org.AverageAttendance, 1)
			} else if org.Id == f.emptyOrg {
				assertAttendanceRate(t, org.AverageAttendance, 0)
			} else {
				t.Errorf("unexpected organization: %v", org)
			}
		}
	})
	t.Run("engagement includes live check-ins", func(t *testing.T) {
		response, err := f.service.GetUserEngagementLevels(f.ctx, connect.NewRequest(&eventsv1.GetUserEngagementLevelsRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if response.Msg.TotalUsers != 11 {
			t.Errorf("engagement user count: %v", response.Msg)
		}
		want := map[string]int32{"active_users": 11, "registered_for_events": 4, "attended_events": 2, "repeat_attendees": 1}
		for _, level := range response.Msg.Levels {
			count, ok := want[level.Level]
			if !ok || level.Count != count {
				t.Errorf("engagement level: %v", level)
			}
			assertAttendanceRate(t, level.Percentage, float64(count)/11*100)
			delete(want, level.Level)
		}
		if len(want) != 0 {
			t.Errorf("missing engagement levels: %v", want)
		}
	})
}

func TestRepeatAttendeesRequireDistinctEvents(t *testing.T) {
	f := newStatisticsAttendanceFixture(t)
	if _, err := f.db.ExecContext(f.ctx, "DELETE FROM event_attendance ea USING event_registrations er WHERE ea.registration_id = er.id AND er.event_id = ?", f.secondEvent); err != nil {
		t.Fatal(err)
	}
	var duplicateRegistration int32
	if err := f.db.NewRaw("INSERT INTO event_registrations(event_id,user_id,status) VALUES(?,?,'registered') RETURNING id", f.firstEvent, f.onceUser).Scan(f.ctx, &duplicateRegistration); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, "INSERT INTO event_attendance(registration_id,status) VALUES(?,'attended')", duplicateRegistration); err != nil {
		t.Fatal(err)
	}
	response, err := f.service.GetUserEngagementLevels(f.ctx, connect.NewRequest(&eventsv1.GetUserEngagementLevelsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, level := range response.Msg.Levels {
		if level.Level == "repeat_attendees" {
			if level.Count != 0 || level.Percentage != 0 {
				t.Fatalf("two registrations for one event are not repeat attendance: %v", level)
			}
			return
		}
	}
	t.Fatal("repeat attendance level missing")
}

func TestAttendanceStatisticsPropagateStorageFailures(t *testing.T) {
	for _, table := range []string{"event_attendance", "event_registrations"} {
		t.Run(table, func(t *testing.T) {
			f := newStatisticsAttendanceFixture(t)
			if _, err := f.db.ExecContext(f.ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO unavailable_%s", table, table)); err != nil {
				t.Fatal(err)
			}
			for name, call := range map[string]func() error{
				"event": func() error {
					_, err := f.service.GetEventStatistics(f.ctx, connect.NewRequest(&eventsv1.GetEventStatisticsRequest{EventId: f.firstEvent}))
					return err
				},
				"dashboard": func() error {
					_, err := f.service.GetDashboardStatistics(f.ctx, connect.NewRequest(&eventsv1.GetDashboardStatisticsRequest{}))
					return err
				},
				"overall": func() error {
					_, err := f.service.GetOverallStatistics(f.ctx, connect.NewRequest(&eventsv1.GetOverallStatisticsRequest{}))
					return err
				},
				"top events": func() error {
					_, err := f.service.GetTopPerformingEvents(f.ctx, connect.NewRequest(&eventsv1.GetTopPerformingEventsRequest{}))
					return err
				},
				"clubs": func() error {
					_, err := f.service.GetTopPerformingClubs(f.ctx, connect.NewRequest(&eventsv1.GetTopPerformingClubsRequest{}))
					return err
				},
				"organization activity": func() error {
					_, err := f.service.GetOrganizationActivity(f.ctx, connect.NewRequest(&eventsv1.GetOrganizationActivityRequest{}))
					return err
				},
				"engagement": func() error {
					_, err := f.service.GetUserEngagementLevels(f.ctx, connect.NewRequest(&eventsv1.GetUserEngagementLevelsRequest{}))
					return err
				},
			} {
				t.Run(name, func(t *testing.T) {
					if err := call(); connect.CodeOf(err) != connect.CodeInternal {
						t.Fatalf("attendance query failure should be internal, got %v", err)
					}
				})
			}
		})
	}
}
