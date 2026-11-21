//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/activity"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestActivityActorsPersistAndResolveLegacyNames(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	for _, query := range []string{
		`CREATE TABLE users (id integer PRIMARY KEY, username text NOT NULL, first_name text, last_name text, telegram_username text)`,
		`INSERT INTO users VALUES (7,'initiator','Ada','Initiator','ada'), (9,'participant','Pat','Participant',NULL)`,
		`CREATE TYPE activity_action AS ENUM ('participant.registered')`,
		`CREATE TABLE domain_event_inbox (consumer text, message_id text, created_at timestamptz DEFAULT now(), PRIMARY KEY(consumer,message_id))`,
	} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.NewCreateTable().Model((*models.ActivityLog)(nil)).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(events.RegistrationCreatedPayload{EventID: 1, OrganizationID: 2, EventTitle: "Workshop", ActorID: 7, ParticipantUserID: 9})
	if err != nil {
		t.Fatal(err)
	}
	msg := message.NewMessage("actor-snapshot-test", payload)
	handler := handleRegistrationCreatedActivity(HandlerDeps{DB: db})
	for range 2 {
		if err := handler(msg); err != nil {
			t.Fatal(err)
		}
	}
	var rows []models.ActivityLog
	if err := db.NewSelect().Model(&rows).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("duplicate delivery created %d rows", len(rows))
	}
	if rows[0].ActorSnapshot["display_name"] != "Ada Initiator" {
		t.Fatalf("wrong actor: %#v", rows[0].ActorSnapshot)
	}
	if rows[0].ActorSnapshot["telegram_username"] != "ada" {
		t.Fatal("avatar identity missing")
	}

	id := int32(7)
	legacy := &models.ActivityLog{Action: models.ActionParticipantRegistered, ActorID: &id, EntityType: "event", EntityID: 1}
	if _, err := db.NewInsert().Model(legacy).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET first_name = 'Updated' WHERE id = 7`); err != nil {
		t.Fatal(err)
	}
	if err := db.NewSelect().Model(&rows).OrderExpr("id ASC").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	entries := []*models.ActivityLog{&rows[0], &rows[1]}
	if err := activity.FillActorSnapshots(ctx, db, entries); err != nil {
		t.Fatal(err)
	}
	if entries[0].ActorSnapshot["display_name"] != "Ada Initiator" {
		t.Fatal("historical snapshot overwritten")
	}
	if entries[1].ActorSnapshot["display_name"] != "Updated Initiator" {
		t.Fatal("legacy actor not resolved")
	}
	var stored models.ActivityLog
	if err := db.NewSelect().Model(&stored).Where("id = ?", legacy.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(stored.ActorSnapshot) != 0 {
		t.Fatal("read enrichment rewrote audit history")
	}
}
