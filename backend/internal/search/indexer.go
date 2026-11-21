package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"github.com/samber/lo"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// structuredLocationJSON represents the JSON structure stored in the database
type structuredLocationJSON struct {
	Type            string          `json:"type"`
	RoomName        *string         `json:"room_name,omitempty"`
	Floor           json.RawMessage `json:"floor,omitempty"`
	ExternalAddress *string         `json:"external_address,omitempty"`
	Latitude        *float64        `json:"latitude,omitempty"`
	Longitude       *float64        `json:"longitude,omitempty"`
	GoogleMapsUrl   *string         `json:"google_maps_url,omitempty"`
	OnlineUrl       *string         `json:"online_url,omitempty"`
	OnlinePlatform  *string         `json:"online_platform,omitempty"`
}

// ExtractLocationFromStructured parses the structured_location JSONB and returns
// a human-readable location string and location type for search indexing
func ExtractLocationFromStructured(data []byte) (location string, locationType string) {
	if len(data) == 0 {
		return "", ""
	}

	var loc structuredLocationJSON
	if err := json.Unmarshal(data, &loc); err != nil {
		slog.Warn("Failed to parse structured_location for search indexing", "error", err)
		return "", ""
	}

	locationType = loc.Type

	switch loc.Type {
	case "campus_room":
		if loc.RoomName != nil && *loc.RoomName != "" {
			location = *loc.RoomName
			// Decode the optional floor separately so an invalid value cannot erase the room.
			var floor string
			if err := json.Unmarshal(loc.Floor, &floor); err != nil {
				var numericFloor int32
				if err := json.Unmarshal(loc.Floor, &numericFloor); err == nil {
					floor = strconv.FormatInt(int64(numericFloor), 10)
				}
			}
			if floor != "" {
				location += ", Floor " + floor
			}
		}
	case "external":
		if loc.ExternalAddress != nil {
			location = *loc.ExternalAddress
		}
	case "online":
		if loc.OnlinePlatform != nil {
			location = *loc.OnlinePlatform
		} else if loc.OnlineUrl != nil {
			location = "Online"
		}
	case "hybrid":
		parts := []string{}
		if loc.RoomName != nil {
			parts = append(parts, *loc.RoomName)
		} else if loc.ExternalAddress != nil {
			parts = append(parts, *loc.ExternalAddress)
		}
		if loc.OnlinePlatform != nil {
			parts = append(parts, *loc.OnlinePlatform)
		} else {
			parts = append(parts, "Online")
		}
		if len(parts) > 0 {
			location = parts[0]
			if len(parts) > 1 {
				location += " + " + parts[1]
			}
		}
	}

	return location, locationType
}

// Indexer handles the indexing of database entities into Meilisearch
type Indexer struct {
	client *Client
	db     *bun.DB
}

// NewIndexer creates a new search indexer
func NewIndexer(client *Client, db *bun.DB) *Indexer {
	return &Indexer{
		client: client,
		db:     db,
	}
}

// ReindexResult contains the results of a reindex operation
type ReindexResult struct {
	EventsIndexed        int
	OrganizationsIndexed int
	UsersIndexed         int
	Errors               []error
}

// ReindexAll reindexes all entities from the database
func (i *Indexer) ReindexAll(ctx context.Context) (*ReindexResult, error) {
	result := &ReindexResult{}

	// Reindex events
	eventsCount, err := i.ReindexEvents(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("events: %w", err))
	}
	result.EventsIndexed = eventsCount

	// Reindex organizations
	orgsCount, err := i.ReindexOrganizations(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("organizations: %w", err))
	}
	result.OrganizationsIndexed = orgsCount

	// Reindex users
	usersCount, err := i.ReindexUsers(ctx)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("users: %w", err))
	}
	result.UsersIndexed = usersCount

	slog.Info("Reindex completed",
		"events", result.EventsIndexed,
		"organizations", result.OrganizationsIndexed,
		"users", result.UsersIndexed,
		"errors", len(result.Errors),
	)

	return result, nil
}

// ReindexEvents reindexes all events
func (i *Indexer) ReindexEvents(ctx context.Context) (int, error) {
	// Include existing indexed IDs to remove deleted/legacy private documents.
	// Collect before deleting so pagination cannot skip shifted index entries.
	indexedIDs, err := i.client.eventDocumentIDs(ctx)
	if err != nil {
		return 0, err
	}
	var databaseIDs []int32
	if err := i.db.NewSelect().Table("events").Column("id").Scan(ctx, &databaseIDs); err != nil {
		return 0, fmt.Errorf("list event IDs: %w", err)
	}
	ids := make(map[int32]struct{}, len(indexedIDs)+len(databaseIDs))
	for _, id := range indexedIDs {
		ids[id] = struct{}{}
	}
	for _, id := range databaseIDs {
		ids[id] = struct{}{}
	}
	orderedIDs := make([]int32, 0, len(ids))
	for id := range ids {
		orderedIDs = append(orderedIDs, id)
	}
	sort.Slice(orderedIDs, func(a, b int) bool { return orderedIDs[a] < orderedIDs[b] })
	const batchSize = 100
	count := 0
	for start := 0; start < len(orderedIDs); start += batchSize {
		batch := orderedIDs[start:min(start+batchSize, len(orderedIDs))]
		indexed, err := RefreshEvents(ctx, i.db, i.client, batch)
		if err != nil {
			return count, fmt.Errorf("refresh event batch starting at %d: %w", batch[0], err)
		}
		count += indexed
	}
	slog.Info("Reindexed events", "count", count)
	return count, nil
}

// ReindexOrganizations reindexes all organizations
func (i *Indexer) ReindexOrganizations(ctx context.Context) (int, error) {
	const batchSize = 100
	var allDocs []OrganizationDocument
	offset := 0

	for {
		var orgs []models.Organization
		err := i.db.NewSelect().Model(&orgs).
			Order("id").
			Limit(batchSize).
			Offset(offset).
			Scan(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list organizations: %w", err)
		}

		if len(orgs) == 0 {
			break
		}

		allDocs = append(allDocs, lo.Map(orgs, func(org models.Organization, _ int) OrganizationDocument {
			status := "active"
			if org.Status != nil {
				status = string(*org.Status)
			}

			description := ""
			if len(org.Description) > 0 {
				description = string(org.Description)
			}

			imageURL := ""
			if org.ImageUrl != nil {
				imageURL = *org.ImageUrl
			}

			return OrganizationDocument{
				ID:                 org.ID,
				Title:              org.Title,
				Description:        description,
				ImageURL:           imageURL,
				OrganizationTypeID: org.OrganizationTypeID,
				Status:             status,
				CreatedAt:          org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		})...)

		offset += len(orgs)
		if len(orgs) < batchSize {
			break
		}
	}

	if len(allDocs) > 0 {
		if err := i.client.IndexOrganizations(ctx, allDocs); err != nil {
			return 0, fmt.Errorf("failed to index organizations: %w", err)
		}
	}

	slog.Info("Reindexed organizations", "count", len(allDocs))
	return len(allDocs), nil
}

// ReindexUsers reindexes all users
func (i *Indexer) ReindexUsers(ctx context.Context) (int, error) {
	const batchSize = 100
	var allDocs []UserDocument
	offset := 0

	for {
		var users []models.User
		err := i.db.NewSelect().Model(&users).
			Order("id").
			Limit(batchSize).
			Offset(offset).
			Scan(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to list users: %w", err)
		}

		if len(users) == 0 {
			break
		}

		allDocs = append(allDocs, lo.Map(users, func(user models.User, _ int) UserDocument {
			return UserDocument{
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		})...)

		offset += len(users)
		if len(users) < batchSize {
			break
		}
	}

	if len(allDocs) > 0 {
		if err := i.client.IndexUsers(ctx, allDocs); err != nil {
			return 0, fmt.Errorf("failed to index users: %w", err)
		}
	}

	slog.Info("Reindexed users", "count", len(allDocs))
	return len(allDocs), nil
}
