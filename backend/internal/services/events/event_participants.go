package events

import (
	"context"
	"fmt"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// Load participants for the whole page, including accepted collaborating clubs.
func (s *Service) populateEventParticipants(ctx context.Context, events []*eventsv1.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]int32, 0, len(events))
	byID := make(map[int32]*eventsv1.Event, len(events))
	for _, event := range events {
		ids = append(ids, event.Id)
		byID[event.Id] = event
		event.Collaborators = nil
		event.Organizers = nil
	}
	var collaborators []eventCollaboratorRow
	if err := s.db.NewSelect().
		TableExpr("event_collaborators AS ec").
		ColumnExpr("ec.*").
		ColumnExpr("o.title AS organization_title").
		ColumnExpr("o.image_url AS organization_image_url").
		ColumnExpr("os.slug AS organization_slug").
		Join("JOIN organizations o ON o.id = ec.organization_id").
		Join("LEFT JOIN organization_slugs os ON os.organization_id = o.id AND os.is_current = true").
		Where("ec.event_id IN (?)", bun.In(ids)).
		OrderExpr("ec.added_at, ec.id").Scan(ctx, &collaborators); err != nil {
		return fmt.Errorf("load event collaborators: %w", err)
	}
	for _, collaborator := range collaborators {
		event := byID[collaborator.EventID]
		event.Collaborators = append(event.Collaborators, dbEventCollaboratorRowToProto(collaborator))
	}
	var organizers []models.EventOrganizer
	if err := s.db.NewSelect().Model(&organizers).
		Where("event_id IN (?)", bun.In(ids)).OrderExpr("added_at, id").Scan(ctx); err != nil {
		return fmt.Errorf("load event organizers: %w", err)
	}
	for _, organizer := range organizers {
		event := byID[organizer.EventID]
		// Visibility checks need user IDs, without adding personal details to public lists.
		event.Organizers = append(event.Organizers, dbEventOrganizerToProto(organizer, nil, nil))
	}
	return nil
}
