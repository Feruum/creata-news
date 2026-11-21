package statistics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetDashboardStatistics(ctx context.Context, req *connect.Request[eventsv1.GetDashboardStatisticsRequest]) (*connect.Response[eventsv1.GetDashboardStatisticsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetDashboardStatistics")

	// Get total counts
	var totalEvents, totalRegs, totalAttendees int32
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events`).Scan(ctx, &totalEvents)
	if err := s.db.NewRaw(`SELECT COUNT(*) FROM event_registrations WHERE status = 'registered'`).Scan(ctx, &totalRegs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count dashboard registrations: %w", err))
	}
	if err := s.db.NewRaw(`
		SELECT COUNT(DISTINCT er.id) FROM event_registrations er
		INNER JOIN event_attendance ea ON ea.registration_id = er.id
		WHERE er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
	`).Scan(ctx, &totalAttendees); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count dashboard attendees: %w", err))
	}

	now := time.Now()
	var upcomingEvents, pastEvents int32
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events WHERE start_time >= ?`, now).Scan(ctx, &upcomingEvents)
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events WHERE start_time < ?`, now).Scan(ctx, &pastEvents)

	// Get recent events with stats
	type recentEvent struct {
		ID        int32     `bun:"id"`
		Title     string    `bun:"title"`
		StartTime time.Time `bun:"start_time"`
	}
	var recent []recentEvent
	err := s.db.NewRaw(`
		SELECT id, title, start_time
		FROM events
		ORDER BY start_time DESC
		LIMIT 5
	`).Scan(ctx, &recent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var recentEvents []*eventsv1.EventStats
	for _, e := range recent {
		var regs, attended int32
		if err := s.db.NewRaw(`SELECT COUNT(*) FROM event_registrations WHERE event_id = ? AND status = 'registered'`, e.ID).Scan(ctx, &regs); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count recent event registrations: %w", err))
		}
		if err := s.db.NewRaw(`
			SELECT COUNT(DISTINCT er.id) FROM event_attendance ea
			INNER JOIN event_registrations er ON er.id = ea.registration_id
			WHERE er.event_id = ? AND er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
		`, e.ID).Scan(ctx, &attended); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count recent event attendees: %w", err))
		}

		recentEvents = append(recentEvents, &eventsv1.EventStats{
			EventId:        e.ID,
			EventTitle:     e.Title,
			Registrations:  regs,
			Attendees:      attended,
			AttendanceRate: safePercentage(attended, regs),
			StartTime:      e.StartTime.Format(time.RFC3339),
		})
	}

	return connect.NewResponse(&eventsv1.GetDashboardStatisticsResponse{
		Statistics: &eventsv1.EventStatistics{
			TotalEvents:        totalEvents,
			TotalRegistrations: totalRegs,
			TotalAttendees:     totalAttendees,
			UpcomingEvents:     upcomingEvents,
			PastEvents:         pastEvents,
			RecentEvents:       recentEvents,
		},
	}), nil
}
