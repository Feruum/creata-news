package services

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/meilisearch/meilisearch-go"
	"github.com/samber/lo"
	searchv1 "github.com/studyverse/ems-backend/gen/search/v1"
	"github.com/studyverse/ems-backend/gen/search/v1/searchv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/uptrace/bun"
)

type searchBackend interface {
	GlobalSearch(context.Context, string, int32) (*search.MultiSearchResult, error)
	SearchEvents(context.Context, string, int32, string) (*meilisearch.SearchResponse, error)
}

type SearchService struct {
	searchv1connect.UnimplementedSearchServiceHandler
	searchClient  searchBackend
	searchIndexer *search.Indexer
	db            *bun.DB
	authz         *authz.Enforcer
}

func NewSearchService(searchClient *search.Client, db *bun.DB, authzEnforcer *authz.Enforcer) *SearchService {
	var indexer *search.Indexer
	var backend searchBackend
	if searchClient != nil {
		backend = searchClient
		indexer = search.NewIndexer(searchClient, db)
	}
	return &SearchService{
		searchClient:  backend,
		searchIndexer: indexer,
		db:            db,
		authz:         authzEnforcer,
	}
}

func (s *SearchService) GlobalSearch(ctx context.Context, req *connect.Request[searchv1.GlobalSearchRequest]) (*connect.Response[searchv1.GlobalSearchResponse], error) {
	slog.Debug("GlobalSearch", "query", req.Msg.Query, "limit", req.Msg.Limit)

	if s.searchClient == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("search service not available"))
	}

	if req.Msg.Query == "" {
		return connect.NewResponse(&searchv1.GlobalSearchResponse{
			Results: []*searchv1.SearchResult{},
			Query:   req.Msg.Query,
		}), nil
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}
	limit = min(limit, 200)

	result, err := s.searchClient.GlobalSearch(ctx, req.Msg.Query, limit)
	if err != nil {
		slog.ErrorContext(ctx, "GlobalSearch failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("search failed: %w", err))
	}

	// Convert internal results to proto
	result.Results, err = s.visibleSearchResults(ctx, result.Results)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check search visibility: %w", err))
	}
	protoResults := lo.Map(result.Results, func(resultItem search.SearchResult, _ int) *searchv1.SearchResult {
		resultType := searchv1.SearchResultType_SEARCH_RESULT_TYPE_UNSPECIFIED
		switch resultItem.Type {
		case search.IndexEvents:
			resultType = searchv1.SearchResultType_SEARCH_RESULT_TYPE_EVENT
		case search.IndexOrganizations:
			resultType = searchv1.SearchResultType_SEARCH_RESULT_TYPE_ORGANIZATION
		case search.IndexUsers:
			resultType = searchv1.SearchResultType_SEARCH_RESULT_TYPE_USER
		}

		protoResult := &searchv1.SearchResult{
			Type:  resultType,
			Id:    resultItem.ID,
			Title: resultItem.Title,
		}
		if resultItem.Description != "" {
			protoResult.Description = &resultItem.Description
		}
		if resultItem.ImageURL != "" {
			protoResult.ImageUrl = &resultItem.ImageURL
		}
		return protoResult
	})

	return connect.NewResponse(&searchv1.GlobalSearchResponse{
		Results:          protoResults,
		TotalHits:        int64(len(protoResults)),
		ProcessingTimeMs: result.TimeMs,
		Query:            result.Query,
	}), nil
}

func (s *SearchService) SearchEvents(ctx context.Context, req *connect.Request[searchv1.SearchEventsRequest]) (*connect.Response[searchv1.SearchEventsResponse], error) {
	slog.Debug("SearchEvents", "query", req.Msg.Query, "limit", req.Msg.Limit)

	if s.searchClient == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("search service not available"))
	}

	if req.Msg.Query == "" {
		return connect.NewResponse(&searchv1.SearchEventsResponse{
			Results: []*searchv1.SearchResult{},
		}), nil
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	// Build filters
	limit = min(limit, 200)
	filters := ""
	if req.Msg.OrganizationId != nil {
		filters = fmt.Sprintf("organizationId = %d", *req.Msg.OrganizationId)
	}

	result, err := s.searchClient.SearchEvents(ctx, req.Msg.Query, limit, filters)
	if err != nil {
		slog.ErrorContext(ctx, "SearchEvents failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("search failed: %w", err))
	}

	// Convert hits to proto results
	protoResults := make([]*searchv1.SearchResult, 0, len(result.Hits))
	for _, hit := range result.Hits {
		var hitMap map[string]interface{}
		if err := hit.DecodeInto(&hitMap); err != nil {
			continue
		}

		sr := &searchv1.SearchResult{
			Type: searchv1.SearchResultType_SEARCH_RESULT_TYPE_EVENT,
		}

		if id, ok := hitMap["id"].(float64); ok {
			sr.Id = int32(id)
		}
		if title, ok := hitMap["title"].(string); ok {
			sr.Title = title
		}
		if desc, ok := hitMap["description"].(string); ok {
			sr.Description = &desc
		}
		if imageURL, ok := hitMap["imageUrl"].(string); ok {
			sr.ImageUrl = &imageURL
		}

		protoResults = append(protoResults, sr)
	}

	candidates := make([]search.SearchResult, 0, len(protoResults))
	for _, item := range protoResults {
		candidates = append(candidates, search.SearchResult{Type: search.IndexEvents, ID: item.Id})
	}
	visible, err := s.visibleSearchResults(ctx, candidates)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check search visibility: %w", err))
	}
	protoResults = make([]*searchv1.SearchResult, 0, len(visible))
	for _, item := range visible {
		sr := &searchv1.SearchResult{Type: searchv1.SearchResultType_SEARCH_RESULT_TYPE_EVENT, Id: item.ID, Title: item.Title}
		if item.Description != "" {
			sr.Description = &item.Description
		}
		if item.ImageURL != "" {
			sr.ImageUrl = &item.ImageURL
		}
		protoResults = append(protoResults, sr)
	}
	return connect.NewResponse(&searchv1.SearchEventsResponse{
		Results:   protoResults,
		TotalHits: int64(len(protoResults)),
	}), nil
}

func (s *SearchService) Reindex(ctx context.Context, req *connect.Request[searchv1.ReindexRequest]) (*connect.Response[searchv1.ReindexResponse], error) {
	slog.InfoContext(ctx, "Reindex requested", "indexes", req.Msg.Indexes)

	if _, err := s.authz.RequirePlatformAdmin(ctx); err != nil {
		return nil, err
	}

	if s.searchIndexer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("search indexer not available"))
	}

	result, err := s.searchIndexer.ReindexAll(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Reindex failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("reindex failed: %w", err))
	}

	message := "Reindex completed successfully"
	if len(result.Errors) > 0 {
		message = fmt.Sprintf("Reindex completed with %d errors", len(result.Errors))
	}

	return connect.NewResponse(&searchv1.ReindexResponse{
		Success:              len(result.Errors) == 0,
		Message:              message,
		EventsIndexed:        int32(result.EventsIndexed),
		OrganizationsIndexed: int32(result.OrganizationsIndexed),
		UsersIndexed:         int32(result.UsersIndexed),
	}), nil
}
