package events

import (
	"context"
	"fmt"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/uptrace/bun"
)

// Load active registration and attendance counts for a page in one query.
func (s *Service) populateEventCounts(ctx context.Context, events []*eventsv1.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]int32, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.Id)
	}
	var rows []struct {
		EventID               int32 `bun:"event_id"`
		Registrations         int32 `bun:"registrations"`
		Attendees             int32 `bun:"attendees"`
		ExternalRegistrations int32 `bun:"external_registrations"`
	}
	if err := s.db.NewRaw(`
		SELECT e.id AS event_id,
		       COUNT(DISTINCT er.id) FILTER (WHERE er.status = 'registered') AS registrations,
		       COUNT(DISTINCT er.id) FILTER (
		           WHERE er.status = 'registered' AND ea.status IN ('checked_in', 'attended')
		       ) AS attendees,
		       (SELECT COUNT(*) FROM external_registrations ex WHERE ex.event_id = e.id AND ex.status = 'registered') AS external_registrations
		FROM events e
		LEFT JOIN event_registrations er ON er.event_id = e.id
		LEFT JOIN event_attendance ea ON ea.registration_id = er.id
		WHERE e.id IN (?)
		GROUP BY e.id
	`, bun.In(ids)).Scan(ctx, &rows); err != nil {
		return fmt.Errorf("load event counts: %w", err)
	}
	byID := make(map[int32][3]int32, len(rows))
	for _, row := range rows {
		byID[row.EventID] = [3]int32{row.Registrations, row.Attendees, row.ExternalRegistrations}
	}
	for _, event := range events {
		counts := byID[event.Id]
		event.TotalRegistrations, event.TotalAttendees = counts[0], counts[1]
		event.ExternalRegistrationCount = counts[2]
		event.AllocatedSeatCount = counts[0] + counts[2]
	}
	return nil
}
