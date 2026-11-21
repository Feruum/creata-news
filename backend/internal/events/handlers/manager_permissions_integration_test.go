//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type staleManagerGraph struct {
	pb.UnimplementedPermissionsServiceServer
	userIDs []string
	fail    atomic.Bool
}

func (*staleManagerGraph) CheckPermission(_ context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	result := pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	if req.Resource.ObjectType == "platform" {
		result = pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: result}, nil
}
func (g *staleManagerGraph) LookupSubjects(_ *pb.LookupSubjectsRequest, stream pb.PermissionsService_LookupSubjectsServer) error {
	if g.fail.Load() {
		return status.Error(codes.Unavailable, "forced lookup failure")
	}
	for _, id := range g.userIDs {
		if err := stream.Send(&pb.LookupSubjectsResponse{Subject: &pb.ResolvedSubject{SubjectObjectId: id}}); err != nil {
			return err
		}
	}
	return nil
}
func TestManagerNotificationsExcludeExpiredAndRevokedMemberships(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var activeID, expiredID, orgTypeID, orgID, roleID int32
	insert := func(query string, target *int32, args ...any) {
		t.Helper()
		if err := db.NewRaw(query, args...).Scan(ctx, target); err != nil {
			t.Fatal(err)
		}
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	insert("INSERT INTO users(username,email,password,telegram_id) VALUES('audit-active-manager','audit-active-manager@test.local','test',101) RETURNING id", &activeID)
	insert("INSERT INTO users(username,email,password,telegram_id) VALUES('audit-expired-manager','audit-expired-manager@test.local','test',202) RETURNING id", &expiredID)
	insert("INSERT INTO organization_types(title) VALUES('Audit manager type') RETURNING id", &orgTypeID)
	insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit manager org',?) RETURNING id", &orgID, orgTypeID)
	insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'notify-manager','Manager',true) RETURNING id", &roleID, orgTypeID)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'manage_settings')", roleID)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?),(?,?,?)", orgID, activeID, roleID, orgID, expiredID, roleID)
	exec("UPDATE organization_members SET valid_until=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE user_id = ?", expiredID)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &staleManagerGraph{userIDs: []string{fmt.Sprint(activeID), fmt.Sprint(expiredID)}}
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	deps := HandlerDeps{DB: db, Perms: client, InternalNotificationSecret: "audit-notification-secret"}
	msg := message.NewMessageWithContext(ctx, "audit-manager", nil)
	ids, err := resolveOrgManagerTelegramIDs(msg, deps, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 101 {
		t.Fatalf("expired manager notified: %v", ids)
	}
	var posted atomic.Int32
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != deps.InternalNotificationSecret {
			t.Error("notification credential missing")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body telegramBulkRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Notifications) != 1 || body.Notifications[0].TelegramID != 101 {
			t.Error("invalid or unauthorized notification recipient payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		posted.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"sent":1,"failed":0,"total":1}`))
	}))
	defer httpServer.Close()
	deps.TelegramBotURL = httpServer.URL
	inviteData, inviteErr := json.Marshal(events.CollaborationInvitedPayload{OrganizationID: orgID, HostOrganizationID: 999, EventTitle: "Audit invite"})
	if inviteErr != nil {
		t.Fatal(inviteErr)
	}
	if err := handleCollaborationInvited(deps)(message.NewMessageWithContext(ctx, "audit-invited-org", inviteData)); err != nil {
		t.Fatal(err)
	}
	if posted.Load() != 1 {
		t.Fatal("invited organization managers were not notified")
	}
	posted.Store(0)
	data, err := json.Marshal(events.BookingApprovedPayload{OrganizationID: orgID})
	if err != nil {
		t.Fatal(err)
	}
	notification := message.NewMessageWithContext(ctx, "audit-manager-retry", data)
	handler := handleBookingApproved(deps)
	graph.fail.Store(true)
	if err := handler(notification); err == nil {
		t.Fatal("lookup failure acknowledged; notification cannot retry")
	}
	graph.fail.Store(false)
	if err := handler(notification); err != nil {
		t.Fatal(err)
	}
	if posted.Load() != 1 {
		t.Fatal("retry did not deliver notification")
	}
	exec("ALTER TABLE users RENAME TO audit_hidden_users")
	failure := handler(notification)
	exec("ALTER TABLE audit_hidden_users RENAME TO users")
	if failure == nil {
		t.Fatal("recipient database failure acknowledged")
	}
	if posted.Load() != 1 {
		t.Fatal("partial lookup sent notification")
	}
	if err := handler(notification); err != nil {
		t.Fatal(err)
	}
	if posted.Load() != 2 {
		t.Fatal("database recovery retry did not deliver notification")
	}
	exec("ALTER TABLE organization_members RENAME TO audit_hidden_members")
	failure = handler(notification)
	exec("ALTER TABLE audit_hidden_members RENAME TO organization_members")
	if failure == nil {
		t.Fatal("membership permission database failure acknowledged")
	}
	if posted.Load() != 2 {
		t.Fatal("failed permission query sent notification")
	}
	exec("DELETE FROM organization_members WHERE user_id = ?", activeID)
	if ids, err := resolveOrgManagerTelegramIDs(msg, deps, orgID); err != nil || len(ids) != 0 {
		t.Fatalf("revoked manager notified: %v", ids)
	}
}

func TestPublishedNotificationSubscriberQueryFailureIsRetryable(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var published struct {
		ID             int32 `bun:"id"`
		OrganizationID int32 `bun:"organization_id"`
	}
	if err := db.NewRaw(`WITH actor AS (
		INSERT INTO users(username,email,password)
		VALUES('subscriber-retry','subscriber-retry@example.invalid','test') RETURNING id
	), kind AS (
		INSERT INTO organization_types(title) VALUES('Subscriber retry') RETURNING id
	), org AS (
		INSERT INTO organizations(title,organization_type_id)
		SELECT 'Subscriber retry',id FROM kind RETURNING id
	)
	INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,
		structured_location,status,publish_on_map,check_in_secret,check_in_token)
	SELECT 'Subscriber retry','{}'::jsonb,actor.id,org.id,NOW(),NOW()+INTERVAL '1 hour',
		'{}'::jsonb,'published',true,'subscriber-retry','subscriber-retry'
	FROM actor,org RETURNING id,organization_id`).Scan(ctx, &published); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(events.EventPublishedPayload{
		EventID: published.ID, OrganizationID: published.OrganizationID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE organization_subscriptions RENAME TO audit_hidden_subscriptions"); err != nil {
		t.Fatal(err)
	}
	handler := handleEventPublished(HandlerDeps{DB: db, TelegramBotURL: "http://127.0.0.1:1"})
	err = handler(message.NewMessageWithContext(ctx, "audit-subscriber-failure", payload))
	if err == nil {
		t.Fatal("subscriber query failure acknowledged")
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE audit_hidden_subscriptions RENAME TO organization_subscriptions"); err != nil {
		t.Fatal(err)
	}
	if err := handler(message.NewMessageWithContext(ctx, "audit-subscriber-failure", payload)); err != nil {
		t.Fatalf("retry after recovery: %v", err)
	}
}
