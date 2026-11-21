package statistics

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/uptrace/bun"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func (s *Service) GetLowRegistrationEvents(ctx context.Context, req *connect.Request[eventsv1.GetLowRegistrationEventsRequest]) (*connect.Response[eventsv1.GetLowRegistrationEventsResponse], error) {
	if err := s.requirePlatformStaff(ctx); err != nil {
		return nil, err
	}
	slog.Debug("GetLowRegistrationEvents", "threshold", req.Msg.Threshold)

	daysAhead := int(req.Msg.DaysAhead)
	if daysAhead <= 0 {
		daysAhead = 30
	}

	endDate := time.Now().AddDate(0, 0, daysAhead)
	threshold := req.Msg.Threshold
	if threshold <= 0 {
		threshold = 50
	}
	if threshold > 100 {
		threshold = 100
	}

	type eventRow struct {
		ID        int32     `bun:"id"`
		Title     string    `bun:"title"`
		ImageURL  *string   `bun:"image_url"`
		StartTime time.Time `bun:"start_time"`
		OrgID     int32     `bun:"organization_id"`
		Capacity  int32     `bun:"capacity"`
		TotalRegs int32     `bun:"total_regs"`
	}
	var results []eventRow
	err := s.db.NewRaw(`
		WITH internal_counts AS (
			SELECT event_id, COUNT(*)::integer AS count
			FROM event_registrations
			WHERE status = 'registered'
			GROUP BY event_id
		), external_counts AS (
			SELECT event_id, COUNT(*)::integer AS count
			FROM external_registrations
			WHERE status = 'registered'
			GROUP BY event_id
		)
		SELECT e.id, e.title, e.image_url, e.start_time, e.organization_id,
			e.capacity,
			(COALESCE(ic.count, 0) + COALESCE(ec.count, 0))::integer AS total_regs
		FROM events e
		LEFT JOIN internal_counts ic ON ic.event_id = e.id
		LEFT JOIN external_counts ec ON ec.event_id = e.id
		WHERE e.start_time >= NOW() AND e.start_time <= ?
			AND e.capacity IS NOT NULL AND e.capacity > 0
			AND (COALESCE(ic.count, 0) + COALESCE(ec.count, 0)) * 100.0 / e.capacity < ?
		ORDER BY e.start_time
	`, endDate, threshold).Scan(ctx, &results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	organizationIDs := make([]int32, 0, len(results))
	for _, result := range results {
		organizationIDs = append(organizationIDs, result.OrgID)
	}
	organizations := make(map[int32]models.Organization, len(organizationIDs))
	if len(organizationIDs) > 0 {
		var organizationRows []models.Organization
		if orgErr := s.db.NewSelect().Model(&organizationRows).Where("id IN (?)", bun.In(organizationIDs)).Scan(ctx); orgErr != nil {
			return nil, connect.NewError(connect.CodeInternal, orgErr)
		}
		for _, organization := range organizationRows {
			organizations[organization.ID] = organization
		}
	}

	var events []*eventsv1.LowRegistrationEvent
	for _, r := range results {
		daysUntil := int32(time.Until(r.StartTime).Hours() / 24)

		event := &eventsv1.LowRegistrationEvent{
			Id:                  r.ID,
			Title:               r.Title,
			ImageUrl:            r.ImageURL,
			StartTime:           r.StartTime.Format(time.RFC3339),
			Capacity:            r.Capacity,
			TotalRegistrations:  r.TotalRegs,
			CapacityUtilization: safePercentage(r.TotalRegs, r.Capacity),
			DaysUntilEvent:      daysUntil,
		}
		if org, ok := organizations[r.OrgID]; ok {
			event.Organization = modelOrganizationToProto(org)
		}
		events = append(events, event)
	}

	return connect.NewResponse(&eventsv1.GetLowRegistrationEventsResponse{
		Events: events,
	}), nil
}
