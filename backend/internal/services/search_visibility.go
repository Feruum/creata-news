package services

import (
	"context"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/studyverse/ems-backend/internal/util"
	"github.com/uptrace/bun"
)

// Search is an eventually consistent candidate index. PostgreSQL determines
// current visibility and returned content, even while an index update retries.
func (s *SearchService) visibleSearchResults(ctx context.Context, candidates []search.SearchResult) ([]search.SearchResult, error) {
	ids := map[string][]int32{}
	for _, result := range candidates {
		ids[result.Type] = append(ids[result.Type], result.ID)
	}
	visible := map[string]map[int32]search.SearchResult{}
	if len(ids[search.IndexEvents]) > 0 {
		var rows []models.Event
		if err := s.db.NewSelect().Model(&rows).Where("id IN (?)", bun.In(ids[search.IndexEvents])).Where("status = ?", models.EventStatusPublished).Where("publish_on_map = TRUE").Scan(ctx); err != nil {
			return nil, err
		}
		visible[search.IndexEvents] = map[int32]search.SearchResult{}
		for _, row := range rows {
			item := search.SearchResult{Type: search.IndexEvents, ID: row.ID, Title: row.Title, Description: util.ExtractPlainText(row.Description)}
			if row.ImageUrl != nil {
				item.ImageURL = *row.ImageUrl
			}
			visible[search.IndexEvents][row.ID] = item
		}
	}
	if len(ids[search.IndexOrganizations]) > 0 {
		var rows []models.Organization
		if err := s.db.NewSelect().Model(&rows).Where("id IN (?)", bun.In(ids[search.IndexOrganizations])).Scan(ctx); err != nil {
			return nil, err
		}
		visible[search.IndexOrganizations] = map[int32]search.SearchResult{}
		for _, row := range rows {
			item := search.SearchResult{Type: search.IndexOrganizations, ID: row.ID, Title: row.Title, Description: util.ExtractPlainText(row.Description)}
			if row.ImageUrl != nil {
				item.ImageURL = *row.ImageUrl
			}
			visible[search.IndexOrganizations][row.ID] = item
		}
	}
	// Email-bearing directory results follow the same policy as ListUsers.
	if len(ids[search.IndexUsers]) > 0 && auth.GetUserID(ctx) != "" && s.authz != nil {
		if _, err := s.authz.RequirePlatformStaff(ctx); err == nil {
			var rows []models.User
			if err := s.db.NewSelect().Model(&rows).Where("id IN (?)", bun.In(ids[search.IndexUsers])).Scan(ctx); err != nil {
				return nil, err
			}
			visible[search.IndexUsers] = map[int32]search.SearchResult{}
			for _, row := range rows {
				visible[search.IndexUsers][row.ID] = search.SearchResult{Type: search.IndexUsers, ID: row.ID, Title: row.Username, Description: row.Email}
			}
		}
	}
	results := make([]search.SearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		if item, ok := visible[candidate.Type][candidate.ID]; ok {
			results = append(results, item)
		}
	}
	return results, nil
}
