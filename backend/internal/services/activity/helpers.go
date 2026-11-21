package activity

import (
	"encoding/json"
	"time"

	activityv1 "github.com/studyverse/ems-backend/gen/activity/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func toProtoEntries(entries []models.ActivityLog) []*activityv1.ActivityLogEntry {
	result := make([]*activityv1.ActivityLogEntry, len(entries))
	for i, e := range entries {
		result[i] = toProtoEntry(e)
	}
	return result
}

func toProtoEntry(e models.ActivityLog) *activityv1.ActivityLogEntry {
	entry := &activityv1.ActivityLogEntry{
		Id:                   e.ID,
		Action:               string(e.Action),
		ActorSnapshot:        toJSONString(e.ActorSnapshot),
		EntityType:           e.EntityType,
		EntityId:             e.EntityID,
		EntitySnapshot:       toJSONString(e.EntitySnapshot),
		OrganizationSnapshot: toJSONString(e.OrganizationSnapshot),
		Metadata:             toJSONString(e.Metadata),
		CreatedAt:            e.CreatedAt.Format(time.RFC3339),
	}
	if e.ActorID != nil {
		entry.ActorId = e.ActorID
	}
	if e.OrganizationID != nil {
		entry.OrganizationId = e.OrganizationID
	}
	return entry
}

func toJSONString(m map[string]any) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}
