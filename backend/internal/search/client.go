package search

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/samber/lo"
	"github.com/studyverse/ems-backend/internal/telemetry"
)

// Index names
const (
	IndexEvents        = "events"
	IndexOrganizations = "organizations"
	IndexUsers         = "users"
)

// Default wait interval for Meilisearch tasks
const defaultWaitInterval = 50 * time.Millisecond

// Client wraps the Meilisearch client with domain-specific methods
type Client struct {
	meili meilisearch.ServiceManager
}

// NewClient creates a new Meilisearch client wrapper with retry logic
func NewClient(url, masterKey string) (*Client, error) {
	client := meilisearch.New(url, meilisearch.WithAPIKey(masterKey), meilisearch.WithCustomClient(telemetry.NewHTTPClient()))

	// Verify connection by checking health with retries
	// This handles cases where Meilisearch may still be starting up
	maxRetries := 5
	retryDelay := 2 * time.Second
	var healthy bool

	for i := 0; i < maxRetries; i++ {
		if client.IsHealthy() {
			healthy = true
			break
		}
		if i < maxRetries-1 {
			slog.Debug("Meilisearch not healthy, retrying...", "attempt", i+1, "maxRetries", maxRetries)
			time.Sleep(retryDelay)
		}
	}

	if !healthy {
		return nil, fmt.Errorf("meilisearch is not healthy at %s after %d attempts", url, maxRetries)
	}

	c := &Client{meili: client}

	// Initialize indexes with proper settings
	if err := c.initializeIndexes(); err != nil {
		return nil, fmt.Errorf("failed to initialize indexes: %w", err)
	}

	return c, nil
}

// initializeIndexes creates indexes and configures their settings
func (c *Client) initializeIndexes() error {
	indexes := []struct {
		name       string
		primaryKey string
		searchable []string
		filterable []string
		sortable   []string
	}{
		{
			name:       IndexEvents,
			primaryKey: "id",
			searchable: []string{"title", "description", "location", "organizationTitle"},
			filterable: []string{"organizationId", "format", "startTime"},
			sortable:   []string{"startTime", "createdAt", "title"},
		},
		{
			name:       IndexOrganizations,
			primaryKey: "id",
			searchable: []string{"title", "description"},
			filterable: []string{"organizationTypeId", "status"},
			sortable:   []string{"title", "createdAt"},
		},
		{
			name:       IndexUsers,
			primaryKey: "id",
			searchable: []string{"username", "email", "firstName", "lastName"},
			filterable: []string{"role"},
			sortable:   []string{"username", "createdAt"},
		},
	}

	for _, idx := range indexes {
		// Create or get index
		task, err := c.meili.CreateIndex(&meilisearch.IndexConfig{
			Uid:        idx.name,
			PrimaryKey: idx.primaryKey,
		})
		if err != nil {
			// Index might already exist, try to get it
			slog.Debug("Index creation returned error (may already exist)", "index", idx.name, "error", err)
		} else {
			// Wait for index creation
			_, err = c.meili.WaitForTask(task.TaskUID, defaultWaitInterval)
			if err != nil {
				slog.Warn("Failed to wait for index creation", "index", idx.name, "error", err)
			}
		}

		index := c.meili.Index(idx.name)

		// Update searchable attributes
		if len(idx.searchable) > 0 {
			task, err := index.UpdateSearchableAttributes(&idx.searchable)
			if err != nil {
				slog.Warn("Failed to update searchable attributes", "index", idx.name, "error", err)
			} else {
				_, _ = c.meili.WaitForTask(task.TaskUID, defaultWaitInterval)
			}
		}

		// Update filterable attributes
		if len(idx.filterable) > 0 {
			filterableAttrs := lo.Map(idx.filterable, func(attr string, _ int) interface{} {
				return attr
			})
			task, err := index.UpdateFilterableAttributes(&filterableAttrs)
			if err != nil {
				slog.Warn("Failed to update filterable attributes", "index", idx.name, "error", err)
			} else {
				_, _ = c.meili.WaitForTask(task.TaskUID, defaultWaitInterval)
			}
		}

		// Update sortable attributes
		if len(idx.sortable) > 0 {
			task, err := index.UpdateSortableAttributes(&idx.sortable)
			if err != nil {
				slog.Warn("Failed to update sortable attributes", "index", idx.name, "error", err)
			} else {
				_, _ = c.meili.WaitForTask(task.TaskUID, defaultWaitInterval)
			}
		}

		slog.Info("Initialized search index", "index", idx.name)
	}

	return nil
}

// EventDocument represents an event in the search index
type EventDocument struct {
	ID                int32  `json:"id"`
	Title             string `json:"title"`
	Description       string `json:"description"`
	Location          string `json:"location"`
	LocationType      string `json:"locationType,omitempty"`
	ImageURL          string `json:"imageUrl,omitempty"`
	OrganizationID    int32  `json:"organizationId"`
	OrganizationTitle string `json:"organizationTitle"`
	Format            string `json:"format"`
	StartTime         string `json:"startTime"`
	EndTime           string `json:"endTime"`
	CreatedAt         string `json:"createdAt"`
}

// OrganizationDocument represents an organization in the search index
type OrganizationDocument struct {
	ID                 int32  `json:"id"`
	Title              string `json:"title"`
	Description        string `json:"description,omitempty"`
	ImageURL           string `json:"imageUrl,omitempty"`
	OrganizationTypeID int32  `json:"organizationTypeId"`
	Status             string `json:"status"`
	CreatedAt          string `json:"createdAt"`
}

// UserDocument represents a user in the search index
type UserDocument struct {
	ID        int32  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// primaryKeyOption creates a DocumentOptions with primary key set to "id"
func primaryKeyOption() *meilisearch.DocumentOptions {
	pk := "id"
	return &meilisearch.DocumentOptions{PrimaryKey: &pk}
}

// IndexEvent adds or updates an event in the search index
func (c *Client) IndexEvent(ctx context.Context, doc *EventDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	task, err := c.meili.Index(IndexEvents).AddDocumentsWithContext(ctx, []EventDocument{*doc}, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index event: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// IndexEvents adds or updates multiple events in the search index
func (c *Client) IndexEvents(ctx context.Context, docs []EventDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if len(docs) == 0 {
		return nil
	}
	task, err := c.meili.Index(IndexEvents).AddDocumentsWithContext(ctx, docs, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index events: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// IndexOrganization adds or updates an organization in the search index
func (c *Client) IndexOrganization(ctx context.Context, doc *OrganizationDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	task, err := c.meili.Index(IndexOrganizations).AddDocumentsWithContext(ctx, []OrganizationDocument{*doc}, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index organization: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// IndexOrganizations adds or updates multiple organizations in the search index
func (c *Client) IndexOrganizations(ctx context.Context, docs []OrganizationDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if len(docs) == 0 {
		return nil
	}
	task, err := c.meili.Index(IndexOrganizations).AddDocumentsWithContext(ctx, docs, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index organizations: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// IndexUser adds or updates a user in the search index
func (c *Client) IndexUser(ctx context.Context, doc *UserDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	task, err := c.meili.Index(IndexUsers).AddDocumentsWithContext(ctx, []UserDocument{*doc}, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index user: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// IndexUsers adds or updates multiple users in the search index
func (c *Client) IndexUsers(ctx context.Context, docs []UserDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if len(docs) == 0 {
		return nil
	}
	task, err := c.meili.Index(IndexUsers).AddDocumentsWithContext(ctx, docs, primaryKeyOption())
	if err != nil {
		return fmt.Errorf("failed to index users: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// DeleteDocument removes a document from an index
func (c *Client) DeleteDocument(ctx context.Context, indexName string, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	task, err := c.meili.Index(indexName).DeleteDocumentWithContext(ctx, fmt.Sprintf("%d", id), nil)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}

// SearchResult represents a single search result
type SearchResult struct {
	Type        string `json:"type"`
	ID          int32  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"imageUrl,omitempty"`
}

// MultiSearchResult contains results from multiple indexes
type MultiSearchResult struct {
	Results   []SearchResult `json:"results"`
	TotalHits int64          `json:"totalHits"`
	Query     string         `json:"query"`
	TimeMs    int64          `json:"timeMs"`
}

// GlobalSearch performs a search across all indexes
func (c *Client) GlobalSearch(ctx context.Context, query string, limit int32) (*MultiSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Perform multi-search across all indexes
	multiSearchReq := &meilisearch.MultiSearchRequest{
		Queries: []*meilisearch.SearchRequest{
			{
				IndexUID: IndexEvents,
				Query:    query,
				Limit:    int64(limit),
			},
			{
				IndexUID: IndexOrganizations,
				Query:    query,
				Limit:    int64(limit),
			},
			{
				IndexUID: IndexUsers,
				Query:    query,
				Limit:    int64(limit / 2), // Fewer user results
			},
		},
	}

	resp, err := c.meili.MultiSearchWithContext(ctx, multiSearchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to perform multi-search: %w", err)
	}

	result := &MultiSearchResult{
		Query:   query,
		Results: make([]SearchResult, 0),
	}

	for _, searchResp := range resp.Results {
		indexUID := searchResp.IndexUID
		result.TotalHits += searchResp.EstimatedTotalHits
		result.TimeMs += searchResp.ProcessingTimeMs

		for _, hit := range searchResp.Hits {
			// Decode hit into a generic map
			var hitMap map[string]interface{}
			if err := hit.DecodeInto(&hitMap); err != nil {
				slog.Debug("Failed to decode hit", "error", err)
				continue
			}

			sr := SearchResult{
				Type: indexUID,
			}

			// Extract common fields
			if id, ok := hitMap["id"].(float64); ok {
				sr.ID = int32(id)
			}
			if imageURL, ok := hitMap["imageUrl"].(string); ok {
				sr.ImageURL = imageURL
			}

			// Extract type-specific fields
			switch indexUID {
			case IndexEvents:
				if title, ok := hitMap["title"].(string); ok {
					sr.Title = title
				}
				if desc, ok := hitMap["description"].(string); ok {
					sr.Description = desc
				}
			case IndexOrganizations:
				if title, ok := hitMap["title"].(string); ok {
					sr.Title = title
				}
				if desc, ok := hitMap["description"].(string); ok {
					sr.Description = desc
				}
			case IndexUsers:
				if username, ok := hitMap["username"].(string); ok {
					sr.Title = username
				}
				if email, ok := hitMap["email"].(string); ok {
					sr.Description = email
				}
			}

			result.Results = append(result.Results, sr)
		}
	}

	return result, nil
}

// SearchEvents searches only the events index
func (c *Client) SearchEvents(ctx context.Context, query string, limit int32, filters string) (*meilisearch.SearchResponse, error) {
	req := &meilisearch.SearchRequest{
		Query: query,
		Limit: int64(limit),
	}
	if filters != "" {
		req.Filter = filters
	}
	return c.meili.Index(IndexEvents).SearchWithContext(ctx, query, req)
}

// SearchOrganizations searches only the organizations index
func (c *Client) SearchOrganizations(ctx context.Context, query string, limit int32) (*meilisearch.SearchResponse, error) {
	return c.meili.Index(IndexOrganizations).SearchWithContext(ctx, query, &meilisearch.SearchRequest{
		Limit: int64(limit),
	})
}

// The SDK returns failed/canceled terminal tasks with no transport error.
// Such writes must fail the consumer so the original message remains retryable.
func (c *Client) waitForSuccessfulTask(ctx context.Context, uid int64) error {
	task, err := c.meili.WaitForTaskWithContext(ctx, uid, defaultWaitInterval)
	if err != nil {
		return fmt.Errorf("wait for search task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("search task %d returned no result", uid)
	}
	if task.Status != "succeeded" {
		return fmt.Errorf("search task %d ended %s (%s)", uid, task.Status, task.Error.Code)
	}
	return nil
}

func (c *Client) eventDocumentIDs(ctx context.Context) ([]int32, error) {
	var ids []int32
	for offset := int64(0); ; {
		var page meilisearch.DocumentsResult
		err := c.meili.Index(IndexEvents).GetDocumentsWithContext(ctx, &meilisearch.DocumentsQuery{Offset: offset, Limit: 1000, Fields: []string{"id"}}, &page)
		if err != nil {
			return nil, fmt.Errorf("list indexed events: %w", err)
		}
		for _, hit := range page.Results {
			var item struct {
				ID int32 `json:"id"`
			}
			if err := hit.DecodeInto(&item); err != nil {
				return nil, fmt.Errorf("decode indexed event ID: %w", err)
			}
			ids = append(ids, item.ID)
		}
		offset += int64(len(page.Results))
		if len(page.Results) == 0 || offset >= page.Total {
			break
		}
	}
	return ids, nil
}

// DeleteDocuments removes a bounded group in one asynchronous task.
func (c *Client) DeleteDocuments(ctx context.Context, indexName string, ids []int32) error {
	if len(ids) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	identifiers := make([]string, len(ids))
	for index, id := range ids {
		identifiers[index] = fmt.Sprint(id)
	}
	task, err := c.meili.Index(indexName).DeleteDocumentsWithContext(ctx, identifiers, nil)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}
	return c.waitForSuccessfulTask(ctx, task.TaskUID)
}
