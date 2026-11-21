//go:build integration

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/activity"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func newTestDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("integration test requires DATABASE_URL")
	}

	sourceDSN, adminDSN, sourceDBName := deriveTestDatabaseDSNs(t, dsn)
	adminSQLDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminDSN)))
	adminDB := bun.NewDB(adminSQLDB, pgdialect.New())
	tempDBName := fmt.Sprintf("domain_event_test_%d", time.Now().UnixNano())

	ctx := context.Background()
	createQuery := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", quoteIdentifier(tempDBName), quoteIdentifier(sourceDBName))
	if _, err := adminDB.ExecContext(ctx, createQuery); err != nil {
		_ = adminDB.Close()
		t.Skipf("cannot create throwaway database (is Postgres running?): %v", err)
	}

	tempDSN := strings.Replace(sourceDSN, "/"+sourceDBName, "/"+tempDBName, 1)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(tempDSN)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Disable FK checks so test data doesn't require real referenced rows.
	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("disable FK triggers: %v", err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE activity_log RESTART IDENTITY CASCADE"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("truncate activity_log: %v", err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE membership_history RESTART IDENTITY CASCADE"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("truncate membership_history: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "SET session_replication_role = 'origin'")
		_ = db.Close()
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
	})
	return db
}

func deriveTestDatabaseDSNs(t *testing.T, dsn string) (sourceDSN string, adminDSN string, dbName string) {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}

	dbName = strings.TrimPrefix(parsed.Path, "/")
	if dbName == "" {
		t.Fatal("DATABASE_URL must include a database name")
	}

	adminURL := *parsed
	adminURL.Path = "/postgres"
	return parsed.String(), adminURL.String(), dbName
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func TestHandleOrganizationCreatedActivityInsertsActivityLog(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	handler := handleOrganizationCreatedActivity(HandlerDeps{DB: db})

	payload, _ := json.Marshal(events.OrganizationCreatedPayload{
		OrganizationID:    9,
		OrganizationTitle: "Robotics Club",
		TypeID:            2,
		ActorID:           7,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(ctx)

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var row models.ActivityLog
	err := db.NewSelect().Model(&row).Where("organization_id = ?", 9).Scan(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if row.Action != activity.ActionOrgCreated {
		t.Fatalf("action = %q, want %q", row.Action, activity.ActionOrgCreated)
	}
	if row.EntityType != "organization" {
		t.Fatalf("entity_type = %q, want %q", row.EntityType, "organization")
	}
	if row.EntityID != 9 {
		t.Fatalf("entity_id = %d, want 9", row.EntityID)
	}
	if row.ActorID == nil || *row.ActorID != 7 {
		t.Fatalf("actor_id = %v, want 7", row.ActorID)
	}
	wantSnap := "Robotics Club"
	if name, ok := row.EntitySnapshot["name"]; !ok || fmt.Sprint(name) != wantSnap {
		t.Fatalf("entity_snapshot[name] = %v, want %q", row.EntitySnapshot["name"], wantSnap)
	}
	if name, ok := row.OrganizationSnapshot["name"]; !ok || fmt.Sprint(name) != wantSnap {
		t.Fatalf("organization_snapshot[name] = %v, want %q", row.OrganizationSnapshot["name"], wantSnap)
	}
	if row.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
}

func TestHandleEventUpdatedActivityCancelledUsesActionCancelled(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	handler := handleEventUpdatedActivity(HandlerDeps{DB: db})

	payload, _ := json.Marshal(events.EventUpdatedPayload{
		EventID:           5,
		Title:             "Cancelled Event",
		OrganizationID:    1,
		OrganizationTitle: "Some Org",
		Status:            "cancelled",
		ActorID:           3,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(ctx)

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var row models.ActivityLog
	err := db.NewSelect().Model(&row).Where("entity_id = ? AND entity_type = ?", 5, "event").Scan(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if row.Action != activity.ActionEventCancelled {
		t.Fatalf("action = %q, want %q", row.Action, activity.ActionEventCancelled)
	}
}

func TestHandleEventUpdatedActivityNonCancelledUsesActionUpdated(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	handler := handleEventUpdatedActivity(HandlerDeps{DB: db})

	payload, _ := json.Marshal(events.EventUpdatedPayload{
		EventID:           6,
		Title:             "Updated Event",
		OrganizationID:    1,
		OrganizationTitle: "Some Org",
		Status:            "published",
		ActorID:           3,
	})
	msg := message.NewMessage("1", payload)
	msg.SetContext(ctx)

	if err := handler(msg); err != nil {
		t.Fatalf("handler() error = %v", err)
	}

	var row models.ActivityLog
	err := db.NewSelect().Model(&row).Where("entity_id = ? AND entity_type = ?", 6, "event").Scan(ctx)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if row.Action != activity.ActionEventUpdated {
		t.Fatalf("action = %q, want %q", row.Action, activity.ActionEventUpdated)
	}
}
