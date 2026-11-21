package statistics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetTopPerformingClubs(ctx context.Context, req *connect.Request[eventsv1.GetTopPerformingClubsRequest]) (*connect.Response[eventsv1.GetTopPerformingClubsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetTopPerformingClubs", "limit", req.Msg.Limit, "days", req.Msg.Days)

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	}
	days := int(req.Msg.Days)
	if days <= 0 {
		days = 90
	}

	startDate := time.Now().AddDate(0, 0, -days)

	type clubRow struct {
		ID            int32   `bun:"id"`
		Title         string  `bun:"title"`
		ImageURL      *string `bun:"image_url"`
		TotalEvents   int32   `bun:"total_events"`
		TotalRegs     int32   `bun:"total_regs"`
		TotalAttended int32   `bun:"total_attended"`
	}
	var results []clubRow
	err := s.db.NewRaw(`
		SELECT o.id, o.title, o.image_url,
			COUNT(DISTINCT e.id) as total_events,
			COUNT(DISTINCT CASE WHEN er.status = 'registered' THEN er.id END) as total_regs,
			COUNT(DISTINCT CASE WHEN er.status = 'registered' AND ea.status IN ('checked_in', 'attended') THEN er.id END) as total_attended
		FROM organizations o
		INNER JOIN events e ON e.organization_id = o.id AND e.start_time >= ?
		LEFT JOIN event_registrations er ON er.event_id = e.id
		LEFT JOIN event_attendance ea ON ea.registration_id = er.id
		GROUP BY o.id, o.title, o.image_url
		ORDER BY total_events DESC, total_regs DESC
		LIMIT ?
	`, startDate, limit).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load top club attendance: %w", err))
	}

	clubs := lo.Map(results, func(r clubRow, _ int) *eventsv1.ClubLeaderboard {
		return &eventsv1.ClubLeaderboard{
			OrganizationId:        r.ID,
			OrganizationTitle:     r.Title,
			OrganizationImage:     r.ImageURL,
			TotalEvents:           r.TotalEvents,
			TotalRegistrations:    r.TotalRegs,
			TotalAttendees:        r.TotalAttended,
			AverageAttendanceRate: safePercentage(r.TotalAttended, r.TotalRegs),
		}
	})

	return connect.NewResponse(&eventsv1.GetTopPerformingClubsResponse{
		Clubs: clubs,
	}), nil
}
