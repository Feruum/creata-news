//go:build integration

package users

import (
	"connectrpc.com/connect"
	"context"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"sync/atomic"
	"testing"
)

type changingPlatformGraph struct {
	pb.UnimplementedPermissionsServiceServer
	calls atomic.Int32
}

func (graph *changingPlatformGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	permission := pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	if graph.calls.Add(1) == 1 {
		permission = pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: permission}, nil
}

func TestGetMeKeepsRoleAndPermissionsConsistentWithinResponse(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.WithValue(context.Background(), auth.UserIDKey, "qa-consistent-admin")
	if _, err := db.ExecContext(ctx, "INSERT INTO users(kratos_id,username,email,password) VALUES('qa-consistent-admin','qa-consistent-admin','qa-admin@example.test','synthetic')"); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &changingPlatformGraph{}
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "synthetic", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, client, authz.NewEnforcer(db, client), nil, nil, "", "")
	response, err := svc.GetMe(ctx, connect.NewRequest(&usersv1.GetMeRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.User.PlatformRole != usersv1.PlatformRole_PLATFORM_ROLE_ADMIN || !response.Msg.Permissions.IsAdmin || !response.Msg.Permissions.IsStaff {
		t.Fatalf("role and permissions disagree: %v", response.Msg)
	}
}
