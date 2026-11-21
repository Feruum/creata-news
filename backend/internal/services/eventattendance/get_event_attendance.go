package eventattendance

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetEventAttendance(ctx context.Context, req *connect.Request[eventsv1.GetEventAttendanceRequest]) (*connect.Response[eventsv1.GetEventAttendanceResponse], error) {
	slog.Debug("GetEventAttendance", "eventId", req.Msg.EventId)

	_, err := s.authz.RequireResourcePermission(ctx, "event", fmt.Sprintf("%d", req.Msg.EventId), "manage_registrations")
	if err != nil {
		return nil, err
	}

	var attendanceList []models.EventAttendance
	err = s.db.NewSelect().
		Model(&attendanceList).
		Join("JOIN event_registrations AS er ON er.id = ea.registration_id").
		Where("er.event_id = ?", req.Msg.EventId).
		Scan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Count attendance stats using raw query
	var totalRegistered, attended, noShow, checkedIn int32
	err = s.db.NewRaw(`
		SELECT
			COUNT(DISTINCT er.id) as total_registered,
			COUNT(DISTINCT er.id) FILTER (WHERE ea.status IN ('checked_in', 'attended')) as attended,
			COUNT(DISTINCT er.id) FILTER (WHERE ea.status = 'no_show') as no_show,
			COUNT(DISTINCT er.id) FILTER (WHERE ea.status = 'checked_in') as checked_in
		FROM event_registrations er
		LEFT JOIN event_attendance ea ON ea.registration_id = er.id
		WHERE er.event_id = ? AND er.status = 'registered'
	`, req.Msg.EventId).Scan(ctx, &totalRegistered, &attended, &noShow, &checkedIn)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load event attendance totals: %w", err))
	}

	protoAtt := lo.Map(attendanceList, func(attendance models.EventAttendance, _ int) *eventsv1.EventAttendance {
		return dbEventAttendanceToProto(attendance)
	})

	return connect.NewResponse(&eventsv1.GetEventAttendanceResponse{
		Attendance:      protoAtt,
		TotalRegistered: totalRegistered,
		TotalAttended:   attended,
		TotalNoShow:     noShow,
		TotalCheckedIn:  checkedIn,
	}), nil
}
