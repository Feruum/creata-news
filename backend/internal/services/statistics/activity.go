package statistics

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetEventActivityByYear(ctx context.Context, req *connect.Request[eventsv1.GetEventActivityByYearRequest]) (*connect.Response[eventsv1.GetEventActivityByYearResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetEventActivityByYear", "year", req.Msg.Year)

	startDate := time.Date(int(req.Msg.Year), 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)

	type activity struct {
		Date  time.Time `bun:"date"`
		Count int32     `bun:"count"`
	}
	var results []activity
	err := s.db.NewRaw(`
		SELECT DATE(start_time) as date, COUNT(*) as count
		FROM events
		WHERE start_time >= ? AND start_time < ?
		GROUP BY DATE(start_time)
		ORDER BY date
	`, startDate, endDate).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var activities []*eventsv1.EventActivity
	var totalEvents int32
	for _, r := range results {
		level := int32(1)
		if r.Count > 5 {
			level = 4
		} else if r.Count > 3 {
			level = 3
		} else if r.Count > 1 {
			level = 2
		}
		activities = append(activities, &eventsv1.EventActivity{
			Date:  r.Date.Format("2006-01-02"),
			Count: r.Count,
			Level: level,
		})
		totalEvents += r.Count
	}

	return connect.NewResponse(&eventsv1.GetEventActivityByYearResponse{
		Activities:  activities,
		TotalEvents: totalEvents,
	}), nil
}
