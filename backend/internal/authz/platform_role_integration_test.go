//go:build integration

package authz

import (
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"testing"
)

type stalePlatformGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (stalePlatformGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func TestDesiredPlatformRoleRejectsStaleAdministrator(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var id int32
	if err := db.NewRaw("INSERT INTO users(username,email,password) VALUES('audit-desired','audit-desired@test.local','test') RETURNING id").Scan(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS platform_user_roles(user_id integer PRIMARY KEY REFERENCES users(id),role text NOT NULL,updated_at timestamptz NOT NULL DEFAULT NOW())"); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, stalePlatformGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	e := NewEnforcer(db, client)
	if !e.CheckPlatformAdmin(ctx, id) {
		t.Fatal("legacy graph administrator denied")
	}
	for _, role := range []string{"none", "staff", "administrator"} {
		if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET role=EXCLUDED.role", id, role); err != nil {
			t.Fatal(err)
		}
		if got := e.CheckPlatformAdmin(ctx, id); got != (role == "administrator") {
			t.Fatalf("desired %s admin=%v", role, got)
		}
		if got := e.CheckPlatformStaff(ctx, id); got != (role != "none") {
			t.Fatalf("desired %s staff=%v", role, got)
		}
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE platform_user_roles"); err != nil {
		t.Fatal(err)
	}
	if e.CheckPlatformAdmin(ctx, id) {
		t.Fatal(fmt.Sprintf("database error authorized user %d", id))
	}
}
