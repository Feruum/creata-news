//go:build integration

package services

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

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestGetUserRegistrations_RequiresPlatformStaff(t *testing.T) {
	svc := &EventRegistrationsService{authz: authz.NewEnforcer(nil, nil)}
	_, err := svc.GetUserRegistrations(context.Background(), connect.NewRequest(&eventsv1.GetUserRegistrationsRequest{UserId: 1}))
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated && code != connect.CodePermissionDenied {
		t.Fatalf("expected unauthenticated or permission_denied, got %v", code)
	}
}

// capturePublisher records the last published topic and message.
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
	tempDBName := fmt.Sprintf("registrations_svc_test_%d", time.Now().UnixNano())

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

	if _, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'"); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("disable FK triggers: %v", err)
	}

	for _, table := range []string{"event_registrations", "events", "organizations", "organization_types", "users"} {
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

func authedContext(kratosID string) context.Context {
	return context.WithValue(context.Background(), auth.UserIDKey, kratosID)
}

func seedAuthUser(t *testing.T, db *bun.DB, kratosID string) int32 {
	t.Helper()
	var id int32
	err := db.NewRaw(
		"INSERT INTO users (kratos_id, username, email, password) VALUES (?, ?, ?, ?) RETURNING id",
		kratosID, "seed-"+kratosID, kratosID+"@test.local", "test",
	).Scan(context.Background(), &id)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	return id
}

// seedEvent creates an organization and event, returns (eventID, orgID, eventTitle).
func seedEvent(t *testing.T, db *bun.DB, userID int32) (int32, int32, string) {
	t.Helper()
	ctx := context.Background()

	var orgTypeID int32
	err := db.NewRaw("INSERT INTO organization_types (title) VALUES ('Test Type') RETURNING id").Scan(ctx, &orgTypeID)
	if err != nil {
		t.Fatalf("seed org type: %v", err)
	}

	var orgID int32
	err = db.NewRaw("INSERT INTO organizations (title, organization_type_id) VALUES ('Test Org', ?) RETURNING id", orgTypeID).Scan(ctx, &orgID)
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}

	var eventID int32
	err = db.NewRaw(
		`INSERT INTO events (title, description, user_id, organization_id, start_time, end_time, structured_location, status, check_in_token, check_in_secret)
		 VALUES ('Test Event', '{}', ?, ?, NOW(), NOW() + interval '1 hour', '{}', 'published', 'testtoken12345', 'test-secret-for-registration')
		 RETURNING id`,
		userID, orgID,
	).Scan(ctx, &eventID)
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}

	return eventID, orgID, "Test Event"
}

func TestRegisterForEventPublishesRegistrationCreated(t *testing.T) {
	t.Run("first-time registration", func(t *testing.T) {
		db := newTestDB(t)
		kratosID := "kratos-reg-user"
		userID := seedAuthUser(t, db, kratosID)
		eventID, _, _ := seedEvent(t, db, userID)

		cap := &capturePublisher{}
		pub := events.NewPublisher(cap)
		enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))

		svc := NewEventRegistrationsService(db, pub, enforcer)
		ctx := authedContext(kratosID)

		resp, err := svc.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{
			EventId: eventID,
			UserId:  userID,
		}))
		if err != nil {
			t.Fatalf("RegisterForEvent() error = %v", err)
		}

		if cap.topic != events.TopicRegistrationCreated {
			t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicRegistrationCreated)
		}
		if cap.msg == nil {
			t.Fatal("expected a published message, got nil")
		}

		var got events.RegistrationCreatedPayload
		if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		if got.EventID != eventID {
			t.Fatalf("payload EventID = %d, want %d", got.EventID, eventID)
		}
		if got.ParticipantUserID != userID {
			t.Fatalf("payload ParticipantUserID = %d, want %d", got.ParticipantUserID, userID)
		}
		if got.ActorID != userID {
			t.Fatalf("payload ActorID = %d, want %d", got.ActorID, userID)
		}
		if got.EventTitle != "Test Event" {
			t.Fatalf("payload EventTitle = %q, want %q", got.EventTitle, "Test Event")
		}

		if resp.Msg == nil || resp.Msg.Registration == nil {
			t.Fatal("expected registration response, got nil")
		}
	})

	t.Run("reactivation of cancelled registration", func(t *testing.T) {
		db := newTestDB(t)
		kratosID := "kratos-react-user"
		userID := seedAuthUser(t, db, kratosID)
		eventID, _, _ := seedEvent(t, db, userID)

		// Seed a cancelled registration
		now := time.Now()
		reg := models.EventRegistration{
			EventID:      eventID,
			UserID:       userID,
			Status:       models.RegistrationStatusCancelled,
			RegisteredAt: now,
			CancelledAt:  &now,
		}
		_, err := db.NewInsert().Model(&reg).Exec(context.Background())
		if err != nil {
			t.Fatalf("seed cancelled registration: %v", err)
		}

		cap := &capturePublisher{}
		pub := events.NewPublisher(cap)
		enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))

		svc := NewEventRegistrationsService(db, pub, enforcer)
		ctx := authedContext(kratosID)

		resp, err := svc.RegisterForEvent(ctx, connect.NewRequest(&eventsv1.RegisterForEventRequest{
			EventId: eventID,
			UserId:  userID,
		}))
		if err != nil {
			t.Fatalf("RegisterForEvent() reactivation error = %v", err)
		}

		// Same topic and payload structure as first-time registration
		if cap.topic != events.TopicRegistrationCreated {
			t.Fatalf("published topic = %q, want %q", cap.topic, events.TopicRegistrationCreated)
		}
		if cap.msg == nil {
			t.Fatal("expected a published message, got nil")
		}

		var got events.RegistrationCreatedPayload
		if err := json.Unmarshal(cap.msg.Payload, &got); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}

		if got.EventID != eventID {
			t.Fatalf("payload EventID = %d, want %d", got.EventID, eventID)
		}
		if got.ParticipantUserID != userID {
			t.Fatalf("payload ParticipantUserID = %d, want %d", got.ParticipantUserID, userID)
		}
		if got.ActorID != userID {
			t.Fatalf("payload ActorID = %d, want %d", got.ActorID, userID)
		}
		if got.EventTitle != "Test Event" {
			t.Fatalf("payload EventTitle = %q, want %q", got.EventTitle, "Test Event")
		}

		if resp.Msg == nil || resp.Msg.Registration == nil {
			t.Fatal("expected registration response, got nil")
		}
	})
}
