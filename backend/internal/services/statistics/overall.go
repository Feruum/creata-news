package statistics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetOverallStatistics(ctx context.Context, req *connect.Request[eventsv1.GetOverallStatisticsRequest]) (*connect.Response[eventsv1.GetOverallStatisticsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetOverallStatistics")

	var totalEvents, totalUsers, totalOrgs, totalRegs, upcomingEvents int32
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events`).Scan(ctx, &totalEvents)
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM users`).Scan(ctx, &totalUsers)
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM organizations`).Scan(ctx, &totalOrgs)
	if err := s.db.NewRaw(`SELECT COUNT(*) FROM event_registrations WHERE status = 'registered'`).Scan(ctx, &totalRegs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count overall registrations: %w", err))
	}
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events WHERE start_time >= NOW()`).Scan(ctx, &upcomingEvents)

	// Calculate average attendance rate
	var avgRate float64
	if err := s.db.NewRaw(`
		SELECT COALESCE(AVG(
			CASE WHEN reg_count > 0 THEN att_count::float / reg_count::float * 100 ELSE 0 END
		), 0)
		FROM (
			SELECT e.id,
				COUNT(DISTINCT CASE WHEN er.status = 'registered' THEN er.id END) as reg_count,
				COUNT(DISTINCT CASE WHEN er.status = 'registered' AND ea.status IN ('checked_in', 'attended') THEN er.id END) as att_count
			FROM events e
			LEFT JOIN event_registrations er ON er.event_id = e.id
			LEFT JOIN event_attendance ea ON ea.registration_id = er.id
			GROUP BY e.id
		) stats
	`).Scan(ctx, &avgRate); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("calculate average attendance rate: %w", err))
	}

	// Events this month
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var eventsThisMonth, regsThisMonth int32
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM events WHERE start_time >= ? AND start_time < ?`, monthStart, monthEnd).Scan(ctx, &eventsThisMonth)
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM event_registrations WHERE registered_at >= ? AND registered_at < ?`, monthStart, monthEnd).Scan(ctx, &regsThisMonth)

	return connect.NewResponse(&eventsv1.GetOverallStatisticsResponse{
		TotalEvents:            totalEvents,
		TotalUsers:             totalUsers,
		TotalOrganizations:     totalOrgs,
		TotalRegistrations:     totalRegs,
		UpcomingEvents:         upcomingEvents,
		AverageAttendanceRate:  avgRate,
		EventsThisMonth:        eventsThisMonth,
		RegistrationsThisMonth: regsThisMonth,
	}), nil
}
