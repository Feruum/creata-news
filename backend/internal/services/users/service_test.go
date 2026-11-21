//go:build integration

package users

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
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
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
// Skips the test when DATABASE_URL is unset (CI without Postgres).
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
	tempDBName := fmt.Sprintf("users_svc_test_%d", time.Now().UnixNano())

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

	// Truncate users table so tests start clean.
	if _, err := db.ExecContext(ctx, "TRUNCATE users RESTART IDENTITY CASCADE"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("truncate users: %v", err)
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
		kratosID, "seed-admin", "seed@test.local", "test",
	).Scan(context.Background(), &id)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	return id
}

func TestCreateUser_PublishesUserCreatedEvent(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-admin"
	seedAuthUser(t, db, kratosID)

	cap := &capturePublisher{}
	pub := events.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t)) // nil perms ⇒ all permission checks pass

	svc := NewService(db, nil, enforcer, nil, pub, "", "")
	ctx := authedContext(kratosID)

	resp, err := svc.CreateUser(ctx, connect.NewRequest(&usersv1.CreateUserRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "secret123",
	}))
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// The publish must have been called after the DB write.
	if cap.topic != events.TopicUserCreated {
		t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicUserCreated)
	}
	if cap.msg == nil {
		t.Fatal("expected a published message, got nil")
	}

	var got events.UserCreatedPayload
	if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	// The payload must carry the DB-assigned ID, not zero.
	if got.UserID == 0 {
		t.Fatal("expected non-zero UserID in payload")
	}
	if got.UserID != resp.Msg.User.Id {
		t.Fatalf("payload UserID = %d, response User.Id = %d — should match", got.UserID, resp.Msg.User.Id)
	}
	if got.Username != "newuser" {
		t.Fatalf("payload Username = %q, want %q", got.Username, "newuser")
	}
	if got.Email != "new@example.com" {
		t.Fatalf("payload Email = %q, want %q", got.Email, "new@example.com")
	}
}

func TestCreateUser_WithNilPublisherStillSucceeds(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-admin-no-publisher"
	seedAuthUser(t, db, kratosID)

	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, nil, "", "")
	ctx := authedContext(kratosID)

	resp, err := svc.CreateUser(ctx, connect.NewRequest(&usersv1.CreateUserRequest{
		Username: "nilpublisher",
		Email:    "nilpublisher@example.com",
		Password: "secret123",
	}))
	if err != nil {
		t.Fatalf("CreateUser() with nil publisher error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.User == nil {
		t.Fatal("expected created user response, got nil")
	}
	if resp.Msg.User.Id == 0 {
		t.Fatal("expected non-zero created user id")
	}
	if resp.Msg.User.Username != "nilpublisher" {
		t.Fatalf("response Username = %q, want %q", resp.Msg.User.Username, "nilpublisher")
	}
}
