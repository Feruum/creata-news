//go:build integration

package users

import (
	"connectrpc.com/connect"
	"context"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"sync/atomic"
	"testing"
)

type unavailableRoleGraph struct {
	pb.UnimplementedPermissionsServiceServer
	fail atomic.Bool
}

func (*unavailableRoleGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func (graph *unavailableRoleGraph) WriteRelationships(context.Context, *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	if graph.fail.Load() {
		return nil, status.Error(codes.Unavailable, "graph unavailable")
	}
	return &pb.WriteRelationshipsResponse{}, nil
}
func TestPlatformRevocationDoesNotReportSuccessWhenGraphWriteFails(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var id int32
	if err := db.NewRaw("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-platform-actor','audit-platform-actor','audit-platform-actor@test.local','test') RETURNING id").Scan(ctx, &id); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &unavailableRoleGraph{}
	graph.fail.Store(true)
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db, client, authz.NewEnforcer(db, client), nil, nil, "", "")
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-platform-actor")
	_, err = service.AssignPlatformRole(ctx, connect.NewRequest(&usersv1.AssignPlatformRoleRequest{UserId: id, Role: usersv1.PlatformRole_PLATFORM_ROLE_USER}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("failed revocation reported as successful: %v", err)
	}
	getMe, getMeErr := service.GetMe(ctx, connect.NewRequest(&usersv1.GetMeRequest{}))
	if getMeErr != nil {
		t.Fatal(getMeErr)
	}
	if getMe.Msg.Permissions.IsAdmin || getMe.Msg.Permissions.IsStaff {
		t.Fatal("GetMe exposed revoked platform permissions")
	}
	if service.authz.CheckPlatformAdmin(ctx, id) || service.authz.CheckPlatformStaff(ctx, id) {
		t.Fatal("failed graph write retained revoked platform access")
	}
	var pending int
	if err := db.NewRaw("SELECT COUNT(*) FROM domain_event_outbox WHERE topic='internal.platform.role.sync'").Scan(ctx, &pending); err != nil {
		t.Fatal(err)
	}
	if pending != 1 {
		t.Fatalf("revocation has no durable retry: %d", pending)
	}
	graph.fail.Store(false)
	dispatcher := events.NewOutboxDispatcher(db, nil, nil, service.authz.SyncPlatformUserRole)
	if err := dispatcher.DispatchOne(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.NewRaw("SELECT COUNT(*) FROM domain_event_outbox WHERE topic='internal.platform.role.sync'").Scan(ctx, &pending); err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatal("successful reconciliation did not complete pending job")
	}
}

func TestPlatformRoleEnqueueFailurePreservesPreviousDesiredRole(t *testing.T) {
	s, db, id := preregTestService(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "UPDATE users SET kratos_id='audit-invite-actor' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,'administrator')", id); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"CREATE FUNCTION reject_role_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced enqueue failure'; END $$", "CREATE TRIGGER reject_role_outbox BEFORE INSERT ON domain_event_outbox FOR EACH ROW EXECUTE FUNCTION reject_role_outbox()"} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-invite-actor")
	_, err := s.AssignPlatformRole(ctx, connect.NewRequest(&usersv1.AssignPlatformRoleRequest{UserId: id, Role: usersv1.PlatformRole_PLATFORM_ROLE_USER}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("enqueue failure swallowed: %v", err)
	}
	var role string
	if err := db.NewRaw("SELECT role FROM platform_user_roles WHERE user_id=?", id).Scan(ctx, &role); err != nil {
		t.Fatal(err)
	}
	if role != "administrator" {
		t.Fatalf("failed enqueue persisted role %s", role)
	}
}
