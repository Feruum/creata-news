package statistics

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetEventStatistics(ctx context.Context, req *connect.Request[eventsv1.GetEventStatisticsRequest]) (*connect.Response[eventsv1.GetEventStatisticsResponse], error) {
	if err := s.requireEventAnalytics(ctx, req.Msg.EventId); err != nil {
		return nil, err
	}
	slog.Debug("GetEventStatistics", "eventId", req.Msg.EventId)

	var totalRegs, totalAttended, checkedIn, noShow int32
	if err := s.db.NewRaw(`SELECT COUNT(*) FROM event_registrations WHERE event_id = ? AND status = 'registered'`, req.Msg.EventId).Scan(ctx, &totalRegs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.db.NewRaw(`
		SELECT COUNT(DISTINCT er.id) FROM event_attendance ea
		INNER JOIN event_registrations er ON er.id = ea.registration_id
		WHERE er.event_id = ? AND er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
	`, req.Msg.EventId).Scan(ctx, &totalAttended); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count event attendees: %w", err))
	}
	if err := s.db.NewRaw(`
		SELECT COUNT(DISTINCT er.id) FROM event_attendance ea
		INNER JOIN event_registrations er ON er.id = ea.registration_id
		WHERE er.event_id = ? AND er.status = 'registered' AND ea.status = 'checked_in'
	`, req.Msg.EventId).Scan(ctx, &checkedIn); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count event check-ins: %w", err))
	}
	if err := s.db.NewRaw(`
		SELECT COUNT(DISTINCT er.id) FROM event_attendance ea
		INNER JOIN event_registrations er ON er.id = ea.registration_id
		WHERE er.event_id = ? AND er.status = 'registered' AND ea.status = 'no_show'
	`, req.Msg.EventId).Scan(ctx, &noShow); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count event no-shows: %w", err))
	}

	return connect.NewResponse(&eventsv1.GetEventStatisticsResponse{
		TotalRegistrations: totalRegs,
		TotalAttendees:     totalAttended,
		CheckedIn:          checkedIn,
		NoShow:             noShow,
		AttendanceRate:     safePercentage(totalAttended, totalRegs),
	}), nil
}
