//go:build integration

package organizations

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
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

// capturePublisher is a stub that records the last published topic and message.
type capturePublisher struct {
	topic string
	msg   *message.Message
}

func (p *capturePublisher) Publish(topic string, messages ...*message.Message) error {
	p.topic = topic
	if len(messages) > 0 {
		p.msg = messages[0]
	}
	return nil
}

func (p *capturePublisher) Close() error { return nil }

// newTestDB creates a throwaway Postgres database cloned from the dev template.
// Skips the test when the database is unreachable.
func newTestDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/eventdb?sslmode=disable"
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	sourceDBName := strings.TrimPrefix(parsed.Path, "/")
	if sourceDBName == "" {
		t.Fatal("DATABASE_URL must include a database name")
	}

	adminURL := *parsed
	adminURL.Path = "/postgres"

	adminSQLDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminURL.String())))
	adminDB := bun.NewDB(adminSQLDB, pgdialect.New())
	tempDBName := fmt.Sprintf("orgs_svc_test_%d", time.Now().UnixNano())

	ctx := context.Background()
	createQuery := fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE %s",
		quoteIdentifier(tempDBName), quoteIdentifier(sourceDBName),
	)
	if _, err := adminDB.ExecContext(ctx, createQuery); err != nil {
		_ = adminDB.Close()
		t.Skipf("cannot create throwaway database (is Postgres running?): %v", err)
	}

	tempDSN := strings.Replace(parsed.String(), "/"+sourceDBName, "/"+tempDBName, 1)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(tempDSN)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Disable FK constraints so test inserts don't require every referenced row.
	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("disable FK triggers: %v", err)
	}

	// Truncate tables so tests start clean.
	for _, table := range []string{"organizations", "organization_slugs", "users"} {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", table)); err != nil {
			_ = db.Close()
			_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
			_ = adminDB.Close()
			t.Fatalf("truncate %s: %v", table, err)
		}
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "SET session_replication_role = 'origin'")
		_ = db.Close()
		_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
	})
	return db
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// authedContext returns a context carrying the given kratosID so that
// authz.Enforcer.GetLocalUserID can resolve the caller.
func authedContext(kratosID string) context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, kratosID)
}

// seedAuthUser inserts a user with the given kratosID so the authz enforcer
// can resolve the caller from the DB.  Returns the generated user ID.
func seedAuthUser(t *testing.T, db *bun.DB, kratosID string) int32 {
	t.Helper()
	var id int32
	err := db.NewRaw(
		"INSERT INTO users (kratos_id, username, email, password) VALUES (?, ?, ?, ?) RETURNING id",
		kratosID, "seed-staff", "staff@test.local", "test",
	).Scan(context.Background(), &id)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	return id
}

func TestCreateOrganization_PublishesOrganizationCreatedEvent(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-staff"
	actorID := seedAuthUser(t, db, kratosID)

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t)) // nil perms ⇒ all permission checks pass

	svc := NewService(db, nil, enforcer, nil, pub, nil)
	ctx := authedContext(kratosID)

	resp, err := svc.CreateOrganization(ctx, connect.NewRequest(&eventsv1.CreateOrganizationRequest{
		Title:              "Robotics Club",
		OrganizationTypeId: 1,
		Status:             eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ACTIVE,
	}))
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}

	// The publish must have been called after the DB write + SpiceDB setup.
	if cap.topic != events.TopicOrganizationCreated {
		t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicOrganizationCreated)
	}
	if cap.msg == nil {
		t.Fatal("expected a published message, got nil")
	}

	var got events.OrganizationCreatedPayload
	if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	// The payload must carry the DB-assigned ID, not zero.
	if got.OrganizationID == 0 {
		t.Fatal("expected non-zero OrganizationID in payload")
	}
	if got.OrganizationID != resp.Msg.Organization.Id {
		t.Fatalf("payload OrganizationID = %d, response Id = %d — should match", got.OrganizationID, resp.Msg.Organization.Id)
	}
	if got.OrganizationTitle != "Robotics Club" {
		t.Fatalf("payload OrganizationTitle = %q, want %q", got.OrganizationTitle, "Robotics Club")
	}
	if got.TypeID != 1 {
		t.Fatalf("payload TypeID = %d, want 1", got.TypeID)
	}
	if got.ActorID != actorID {
		t.Fatalf("payload ActorID = %d, want %d (seeded user)", got.ActorID, actorID)
	}
}

func TestCreateOrganization_WithNilPublisherStillSucceeds(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-staff-no-publisher"
	seedAuthUser(t, db, kratosID)

	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, nil)
	ctx := authedContext(kratosID)

	resp, err := svc.CreateOrganization(ctx, connect.NewRequest(&eventsv1.CreateOrganizationRequest{
		Title:              "Nil Publisher Club",
		OrganizationTypeId: 1,
		Status:             eventsv1.OrganizationStatus_ORGANIZATION_STATUS_ACTIVE,
	}))
	if err != nil {
		t.Fatalf("CreateOrganization() with nil publisher error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.Organization == nil {
		t.Fatal("expected created organization response, got nil")
	}
	if resp.Msg.Organization.Id == 0 {
		t.Fatal("expected non-zero created organization id")
	}
	if resp.Msg.Organization.Title != "Nil Publisher Club" {
		t.Fatalf("response Title = %q, want %q", resp.Msg.Organization.Title, "Nil Publisher Club")
	}
}
