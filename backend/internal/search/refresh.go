package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/util"
	"github.com/uptrace/bun"
)

type EventIndexClient interface {
	IndexEvent(context.Context, *EventDocument) error
	DeleteDocument(context.Context, string, int32) error
}

// The single-bigint advisory key space is separate from the two-int keys used
// for organization member projections. "SEAR" reserves this namespace for search.
const eventSearchLockNamespace int64 = 0x53454152 << 32

func lockEventSearch(ctx context.Context, tx bun.Tx, eventID int32) error {
	key := eventSearchLockNamespace | int64(uint32(eventID))
	if _, err := tx.NewRaw("SELECT pg_advisory_xact_lock(?::bigint)", key).Exec(ctx); err != nil {
		return fmt.Errorf("lock event search refresh: %w", err)
	}
	return nil
}

// RefreshEvent serializes search writes without blocking event mutations. It
// reads current committed data after taking the same lock used by batch refreshes.
func RefreshEvent(ctx context.Context, db *bun.DB, client EventIndexClient, eventID int32) (bool, error) {
	indexed := false
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := lockEventSearch(ctx, tx, eventID); err != nil {
			return err
		}
		var event models.Event
		err := tx.NewSelect().Model(&event).Where("id = ?", eventID).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return client.DeleteDocument(ctx, IndexEvents, eventID)
		}
		if err != nil {
			return fmt.Errorf("load event for indexing: %w", err)
		}
		if event.Status != models.EventStatusPublished || !event.PublishOnMap {
			return client.DeleteDocument(ctx, IndexEvents, eventID)
		}
		var org models.Organization
		if err := tx.NewSelect().Model(&org).Where("id = ?", event.OrganizationID).Scan(ctx); err != nil {
			return fmt.Errorf("load event organization for indexing: %w", err)
		}
		doc := currentEventDocument(event, org.Title)
		if err := client.IndexEvent(ctx, doc); err != nil {
			return err
		}
		indexed = true
		return nil
	})
	return indexed, err
}

func currentEventDocument(event models.Event, organizationTitle string) *EventDocument {
	location, locationType := ExtractLocationFromStructured(event.StructuredLocation)
	doc := &EventDocument{ID: event.ID, Title: event.Title, Description: util.ExtractPlainText(event.Description), Location: location, LocationType: locationType, OrganizationID: event.OrganizationID, OrganizationTitle: organizationTitle, Format: "offline", StartTime: event.StartTime.Format(time.RFC3339), EndTime: event.EndTime.Format(time.RFC3339), CreatedAt: event.CreatedAt.Format(time.RFC3339)}
	if event.Format != nil && *event.Format == models.FormatOnline {
		doc.Format = "online"
	}
	if event.ImageUrl != nil {
		doc.ImageURL = *event.ImageUrl
	}
	return doc
}

type EventBatchIndexClient interface {
	IndexEvents(context.Context, []EventDocument) error
	DeleteDocuments(context.Context, string, []int32) error
}

// RefreshEvents locks unique search IDs in order until both bounded tasks finish,
// including missing IDs, so concurrent refreshes cannot restore a stale document.
// Event mutations remain free to commit while search waits on external tasks.
func RefreshEvents(ctx context.Context, db *bun.DB, client EventBatchIndexClient, eventIDs []int32) (int, error) {
	if len(eventIDs) == 0 {
		return 0, nil
	}
	eventIDs = slices.Clone(eventIDs)
	slices.Sort(eventIDs)
	eventIDs = slices.Compact(eventIDs)
	count := 0
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, eventID := range eventIDs {
			if err := lockEventSearch(ctx, tx, eventID); err != nil {
				return err
			}
		}
		var current []models.Event
		if err := tx.NewSelect().Model(&current).Where("id IN (?)", bun.In(eventIDs)).OrderExpr("id ASC").Scan(ctx); err != nil {
			return fmt.Errorf("load event batch for indexing: %w", err)
		}
		orgIDs := make(map[int32]struct{})
		for _, event := range current {
			if event.Status == models.EventStatusPublished && event.PublishOnMap {
				orgIDs[event.OrganizationID] = struct{}{}
			}
		}
		organizationTitles := make(map[int32]string, len(orgIDs))
		if len(orgIDs) > 0 {
			ids := make([]int32, 0, len(orgIDs))
			for id := range orgIDs {
				ids = append(ids, id)
			}
			var organizations []models.Organization
			if err := tx.NewSelect().Model(&organizations).Column("id", "title").Where("id IN (?)", bun.In(ids)).Scan(ctx); err != nil {
				return fmt.Errorf("load event batch organizations: %w", err)
			}
			for _, org := range organizations {
				organizationTitles[org.ID] = org.Title
			}
		}
		published := make(map[int32]struct{})
		documents := make([]EventDocument, 0, len(current))
		for _, event := range current {
			if event.Status != models.EventStatusPublished || !event.PublishOnMap {
				continue
			}
			title, exists := organizationTitles[event.OrganizationID]
			if !exists {
				return fmt.Errorf("organization %d for event %d not found", event.OrganizationID, event.ID)
			}
			documents = append(documents, *currentEventDocument(event, title))
			published[event.ID] = struct{}{}
		}
		deleted := make([]int32, 0, len(eventIDs)-len(documents))
		for _, id := range eventIDs {
			if _, exists := published[id]; !exists {
				deleted = append(deleted, id)
			}
		}
		if len(documents) > 0 {
			if err := client.IndexEvents(ctx, documents); err != nil {
				return err
			}
		}
		if len(deleted) > 0 {
			if err := client.DeleteDocuments(ctx, IndexEvents, deleted); err != nil {
				return err
			}
		}
		count = len(documents)
		return nil
	})
	return count, err
}
