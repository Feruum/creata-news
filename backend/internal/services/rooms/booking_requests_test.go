//go:build integration

package rooms

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
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
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
	tempDBName := fmt.Sprintf("rooms_svc_test_%d", time.Now().UnixNano())

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
		"TRUNCATE room_bookings RESTART IDENTITY CASCADE",
		"TRUNCATE booking_requests RESTART IDENTITY CASCADE",
		"TRUNCATE rooms RESTART IDENTITY CASCADE",
		"TRUNCATE events RESTART IDENTITY CASCADE",
		"TRUNCATE organizations RESTART IDENTITY CASCADE",
		"TRUNCATE users RESTART IDENTITY CASCADE",
	}, "; ")
	if _, err := db.ExecContext(ctx, truncate); err != nil {
		_ = db.Close()
		_, _ = adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdentifier(tempDBName)))
		_ = adminDB.Close()
		t.Fatalf("truncate room test tables: %v", err)
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

func seedEvent(t *testing.T, db *bun.DB, actorID, orgID int32, title string) models.Event {
	t.Helper()
	start := time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	checkInSecret := fmt.Sprintf("%064x", time.Now().UnixNano())
	checkInToken := fmt.Sprintf("%012x", time.Now().UnixNano())[:12]
	event := models.Event{
		Title:              title,
		Description:        json.RawMessage(`"room booking event"`),
		UserID:             actorID,
		OrganizationID:     orgID,
		StartTime:          start,
		EndTime:            end,
		Status:             models.EventStatusPublished,
		PublishOnMap:       true,
		StructuredLocation: json.RawMessage(`{"type":"campus_room","room_id":"101","room_name":"Conference Room"}`),
		CheckInSecret:      &checkInSecret,
		CheckInToken:       &checkInToken,
	}
	if _, err := db.NewInsert().Model(&event).Returning("*").Exec(context.Background()); err != nil {
		t.Fatalf("seed event %q: %v", title, err)
	}
	return event
}

func seedRoom(t *testing.T, db *bun.DB, id string, name string) models.Room {
	t.Helper()
	room := models.Room{
		ID:           id,
		Name:         name,
		RoomNumber:   id,
		Block:        "A",
		Floor:        1,
		Type:         models.RoomTypeClassroom,
		IsAccessible: true,
		IsActive:     true,
		IsBookable:   true,
	}
	if _, err := db.NewInsert().Model(&room).Exec(context.Background()); err != nil {
		t.Fatalf("seed room %q: %v", id, err)
	}
	return room
}

func seedBookingRequest(t *testing.T, db *bun.DB, requesterID int32, orgID int32, eventID int32, roomID string, title string) models.BookingRequest {
	t.Helper()
	start := time.Date(2026, time.April, 12, 10, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	request := models.BookingRequest{
		RoomID:         roomID,
		RequesterID:    requesterID,
		OrganizationID: &orgID,
		Priority:       models.BookingPriorityStudentClub,
		Title:          title,
		StartTime:      start,
		EndTime:        end,
		Status:         models.BookingRequestStatusPending,
		EventID:        &eventID,
	}
	if _, err := db.NewInsert().Model(&request).Returning("*").Exec(context.Background()); err != nil {
		t.Fatalf("seed booking request %q: %v", title, err)
	}
	return request
}

func TestApproveBookingRequestPublishesBookingApproved(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-booking-approved"
	actorID := seedAuthUser(t, db, kratosID)
	org := seedOrganization(t, db, "Physics Club")
	event := seedEvent(t, db, actorID, org.ID, "Physics Club Seminar")
	room := seedRoom(t, db, "101", "Conference Room")
	bookingReq := seedBookingRequest(t, db, actorID, org.ID, event.ID, room.ID, "Physics Club Seminar")

	cap := &capturePublisher{}
	pub := domainevents.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)
	ctx := authedContext(kratosID)
	eventID := event.ID

	resp, err := svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{
		Id:      bookingReq.ID,
		EventId: &eventID,
	}))
	if err != nil {
		t.Fatalf("ApproveBookingRequest() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.BookingRequest == nil || resp.Msg.Booking == nil {
		t.Fatal("expected approved booking response, got nil")
	}
	if len(cap.topics) != 1 {
		t.Fatalf("publish call count = %d, want 1", len(cap.topics))
	}
	if cap.topics[0] != domainevents.TopicBookingApproved {
		t.Fatalf("published topic = %q, want %q", cap.topics[0], domainevents.TopicBookingApproved)
	}
	if len(cap.msgs) != 1 || cap.msgs[0] == nil {
		t.Fatal("expected one published message")
	}

	var got domainevents.BookingApprovedPayload
	if err := json.Unmarshal(cap.msgs[0].Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if got.BookingRequestID != bookingReq.ID {
		t.Fatalf("payload BookingRequestID = %d, want %d", got.BookingRequestID, bookingReq.ID)
	}
	if got.EventID != event.ID {
		t.Fatalf("payload EventID = %d, want %d", got.EventID, event.ID)
	}
	if got.EventTitle != bookingReq.Title {
		t.Fatalf("payload EventTitle = %q, want %q", got.EventTitle, bookingReq.Title)
	}
	if got.OrganizationID != org.ID {
		t.Fatalf("payload OrganizationID = %d, want %d", got.OrganizationID, org.ID)
	}
	if got.RoomID != "101" {
		t.Fatalf("payload RoomID = %q, want %q", got.RoomID, "101")
	}
	if got.RoomName != room.Name {
		t.Fatalf("payload RoomName = %q, want %q", got.RoomName, room.Name)
	}
	if got.ActorID != actorID {
		t.Fatalf("payload ActorID = %d, want %d", got.ActorID, actorID)
	}
}

func TestApproveBookingRequest_PublishesBookingApprovedWithStringRoomID(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-booking-approved-nonnumeric-room"
	actorID := seedAuthUser(t, db, kratosID)
	org := seedOrganization(t, db, "Physics Club")
	event := seedEvent(t, db, actorID, org.ID, "Canonical Event Title")
	room := seedRoom(t, db, "A-101", "Studio A-101")
	bookingReq := seedBookingRequest(t, db, actorID, org.ID, event.ID, room.ID, "Booking Request Title")

	cap := &capturePublisher{}
	pub := domainevents.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)
	ctx := authedContext(kratosID)
	eventID := event.ID

	resp, err := svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{
		Id:      bookingReq.ID,
		EventId: &eventID,
	}))
	if err != nil {
		t.Fatalf("ApproveBookingRequest() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.BookingRequest == nil || resp.Msg.Booking == nil {
		t.Fatal("expected approved booking response, got nil")
	}
	if len(cap.topics) != 1 {
		t.Fatalf("publish call count = %d, want 1", len(cap.topics))
	}
	if cap.topics[0] != domainevents.TopicBookingApproved {
		t.Fatalf("published topic = %q, want %q", cap.topics[0], domainevents.TopicBookingApproved)
	}
	if len(cap.msgs) != 1 || cap.msgs[0] == nil {
		t.Fatal("expected one published message")
	}

	var got domainevents.BookingApprovedPayload
	if err := json.Unmarshal(cap.msgs[0].Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.RoomID != "A-101" {
		t.Fatalf("payload RoomID = %q, want %q", got.RoomID, "A-101")
	}
	if got.RoomName != room.Name {
		t.Fatalf("payload RoomName = %q, want %q", got.RoomName, room.Name)
	}
}

func TestApproveBookingRequest_WhenPublishFailsStillSucceeds(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-booking-approved-publish-failure"
	actorID := seedAuthUser(t, db, kratosID)
	org := seedOrganization(t, db, "Physics Club")
	event := seedEvent(t, db, actorID, org.ID, "Canonical Event Title")
	room := seedRoom(t, db, "101", "Conference Room")
	bookingReq := seedBookingRequest(t, db, actorID, org.ID, event.ID, room.ID, "Booking Request Title")

	pub := domainevents.NewDurablePublisher(db, &failingPublisher{})
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)
	ctx := authedContext(kratosID)
	eventID := event.ID

	resp, err := svc.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{
		Id:      bookingReq.ID,
		EventId: &eventID,
	}))
	if err != nil {
		t.Fatalf("ApproveBookingRequest() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.BookingRequest == nil || resp.Msg.Booking == nil {
		t.Fatal("expected approved booking response, got nil")
	}
	if resp.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_APPROVED {
		t.Fatalf("booking request status = %v, want %v", resp.Msg.BookingRequest.Status, roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_APPROVED)
	}

	var queued []domainevents.OutboxMessage
	if err := db.NewSelect().Model(&queued).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(queued) != 1 || queued[0].Topic != domainevents.TopicBookingApproved {
		t.Fatalf("approval not retained for delivery: %+v", queued)
	}

	var persisted models.BookingRequest
	if err := db.NewSelect().Model(&persisted).Where("id = ?", bookingReq.ID).Scan(context.Background()); err != nil {
		t.Fatalf("reload booking request: %v", err)
	}
	if persisted.Status != models.BookingRequestStatusApproved {
		t.Fatalf("persisted status = %q, want %q", persisted.Status, models.BookingRequestStatusApproved)
	}
}

func TestRejectBookingRequest_PublishesBookingRejectedWithStringRoomID(t *testing.T) {
	db := newTestDB(t)
	kratosID := "kratos-test-booking-rejected-nonnumeric-room"
	actorID := seedAuthUser(t, db, kratosID)
	org := seedOrganization(t, db, "Physics Club")
	event := seedEvent(t, db, actorID, org.ID, "Canonical Event Title")
	room := seedRoom(t, db, "A-101", "Studio A-101")
	bookingReq := seedBookingRequest(t, db, actorID, org.ID, event.ID, room.ID, "Booking Request Title")

	cap := &capturePublisher{}
	pub := domainevents.NewPublisher(cap)
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, pub)
	ctx := authedContext(kratosID)

	resp, err := svc.RejectBookingRequest(ctx, connect.NewRequest(&roomsv1.RejectBookingRequestRequest{
		Id:     bookingReq.ID,
		Reason: "conflict",
	}))
	if err != nil {
		t.Fatalf("RejectBookingRequest() error = %v", err)
	}
	if resp.Msg == nil || resp.Msg.BookingRequest == nil {
		t.Fatal("expected rejected booking response, got nil")
	}
	if len(cap.topics) != 1 {
		t.Fatalf("publish call count = %d, want 1", len(cap.topics))
	}
	if cap.topics[0] != domainevents.TopicBookingRejected {
		t.Fatalf("published topic = %q, want %q", cap.topics[0], domainevents.TopicBookingRejected)
	}
	if len(cap.msgs) != 1 || cap.msgs[0] == nil {
		t.Fatal("expected one published message")
	}

	var got domainevents.BookingRejectedPayload
	if err := json.Unmarshal(cap.msgs[0].Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.RoomID != "A-101" {
		t.Fatalf("payload RoomID = %q, want %q", got.RoomID, "A-101")
	}
	if got.RoomName != room.Name {
		t.Fatalf("payload RoomName = %q, want %q", got.RoomName, room.Name)
	}
}
