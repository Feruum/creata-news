package events

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

func assignPublicationTargetsFromHosts(ctx context.Context, database bun.IDB, event *models.Event) error {
	// Slugs stay unique and retain their organization after a rename, so the
	// historical Sunday Open slug identifies the same primary or accepted co-host.
	// An accepted invitation alone is insufficient after its collaborator is removed.
	// Its expiry only limits acceptance, never the duration of accepted consent.
	onSundayOpen, err := database.NewSelect().TableExpr("organization_slugs AS sunday").
		Where("sunday.slug = ?", "sunday-open").
		Where(`sunday.organization_id = ? OR EXISTS (
			SELECT 1 FROM event_collaborators AS ec
			JOIN event_collaboration_invites AS i
				ON i.event_id = ec.event_id AND i.organization_id = ec.organization_id
			WHERE ec.event_id = ? AND ec.organization_id = sunday.organization_id AND i.status = ?
		)`, event.OrganizationID, event.ID, models.CollaborationInviteStatusAccepted).
		Exists(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to determine hosts' publication platforms: %w", err))
	}
	event.PublishOnMap = true
	event.PublishOnSundayOpen = onSundayOpen
	return nil
}

func organizationIsSundayOpen(ctx context.Context, database bun.IDB, organizationID int32) (bool, error) {
	eligible, err := database.NewSelect().Model((*models.OrganizationSlug)(nil)).
		Where("organization_id = ?", organizationID).
		Where("slug = ?", "sunday-open").Exists(ctx)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to identify Sunday Open organization: %w", err))
	}
	return eligible, nil
}

func syncCollaborationPublication(ctx context.Context, database bun.IDB, event *models.Event, organizationID int32) error {
	isSundayOpen, err := organizationIsSundayOpen(ctx, database, organizationID)
	if err != nil || !isSundayOpen {
		return err
	}
	if err := assignPublicationTargetsFromHosts(ctx, database, event); err != nil {
		return err
	}
	if _, err := database.NewUpdate().Model(event).
		Column("publish_on_map", "publish_on_sunday_open").WherePK().Exec(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update collaboration publication: %w", err))
	}
	return nil
}
