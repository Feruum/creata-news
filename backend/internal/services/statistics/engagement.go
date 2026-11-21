package statistics

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetUserEngagementLevels(ctx context.Context, req *connect.Request[eventsv1.GetUserEngagementLevelsRequest]) (*connect.Response[eventsv1.GetUserEngagementLevelsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetUserEngagementLevels")

	var totalUsers int32
	_ = s.db.NewRaw(`SELECT COUNT(*) FROM users`).Scan(ctx, &totalUsers)

	var registeredUsers int32
	if err := s.db.NewRaw(`SELECT COUNT(DISTINCT user_id) FROM event_registrations WHERE status = 'registered'`).Scan(ctx, &registeredUsers); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count registered users: %w", err))
	}

	var attendedUsers int32
	if err := s.db.NewRaw(`
		SELECT COUNT(DISTINCT er.user_id) FROM event_registrations er
		INNER JOIN event_attendance ea ON ea.registration_id = er.id
		WHERE er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
	`).Scan(ctx, &attendedUsers); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count attending users: %w", err))
	}

	var repeatAttendees int32
	if err := s.db.NewRaw(`
		SELECT COUNT(*) FROM (
			SELECT er.user_id FROM event_registrations er
			INNER JOIN event_attendance ea ON ea.registration_id = er.id
			WHERE er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
			GROUP BY er.user_id
			HAVING COUNT(DISTINCT er.event_id) > 1
		) repeat_users
	`).Scan(ctx, &repeatAttendees); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count repeat attendees: %w", err))
	}

	levels := []*eventsv1.UserEngagementLevel{
		{Level: "active_users", Count: totalUsers, Percentage: 100},
		{Level: "registered_for_events", Count: registeredUsers, Percentage: safePercentage(registeredUsers, totalUsers)},
		{Level: "attended_events", Count: attendedUsers, Percentage: safePercentage(attendedUsers, totalUsers)},
		{Level: "repeat_attendees", Count: repeatAttendees, Percentage: safePercentage(repeatAttendees, totalUsers)},
	}

	return connect.NewResponse(&eventsv1.GetUserEngagementLevelsResponse{
		Levels:          levels,
		TotalUsers:      totalUsers,
		TrendMessage:    "Showing engagement metrics",
		Description:     "User engagement breakdown",
		IsPositiveTrend: true,
	}), nil
}
