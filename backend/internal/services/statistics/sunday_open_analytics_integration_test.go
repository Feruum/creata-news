//go:build integration

package statistics

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func TestSundayOpenAnalyticsCountsBothChannels(t *testing.T) {
	f := newStatisticsAttendanceFixture(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := f.db.ExecContext(f.ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("UPDATE events SET publish_on_sunday_open=true WHERE id IN (?,?)", f.firstEvent, f.emptyEvent)
	exec("UPDATE events SET status='archived' WHERE id=?", f.emptyEvent)
	exec(`INSERT INTO external_registrations(event_id,full_name,email,status,checked_in_at) VALUES
 (?, 'Visitor','one@test.local','registered',NOW()),
 (?, 'Visitor','two@test.local','registered',NULL),
 (?, 'Cancelled','cancel@test.local','cancelled',NOW()),
 (?, 'Pending','pending@test.local','pending_approval',NOW())`, f.firstEvent, f.firstEvent, f.firstEvent, f.firstEvent)
	request := connect.NewRequest(&eventsv1.GetSundayOpenAnalyticsRequest{})
	response, err := f.service.GetSundayOpenAnalytics(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	got := response.Msg
	if got.TotalEvents != 2 || got.EmsRegistrations != 4 || got.ExternalRegistrations != 2 || got.TotalRegistrations != 6 || got.TotalAttendees != 3 {
		t.Fatalf("wrong totals: %v", got)
	}
	if len(got.Weeks) != 7 {
		t.Fatalf("expected seven weeks: %v", got.Weeks)
	}
	var count int32
	for _, week := range got.Weeks {
		count += week.Registrations
	}
	if count != 6 {
		t.Fatalf("wrong weekly total: %v", got.Weeks)
	}
	if len(got.TopEvents) != 2 || got.TopEvents[0].EventId != f.firstEvent || got.TopEvents[0].Registrations != 6 {
		t.Fatalf("wrong top events: %v", got.TopEvents)
	}
	// Local Monday midnight belongs to the new week, even though it is
	// Sunday in UTC. A registration just before the window is excluded.
	exec(`UPDATE event_registrations SET registered_at=(date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Qyzylorda') - interval '6 weeks') AT TIME ZONE 'Asia/Qyzylorda' WHERE event_id=? AND user_id=?`, f.firstEvent, f.onceUser)
	exec(`UPDATE external_registrations SET registered_at=((date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Qyzylorda') - interval '6 weeks') AT TIME ZONE 'Asia/Qyzylorda') - interval '1 microsecond' WHERE email='two@test.local'`)
	response, err = f.service.GetSundayOpenAnalytics(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Weeks[0].Registrations != 1 || response.Msg.Weeks[6].Registrations != 4 || response.Msg.TotalRegistrations != 6 {
		t.Fatalf("incorrect week boundaries: %v", response.Msg)
	}
	for i, week := range response.Msg.Weeks {
		day, err := time.Parse("2006-01-02", week.WeekStart)
		if err != nil || day.Weekday() != time.Monday {
			t.Fatalf("invalid Monday: %v", week)
		}
		if i > 0 && i < 6 && week.Registrations != 0 {
			t.Fatalf("empty week is nonzero: %v", week)
		}
	}
	exec("UPDATE events SET status='draft' WHERE id=?", f.firstEvent)
	response, err = f.service.GetSundayOpenAnalytics(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalEvents != 1 || response.Msg.TotalRegistrations != 0 {
		t.Fatalf("draft counted: %v", response.Msg)
	}
	exec("UPDATE events SET status='cancelled' WHERE id=?", f.emptyEvent)
	response, err = f.service.GetSundayOpenAnalytics(f.ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.TotalEvents != 0 || response.Msg.TotalRegistrations != 0 || len(response.Msg.TopEvents) != 0 {
		t.Fatalf("empty scope: %v", response.Msg)
	}
	exec("ALTER TABLE external_registrations RENAME TO unavailable_external_registrations")
	if _, err = f.service.GetSundayOpenAnalytics(f.ctx, request); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("storage failure masked: %v", err)
	}
}

func TestSundayOpenAnalyticsIncludesArchivedHistoryAndLimitsRanking(t *testing.T) {
	f := newStatisticsAttendanceFixture(t)
	if _, err := f.db.ExecContext(f.ctx, `UPDATE events SET publish_on_sunday_open=true,status='archived' WHERE id=?;
	UPDATE event_registrations SET registered_at=NOW()-interval '1 year' WHERE event_id=?`, f.firstEvent, f.firstEvent); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.ExecContext(f.ctx, `INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,structured_location,status,check_in_secret,check_in_token,publish_on_sunday_open)
	SELECT 'Ranking event '||n,description,user_id,organization_id,start_time,end_time,structured_location,'published','ranking-'||n,'ranking-'||n,true
	FROM events CROSS JOIN generate_series(1,5) n WHERE id=?`, f.firstEvent); err != nil {
		t.Fatal(err)
	}
	response, err := f.service.GetSundayOpenAnalytics(f.ctx, connect.NewRequest(&eventsv1.GetSundayOpenAnalyticsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := response.Msg
	if got.TotalEvents != 6 || got.TotalRegistrations != 4 || got.TotalAttendees != 2 || len(got.TopEvents) != 4 || got.TopEvents[0].EventId != f.firstEvent || got.TopEvents[0].Registrations != 4 {
		t.Fatalf("incorrect historical ranking: %v", got)
	}
	for _, week := range got.Weeks {
		if week.Registrations != 0 {
			t.Fatalf("old registration in weekly window: %v", week)
		}
	}
}
