package activity

import (
	"context"
	"fmt"
	"strings"

	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

// FillActorSnapshots resolves missing names from the recorded initiator ID, never
// from the event owner or participant. Existing snapshots remain historical.
// Callers can use this on reads for legacy rows without rewriting the audit log.
func FillActorSnapshots(ctx context.Context, db bun.IDB, entries []*models.ActivityLog) error {
	ids := make(map[int32]struct{})
	for _, entry := range entries {
		if entry.ActorID != nil && *entry.ActorID > 0 && !hasActorName(entry.ActorSnapshot) {
			ids[*entry.ActorID] = struct{}{}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	userIDs := make([]int32, 0, len(ids))
	for id := range ids {
		userIDs = append(userIDs, id)
	}
	var users []models.User
	if err := db.NewSelect().Model(&users).Column("id", "username", "first_name", "last_name", "telegram_username").Where("id IN (?)", bun.In(userIDs)).Scan(ctx); err != nil {
		return fmt.Errorf("resolve activity actors: %w", err)
	}
	applyActorSnapshots(entries, users)
	return nil
}

func hasActorName(snapshot map[string]any) bool {
	for _, key := range []string{"display_name", "username"} {
		if value, ok := snapshot[key].(string); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func applyActorSnapshots(entries []*models.ActivityLog, users []models.User) {
	byID := make(map[int32]models.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}
	for _, entry := range entries {
		if entry.ActorID == nil || *entry.ActorID <= 0 || hasActorName(entry.ActorSnapshot) {
			continue
		}
		user, ok := byID[*entry.ActorID]
		if !ok {
			continue
		}
		snapshot := make(map[string]any, len(entry.ActorSnapshot)+3)
		for key, value := range entry.ActorSnapshot {
			snapshot[key] = value
		}
		parts := []string{}
		for _, name := range []*string{user.FirstName, user.LastName} {
			if name != nil && strings.TrimSpace(*name) != "" {
				parts = append(parts, strings.TrimSpace(*name))
			}
		}
		snapshot["display_name"] = strings.Join(parts, " ")
		snapshot["username"] = user.Username
		if user.TelegramUsername != nil {
			snapshot["telegram_username"] = *user.TelegramUsername
		}
		entry.ActorSnapshot = snapshot
	}
}
