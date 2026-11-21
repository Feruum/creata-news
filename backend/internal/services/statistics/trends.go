package statistics

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetEventTrends(ctx context.Context, req *connect.Request[eventsv1.GetEventTrendsRequest]) (*connect.Response[eventsv1.GetEventTrendsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetEventTrends", "days", req.Msg.Days)

	days := int(req.Msg.Days)
	if days <= 0 {
		days = 90
	}

	startDate := time.Now().AddDate(0, 0, -days)

	type trendRow struct {
		Date       time.Time `bun:"date"`
		EventCount int32     `bun:"event_count"`
		RegCount   int32     `bun:"reg_count"`
	}
	var results []trendRow
	err := s.db.NewRaw(`
		SELECT DATE(e.start_time) as date,
			COUNT(DISTINCT e.id) as event_count,
			COUNT(DISTINCT er.id) as reg_count
		FROM events e
		LEFT JOIN event_registrations er ON er.event_id = e.id AND er.status = 'registered'
		WHERE e.start_time >= ? AND e.start_time <= NOW()
		GROUP BY DATE(e.start_time)
		ORDER BY date
	`, startDate).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	trends := lo.Map(results, func(r trendRow, _ int) *eventsv1.EventTrend {
		return &eventsv1.EventTrend{
			Date:              r.Date.Format("2006-01-02"),
			EventCount:        r.EventCount,
			RegistrationCount: r.RegCount,
		}
	})

	return connect.NewResponse(&eventsv1.GetEventTrendsResponse{
		Trends: trends,
	}), nil
}
