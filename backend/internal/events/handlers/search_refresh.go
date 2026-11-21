package handlers

import (
	"context"
	"github.com/studyverse/ems-backend/internal/search"
)

func refreshEventSearch(ctx context.Context, deps HandlerDeps, eventID int32) error {
	_, err := search.RefreshEvent(ctx, deps.DB, deps.SearchClient, eventID)
	return err
}
