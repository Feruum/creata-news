//go:build integration

package users

import (
	"context"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
	"net"
	"testing"
)

type successfulRoleGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (*successfulRoleGraph) WriteRelationships(context.Context, *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	return &pb.WriteRelationshipsResponse{}, nil
}
func (*successfulRoleGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func preregTestService(t *testing.T) (*Service, *bun.DB, int32) {
	t.Helper()
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var id int32
	if err := db.NewRaw("INSERT INTO users(username,email,password) VALUES('audit-invite','audit-invite@test.local','test') RETURNING id").Scan(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS platform_user_roles(user_id integer PRIMARY KEY REFERENCES users(id),role text NOT NULL,updated_at timestamptz NOT NULL DEFAULT NOW())"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO pre_registered_users(email,platform_role) VALUES('audit-invite@test.local','admin')"); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, &successfulRoleGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	return NewService(db, client, authz.NewEnforcer(db, client), nil, nil, "", ""), db, id
}
func TestInvitationConsumptionRollsBackWhenReconciliationEnqueueFails(t *testing.T) {
	s, db, id := preregTestService(t)
	ctx := context.Background()
	for _, query := range []string{"CREATE FUNCTION reject_invite_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced enqueue failure'; END $$", "CREATE TRIGGER reject_invite_outbox BEFORE INSERT ON domain_event_outbox FOR EACH ROW EXECUTE FUNCTION reject_invite_outbox()"} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.applyPreRegisteredRole(ctx, "audit-invite@test.local", id); err == nil {
		t.Fatal("enqueue failure swallowed")
	}
	var consumed bool
	if err := db.NewRaw("SELECT used_at IS NOT NULL FROM pre_registered_users WHERE email='audit-invite@test.local'").Scan(ctx, &consumed); err != nil {
		t.Fatal(err)
	}
	if consumed {
		t.Fatal("invitation consumed despite failed durable reconciliation")
	}
	var count int
	if err := db.NewRaw("SELECT COUNT(*) FROM platform_user_roles WHERE user_id=?", id).Scan(ctx, &count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("desired grant persisted despite failed enqueue")
	}
}
func TestInvitationCannotRegrantExplicitlyRevokedRole(t *testing.T) {
	s, db, id := preregTestService(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,'none')", id); err != nil {
		t.Fatal(err)
	}
	s.applyPreRegisteredRole(ctx, "audit-invite@test.local", id)
	var role string
	if err := db.NewRaw("SELECT role FROM platform_user_roles WHERE user_id=?", id).Scan(ctx, &role); err != nil {
		t.Fatal(err)
	}
	if role != "none" {
		t.Fatalf("revoked role regranted: %s", role)
	}
	if s.authz.CheckPlatformAdmin(ctx, id) {
		t.Fatal("stale invitation graph grant restored admin access")
	}
	var count int
	if err := db.NewRaw("SELECT COUNT(*) FROM domain_event_outbox").Scan(ctx, &count); err != nil {
		t.Fatal(err)
	}
	s.applyPreRegisteredRole(ctx, "audit-invite@test.local", id)
	var after int
	if err := db.NewRaw("SELECT COUNT(*) FROM domain_event_outbox").Scan(ctx, &after); err != nil {
		t.Fatal(err)
	}
	if after != count {
		t.Fatal("consumed invitation re-enqueued")
	}
}
