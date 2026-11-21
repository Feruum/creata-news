//go:build integration

package main

import (
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type bootstrapRoleGraph struct {
	pb.UnimplementedPermissionsServiceServer
	mu      sync.Mutex
	updates []*pb.RelationshipUpdate
}

func (*bootstrapRoleGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func (g *bootstrapRoleGraph) WriteRelationships(_ context.Context, request *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.updates = request.Updates
	return &pb.WriteRelationshipsResponse{}, nil
}
func TestBootstrapRoleChangesReplaceDesiredAndGraphRoles(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var id int32
	if err := db.NewRaw("INSERT INTO users(kratos_id,username,email,password) VALUES('bootstrap-role-user','bootstrap-role-user','bootstrap-role-user@test.local','test') RETURNING id").Scan(ctx, &id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,'none')", id); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &bootstrapRoleGraph{}
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, client)
	handler := adminPromotionHandler(db, enforcer, "configured-secret")
	for _, role := range []string{"administrator", "staff", "none"} {
		request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/promote?user_id=%d&role=%s", id, role), nil)
		request.Header.Set("X-Admin-Secret", "configured-secret")
		response := httptest.NewRecorder()
		handler(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("role %s status=%d body=%s", role, response.Code, response.Body.String())
		}
		var desired string
		if err := db.NewRaw("SELECT role FROM platform_user_roles WHERE user_id=?", id).Scan(ctx, &desired); err != nil {
			t.Fatal(err)
		}
		if desired != role {
			t.Fatalf("role %s left desired %s", role, desired)
		}
		if enforcer.CheckPlatformAdmin(ctx, id) != (role == "administrator") || enforcer.CheckPlatformStaff(ctx, id) != (role != "none") {
			t.Fatalf("role %s retained wrong SQL-guarded grants", role)
		}
		graph.mu.Lock()
		updates := graph.updates
		graph.mu.Unlock()
		if len(updates) != 2 {
			t.Fatalf("non-atomic platform replacement: %d", len(updates))
		}
		for _, update := range updates {
			expected := pb.RelationshipUpdate_OPERATION_DELETE
			if update.Relationship.Relation == role {
				expected = pb.RelationshipUpdate_OPERATION_TOUCH
			}
			if update.Operation != expected {
				t.Fatalf("role %s left old %s grant", role, update.Relationship.Relation)
			}
		}
	}
	var pending int
	if err := db.NewRaw("SELECT COUNT(*) FROM domain_event_outbox WHERE topic='internal.platform.role.sync'").Scan(ctx, &pending); err != nil {
		t.Fatal(err)
	}
	if pending == 0 {
		t.Fatal("bootstrap role changes have no durable reconciliation")
	}
}
