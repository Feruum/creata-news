package statistics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetTopPerformingEvents(ctx context.Context, req *connect.Request[eventsv1.GetTopPerformingEventsRequest]) (*connect.Response[eventsv1.GetTopPerformingEventsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetTopPerformingEvents", "limit", req.Msg.Limit)

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	}
	days := int(req.Msg.Days)
	if days <= 0 {
		days = 90
	}

	startDate := time.Now().AddDate(0, 0, -days)

	type eventRow struct {
		ID            int32     `bun:"id"`
		Title         string    `bun:"title"`
		ImageURL      *string   `bun:"image_url"`
		StartTime     time.Time `bun:"start_time"`
		OrgID         int32     `bun:"organization_id"`
		TotalRegs     int32     `bun:"total_regs"`
		TotalAttended int32     `bun:"total_attended"`
	}
	var results []eventRow
	err := s.db.NewRaw(`
		SELECT e.id, e.title, e.image_url, e.start_time, e.organization_id,
			COUNT(DISTINCT CASE WHEN er.status = 'registered' THEN er.id END) as total_regs,
			COUNT(DISTINCT CASE WHEN er.status = 'registered' AND ea.status IN ('checked_in', 'attended') THEN er.id END) as total_attended
		FROM events e
		LEFT JOIN event_registrations er ON er.event_id = e.id
		LEFT JOIN event_attendance ea ON ea.registration_id = er.id
		WHERE e.start_time >= ?
		GROUP BY e.id, e.title, e.image_url, e.start_time, e.organization_id
		ORDER BY total_regs DESC, total_attended DESC
		LIMIT ?
	`, startDate, limit).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load top event attendance: %w", err))
	}

	var events []*eventsv1.TopPerformingEvent
	for _, r := range results {
		var org models.Organization
		orgErr := s.db.NewSelect().Model(&org).Where("id = ?", r.OrgID).Scan(ctx)

		event := &eventsv1.TopPerformingEvent{
			Id:                 r.ID,
			Title:              r.Title,
			ImageUrl:           r.ImageURL,
			StartTime:          r.StartTime.Format(time.RFC3339),
			TotalRegistrations: r.TotalRegs,
			TotalAttendees:     r.TotalAttended,
			AttendanceRate:     safePercentage(r.TotalAttended, r.TotalRegs),
		}
		if orgErr == nil && org.ID != 0 {
			event.Organization = modelOrganizationToProto(org)
		}
		events = append(events, event)
	}

	return connect.NewResponse(&eventsv1.GetTopPerformingEventsResponse{
		Events: events,
	}), nil
}
