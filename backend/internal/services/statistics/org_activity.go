package statistics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"github.com/uptrace/bun"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetOrganizationActivity(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationActivityRequest]) (*connect.Response[eventsv1.GetOrganizationActivityResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetOrganizationActivity", "limit", req.Msg.Limit)

	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 20
	}

	now := time.Now()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)

	type orgRow struct {
		ID              int32   `bun:"id"`
		Title           string  `bun:"title"`
		ImageURL        *string `bun:"image_url"`
		EventsThisMonth int32   `bun:"events_this_month"`
		EventsLastMonth int32   `bun:"events_last_month"`
		TotalEvents     int32   `bun:"total_events"`
		AvgAttendance   float64 `bun:"avg_attendance"`
	}
	var results []orgRow
	err := s.db.NewRaw(`
		SELECT o.id, o.title, o.image_url,
			COUNT(DISTINCT CASE WHEN e.start_time >= ? THEN e.id END) as events_this_month,
			COUNT(DISTINCT CASE WHEN e.start_time >= ? AND e.start_time < ? THEN e.id END) as events_last_month,
			COUNT(DISTINCT e.id) as total_events,
			COALESCE(
				CASE WHEN COUNT(DISTINCT e.id) > 0 
				THEN COUNT(DISTINCT CASE WHEN er.status = 'registered' AND ea.status IN ('checked_in', 'attended') THEN er.id END)::float / COUNT(DISTINCT e.id)::float
				ELSE 0 END,
				0
			) as avg_attendance
		FROM organizations o
		LEFT JOIN events e ON e.organization_id = o.id
		LEFT JOIN event_registrations er ON er.event_id = e.id
		LEFT JOIN event_attendance ea ON ea.registration_id = er.id
		GROUP BY o.id, o.title, o.image_url
		ORDER BY events_this_month DESC, total_events DESC
		LIMIT ?
	`, thisMonthStart, lastMonthStart, thisMonthStart, limit).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load organization attendance: %w", err))
	}

	orgs := lo.Map(results, func(r orgRow, _ int) *eventsv1.OrganizationActivity {
		var growthRate float64
		if r.EventsLastMonth > 0 {
			growthRate = (float64(r.EventsThisMonth) - float64(r.EventsLastMonth)) / float64(r.EventsLastMonth) * 100
		} else if r.EventsThisMonth > 0 {
			growthRate = 100
		}

		return &eventsv1.OrganizationActivity{
			Id:                r.ID,
			Title:             r.Title,
			ImageUrl:          r.ImageURL,
			EventsThisMonth:   r.EventsThisMonth,
			EventsLastMonth:   r.EventsLastMonth,
			TotalEvents:       r.TotalEvents,
			AverageAttendance: r.AvgAttendance,
			GrowthRate:        growthRate,
		}
	})

	return connect.NewResponse(&eventsv1.GetOrganizationActivityResponse{
		Organizations: orgs,
	}), nil
}

func (s *Service) GetOrganizationTrends(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationTrendsRequest]) (*connect.Response[eventsv1.GetOrganizationTrendsResponse], error) {
	if err := s.requireOrganizationAnalytics(ctx, req.Msg.OrganizationIds); err != nil {
		return nil, err
	}
	slog.Debug("GetOrganizationTrends", "organization_ids", req.Msg.OrganizationIds, "days", req.Msg.Days)

	days := int(req.Msg.Days)
	if days <= 0 {
		days = 90
	}

	orgIDs := req.Msg.OrganizationIds
	if len(orgIDs) == 0 {
		return connect.NewResponse(&eventsv1.GetOrganizationTrendsResponse{
			Trends: []*eventsv1.OrganizationTrend{},
		}), nil
	}

	startDate := time.Now().AddDate(0, 0, -days)

	type trendRow struct {
		OrgID      int32     `bun:"organization_id"`
		WeekStart  time.Time `bun:"week_start"`
		EventCount int32     `bun:"event_count"`
	}
	var results []trendRow
	err := s.db.NewRaw(`
		SELECT organization_id, 
			DATE_TRUNC('week', start_time)::date as week_start, 
			COUNT(*) as event_count
		FROM events 
		WHERE organization_id IN (?) AND start_time >= ?
		GROUP BY organization_id, DATE_TRUNC('week', start_time)
		ORDER BY organization_id, week_start
	`, bun.In(orgIDs), startDate).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	trendsByOrg := make(map[int32][]*eventsv1.TrendPoint)
	for _, r := range results {
		trendsByOrg[r.OrgID] = append(trendsByOrg[r.OrgID], &eventsv1.TrendPoint{
			Date:  r.WeekStart.Format("2006-01-02"),
			Value: r.EventCount,
		})
	}

	trends := lo.Map(orgIDs, func(orgID int32, _ int) *eventsv1.OrganizationTrend {
		points := trendsByOrg[orgID]
		if points == nil {
			points = []*eventsv1.TrendPoint{}
		}
		return &eventsv1.OrganizationTrend{
			OrganizationId: orgID,
			Points:         points,
		}
	})

	return connect.NewResponse(&eventsv1.GetOrganizationTrendsResponse{
		Trends: trends,
	}), nil
}
