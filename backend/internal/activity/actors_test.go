package activity

import (
	"github.com/studyverse/ems-backend/internal/models"
	"testing"
)

func TestActorSnapshotsUseInitiatorAndPreserveHistory(t *testing.T) {
	id, other, missing, zero := int32(7), int32(9), int32(404), int32(0)
	first, last, telegram := " Ada ", "Lovelace", "ada"
	entries := []*models.ActivityLog{
		{ActorID: &id, ActorSnapshot: map[string]any{"extra": "keep"}},
		{ActorID: &other},
		{ActorID: &id, ActorSnapshot: map[string]any{"display_name": "Historical name"}},
		{ActorID: &missing}, {}, {ActorID: &zero},
	}
	applyActorSnapshots(entries, []models.User{
		{ID: id, Username: "ada-user", FirstName: &first, LastName: &last, TelegramUsername: &telegram},
		{ID: other, Username: "organizer"},
	})
	if got := entries[0].ActorSnapshot["display_name"]; got != "Ada Lovelace" {
		t.Fatalf("name = %v", got)
	}
	if entries[0].ActorSnapshot["telegram_username"] != "ada" || entries[0].ActorSnapshot["extra"] != "keep" {
		t.Fatal("snapshot fields lost")
	}
	if entries[1].ActorSnapshot["username"] != "organizer" {
		t.Fatal("wrong initiator")
	}
	if entries[2].ActorSnapshot["display_name"] != "Historical name" {
		t.Fatal("historical name overwritten")
	}
	for _, entry := range entries[3:] {
		if len(entry.ActorSnapshot) != 0 {
			t.Fatal("invented identity")
		}
	}
}
