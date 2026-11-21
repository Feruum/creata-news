//go:build integration

package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/ThreeDotsLabs/watermill/message"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type capturePublisher struct {
	topics []string
	msgs   []*message.Message
}

func (p *capturePublisher) Publish(topic string, messages ...*message.Message) error {
	p.topics = append(p.topics, topic)
	if len(messages) > 0 {
		p.msgs = append(p.msgs, messages[0])
	}
	return nil
}

func (p *capturePublisher) Close() error { return nil }

type failingPublisher struct{}

func (p *failingPublisher) Publish(topic string, messages ...*message.Message) error {
	return errors.New("publish failed")
}

func (p *failingPublisher) Close() error { return nil }

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
	tempDBName := fmt.Sprintf("events_svc_test_%d", time.Now().UnixNano())

	ctx := context.Background()
	createQuery := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", quoteIdentifier(tempDBName), quoteIdentifier(sourceDBName))
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

	truncate := strings.Join([]string{
		"TRUNCATE event_collaborators RESTART IDENTITY CASCADE",
		"TRUNCATE event_collaboration_invites RESTART IDENTITY CASCADE",
		"TRUNCATE event_slugs RESTART IDENTITY CASCADE",
		"TRUNCATE events RESTART IDENTITY CASCADE",
		"TRUNCATE organizations RESTART IDENTITY CASCADE",
		"TRUNCATE users RESTART IDENTITY CASCADE",
	}, "; ")
	if _, err := db.ExecContext(ctx, truncate); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("truncate event test tables: %v", err)
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
		kratosID,
		fmt.Sprintf("user_%s", strings.ReplaceAll(kratosID, "-", "_")),
		fmt.Sprintf("%s@test.local", strings.ReplaceAll(kratosID, "-", "_")),
		"test",
	).Scan(context.Background(), &id)
	if err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	return id
}

func seedOrganization(t *testing.T, db *bun.DB, title string) models.Organization {
	t.Helper()
	status := models.OrganizationStatusActive
	org := models.Organization{
		Title:              title,
		OrganizationTypeID: 1,
		Status:             &status,
	}
	if _, err := db.NewInsert().Model(&org).Returning("*").Exec(context.Background()); err != nil {
		t.Fatalf("seed organization %q: %v", title, err)
	}
	return org
}

func seedDraftEvent(t *testing.T, db *bun.DB, actorID, orgID int32, title string, location string) models.Event {
	t.Helper()
	start := time.Date(2026, time.April, 10, 15, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	structuredLocation := json.RawMessage(fmt.Sprintf(`{"type":"external","external_address":%q}`, location))
	checkInSecret := fmt.Sprintf("%064x", time.Now().UnixNano())
	checkInToken := fmt.Sprintf("%012x", time.Now().UnixNano())[:12]
	event := models.Event{
		Title:              title,
		Description:        json.RawMessage(`"test event"`),
		UserID:             actorID,
		OrganizationID:     orgID,
		StartTime:          start,
		EndTime:            end,
		Status:             models.EventStatusDraft,
		PublishOnMap:       true,
		StructuredLocation: structuredLocation,
		CheckInSecret:      &checkInSecret,
		CheckInToken:       &checkInToken,
	}
	if _, err := db.NewInsert().Model(&event).Returning("*").Exec(context.Background()); err != nil {
		t.Fatalf("seed event %q: %v", title, err)
	}
	return event
}

func seedEventSlug(t *testing.T, db *bun.DB, eventID int32, slug string) {
	t.Helper()
	eventSlug := models.EventSlug{EventID: eventID, Slug: slug, IsCurrent: true}
	if _, err := db.NewInsert().Model(&eventSlug).Exec(context.Background()); err != nil {
		t.Fatalf("seed event slug %q: %v", slug, err)
	}
}

func TestPublishEventPublishesEventPublished(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-publish-event"
	actorID := seedAuthUser(t, db, kratosID)
	hostOrg := seedOrganization(t, db, "Host Organization")
	event := seedDraftEvent(t, db, actorID, hostOrg.ID, "Robotics Summit", "Main Hall")
	seedEventSlug(t, db, event.ID, "robotics-summit")

	cap := &capturePublisher{}
	pub := domainevents.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, pub, "https://map.studyverse.test", "test-reminder-secret")
	ctx := authedContext(kratosID)

	resp, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
	if err != nil {
		t.Fatalf("PublishEvent() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.Event == nil {
		t.Fatal("expected published event response, got nil")
	}
	if len(cap.topics) != 1 {
		t.Fatalf("publish call count = %d, want 1", len(cap.topics))
	}
	if cap.topics[0] != domainevents.TopicEventPublished {
		t.Fatalf("published topic = %q, want %q", cap.topics[0], domainevents.TopicEventPublished)
	}
	if len(cap.msgs) != 1 || cap.msgs[0] == nil {
		t.Fatal("expected one published message")
	}

	var got domainevents.EventPublishedPayload
	if err := json.Unmarshal(cap.msgs[0].Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if got.EventID != resp.Msg.Event.Id {
		t.Fatalf("payload EventID = %d, response Event.Id = %d", got.EventID, resp.Msg.Event.Id)
	}
	if got.Title != event.Title {
		t.Fatalf("payload Title = %q, want %q", got.Title, event.Title)
	}
	if got.OrganizationID != hostOrg.ID {
		t.Fatalf("payload OrganizationID = %d, want %d", got.OrganizationID, hostOrg.ID)
	}
	if got.OrganizationName != hostOrg.Title {
		t.Fatalf("payload OrganizationName = %q, want %q", got.OrganizationName, hostOrg.Title)
	}
	if got.StartTime != event.StartTime.Format(time.RFC3339) {
		t.Fatalf("payload StartTime = %q, want %q", got.StartTime, event.StartTime.Format(time.RFC3339))
	}
	if got.Location != "Main Hall" {
		t.Fatalf("payload Location = %q, want %q", got.Location, "Main Hall")
	}
	if got.EventSlug != "robotics-summit" {
		t.Fatalf("payload EventSlug = %q, want %q", got.EventSlug, "robotics-summit")
	}
	if got.MapAppURL != "https://map.studyverse.test" {
		t.Fatalf("payload MapAppURL = %q, want %q", got.MapAppURL, "https://map.studyverse.test")
	}
	if got.ActorID != actorID {
		t.Fatalf("payload ActorID = %d, want %d", got.ActorID, actorID)
	}
}

func TestPublishEvent_WhenPublishFailsStillSucceeds(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-publish-event-publish-failure"
	actorID := seedAuthUser(t, db, kratosID)
	hostOrg := seedOrganization(t, db, "Host Organization")
	event := seedDraftEvent(t, db, actorID, hostOrg.ID, "Robotics Summit", "Main Hall")
	seedEventSlug(t, db, event.ID, "robotics-summit")

	pub := domainevents.NewPublisher(&failingPublisher{})
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, pub, "https://map.studyverse.test", "test-reminder-secret")
	ctx := authedContext(kratosID)

	resp, err := svc.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: event.ID}))
	if err != nil {
		t.Fatalf("PublishEvent() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.Event == nil {
		t.Fatal("expected published event response, got nil")
	}
	if resp.Msg.Event.Status != eventsv1.EventStatus_EVENT_STATUS_PUBLISHED {
		t.Fatalf("response status = %v, want %v", resp.Msg.Event.Status, eventsv1.EventStatus_EVENT_STATUS_PUBLISHED)
	}

	var persisted models.Event
	if err := db.NewSelect().Model(&persisted).Where("id = ?", event.ID).Scan(context.Background()); err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if persisted.Status != models.EventStatusPublished {
		t.Fatalf("persisted status = %q, want %q", persisted.Status, models.EventStatusPublished)
	}
}
