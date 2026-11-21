package sundayopen

import (
	"context"
	"fmt"

	pbv1 "github.com/studyverse/ems-backend/gen/sundayopen/v1"
	"github.com/uptrace/bun"
)

type publicHostRow struct {
	EventID              int32   `bun:"event_id"`
	OrganizationID       int32   `bun:"organization_id"`
	OrganizationTitle    string  `bun:"organization_title"`
	OrganizationImageURL *string `bun:"organization_image_url"`
	OrganizationSlug     *string `bun:"organization_slug"`
}

// Match MAP's accepted collaborator order, loading the whole public page at once.
// Invitation records and user organizers are deliberately outside this query.
func (s *Service) populatePublicHosts(ctx context.Context, events []*pbv1.SundayOpenEvent) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]int32, 0, len(events))
	byID := make(map[int32]*pbv1.SundayOpenEvent, len(events))
	seen := make(map[[2]int32]bool)
	for _, event := range events {
		ids = append(ids, event.Id)
		byID[event.Id] = event
		event.Collaborators = nil
		seen[[2]int32{event.Id, event.GetOrganizationId()}] = true
	}
	var rows []publicHostRow
	if err := s.db.NewSelect().TableExpr("event_collaborators AS ec").
		ColumnExpr("ec.event_id, ec.organization_id").
		ColumnExpr("o.title AS organization_title, o.image_url AS organization_image_url, os.slug AS organization_slug").
		Join("JOIN organizations AS o ON o.id = ec.organization_id").
		Join("LEFT JOIN organization_slugs AS os ON os.organization_id = o.id AND os.is_current = true").
		Where("ec.event_id IN (?)", bun.In(ids)).
		OrderExpr("ec.added_at, ec.id").Scan(ctx, &rows); err != nil {
		return fmt.Errorf("load public sunday-open hosts: %w", err)
	}
	for _, row := range rows {
		key := [2]int32{row.EventID, row.OrganizationID}
		if seen[key] {
			continue
		}
		seen[key] = true
		event := byID[row.EventID]
		event.Collaborators = append(event.Collaborators, &pbv1.PublicEventHost{
			OrganizationId:       row.OrganizationID,
			OrganizationTitle:    row.OrganizationTitle,
			OrganizationImageUrl: row.OrganizationImageURL,
			OrganizationSlug:     row.OrganizationSlug,
		})
	}
	return nil
}
