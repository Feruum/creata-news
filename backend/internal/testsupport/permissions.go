//go:build integration

// Package testsupport supplies explicit fixtures for database/publisher integration tests.
package testsupport

import (
	"context"
	"net"
	"testing"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/perms"
	"google.golang.org/grpc"
)

type allowedPermissions struct {
	pb.UnimplementedPermissionsServiceServer
}

func (allowedPermissions) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{
		Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION,
	}, nil
}

// AllowedPermissions represents an already-authorized caller in publisher tests.
// It is excluded from production builds. Authorization tests must use real policies.
func AllowedPermissions(t *testing.T) *perms.Client {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for permission fixture: %v", err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, allowedPermissions{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test-only", true, false, "default")
	if err != nil {
		t.Fatalf("create permission fixture client: %v", err)
	}
	return client
}
