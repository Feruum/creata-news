package statistics

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetEventCategoriesDistributionByMonth(ctx context.Context, req *connect.Request[eventsv1.GetEventCategoriesDistributionByMonthRequest]) (*connect.Response[eventsv1.GetEventCategoriesDistributionByMonthResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetEventCategoriesDistributionByMonth", "year", req.Msg.Year, "month", req.Msg.Month)

	location := time.UTC
	if almaty, err := time.LoadLocation("Asia/Almaty"); err == nil {
		location = almaty
	}

	startDate := time.Date(int(req.Msg.Year), time.Month(req.Msg.Month), 1, 0, 0, 0, 0, location)
	endDate := startDate.AddDate(0, 1, 0)

	type catDist struct {
		ID         int32  `bun:"id"`
		Name       string `bun:"name"`
		Color      string `bun:"color"`
		EventCount int32  `bun:"event_count"`
	}
	var results []catDist
	err := s.db.NewRaw(`
		SELECT
			COALESCE(c.id, 0) AS id,
			COALESCE(c.name, 'Uncategorized') AS name,
			COALESCE(c.color, '#6B7280') AS color,
			COUNT(DISTINCT e.id) AS event_count
		FROM events e
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.start_time >= ? AND e.start_time < ?
		GROUP BY
			COALESCE(c.id, 0),
			COALESCE(c.name, 'Uncategorized'),
			COALESCE(c.color, '#6B7280')
		ORDER BY event_count DESC, name ASC
	`, startDate, endDate).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var categories []*eventsv1.CategoryDistribution
	var totalEvents int32
	for _, r := range results {
		categories = append(categories, &eventsv1.CategoryDistribution{
			CategoryId:    r.ID,
			CategoryName:  r.Name,
			CategoryColor: r.Color,
			EventCount:    r.EventCount,
		})
		totalEvents += r.EventCount
	}

	return connect.NewResponse(&eventsv1.GetEventCategoriesDistributionByMonthResponse{
		Categories:  categories,
		TotalEvents: totalEvents,
	}), nil
}
