package statistics

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetSundayOpenAnalytics(ctx context.Context, req *connect.Request[eventsv1.GetSundayOpenAnalyticsRequest]) (*connect.Response[eventsv1.GetSundayOpenAnalyticsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	var rows []struct {
		Kind    string
		Key     string
		Count   int32
		EventID int32
		Title   string
	}
	// One statement provides one snapshot. Each channel produces one row per
	// registration; attendance is an existence check, never a multiplying join.
	err := s.db.NewRaw(`WITH scoped AS MATERIALIZED (
 SELECT id,title FROM events WHERE publish_on_sunday_open AND status IN ('published','archived')
 ), registrations AS MATERIALIZED (
 SELECT er.event_id,er.registered_at,'ems' AS channel,
 EXISTS(SELECT 1 FROM event_attendance a WHERE a.registration_id=er.id AND a.status IN ('checked_in','attended')) AS attended
 FROM event_registrations er JOIN scoped e ON e.id=er.event_id WHERE er.status='registered'
 UNION ALL
 SELECT xr.event_id,xr.registered_at,'external',xr.checked_in_at IS NOT NULL
 FROM external_registrations xr JOIN scoped e ON e.id=xr.event_id WHERE xr.status='registered'
 ), weeks AS (
 SELECT generate_series(date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Qyzylorda') - interval '6 weeks',
 date_trunc('week',CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Qyzylorda'),interval '1 week') AS start
 ), top_events AS (
 SELECT e.id,e.title,count(r.event_id)::integer AS count FROM scoped e
 LEFT JOIN registrations r ON r.event_id=e.id GROUP BY e.id,e.title ORDER BY count DESC,e.id LIMIT 4
 )
 SELECT 'events' AS kind,'' AS key,count(*)::integer AS count,0::integer AS event_id,'' AS title FROM scoped
 UNION ALL SELECT 'ems','',count(*)::integer,0,'' FROM registrations WHERE channel='ems'
 UNION ALL SELECT 'external','',count(*)::integer,0,'' FROM registrations WHERE channel='external'
 UNION ALL SELECT 'attendees','',count(*)::integer,0,'' FROM registrations WHERE attended
 UNION ALL SELECT 'week',to_char(w.start,'YYYY-MM-DD'),count(r.event_id)::integer,0,'' FROM weeks w
 LEFT JOIN registrations r ON r.registered_at >= (w.start AT TIME ZONE 'Asia/Qyzylorda')
 AND r.registered_at < ((w.start + interval '1 week') AT TIME ZONE 'Asia/Qyzylorda')
 GROUP BY w.start
 UNION ALL SELECT 'event','',count,id,title FROM top_events
 ORDER BY kind,key,count DESC,event_id`).Scan(ctx, &rows)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Sunday Open analytics: %w", err))
	}
	response := &eventsv1.GetSundayOpenAnalyticsResponse{}
	for _, row := range rows {
		switch row.Kind {
		case "events":
			response.TotalEvents = row.Count
		case "ems":
			response.EmsRegistrations = row.Count
		case "external":
			response.ExternalRegistrations = row.Count
		case "attendees":
			response.TotalAttendees = row.Count
		case "week":
			response.Weeks = append(response.Weeks, &eventsv1.SundayOpenRegistrationWeek{WeekStart: row.Key, Registrations: row.Count})
		case "event":
			response.TopEvents = append(response.TopEvents, &eventsv1.SundayOpenAnalyticsEvent{EventId: row.EventID, Title: row.Title, Registrations: row.Count})
		}
	}
	response.TotalRegistrations = response.EmsRegistrations + response.ExternalRegistrations
	return connect.NewResponse(response), nil
}
