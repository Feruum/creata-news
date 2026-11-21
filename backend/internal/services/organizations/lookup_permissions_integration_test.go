//go:build integration

package organizations

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/services/users"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"testing"
)

type staleOrganizationLookup struct {
	pb.UnimplementedPermissionsServiceServer
	orgID string
}

func (*staleOrganizationLookup) CheckPermission(_ context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	result := pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	if req.Resource.ObjectType == "platform" {
		result = pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: result}, nil
}
func (g *staleOrganizationLookup) LookupResources(_ *pb.LookupResourcesRequest, stream pb.PermissionsService_LookupResourcesServer) error {
	return stream.Send(&pb.LookupResourcesResponse{ResourceObjectId: g.orgID})
}
func TestOrganizationListsAndPermissionSummaryFilterStaleGraph(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	var userID, orgTypeID, orgID, roleID int32
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
	insert("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-lookup-user','audit-lookup-user','audit-lookup-user@test.local','test') RETURNING id", &userID)
	insert("INSERT INTO organization_types(title) VALUES('Audit lookup type') RETURNING id", &orgTypeID)
	insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit lookup org',?) RETURNING id", &orgID, orgTypeID)
	insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'lookup-reader','Reader',true) RETURNING id", &roleID, orgTypeID)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event'),(?,'manage_settings')", roleID, roleID)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?)", orgID, userID, roleID)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, &staleOrganizationLookup{orgID: fmt.Sprint(orgID)})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, client)
	service := NewService(db, client, enforcer, nil, nil, nil)
	userService := users.NewService(db, client, enforcer, nil, nil, "", "")
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-lookup-user")
	check := func(expected int) {
		t.Helper()
		response, err := service.GetUserOrganizations(ctx, connect.NewRequest(&eventsv1.GetUserOrganizationsRequest{}))
		if err != nil || len(response.Msg.Organizations) != expected {
			t.Fatalf("organization view count: %v %v", response, err)
		}
		publishable, err := service.GetPublishableOrganizations(ctx, connect.NewRequest(&eventsv1.GetPublishableOrganizationsRequest{}))
		if err != nil || len(publishable.Msg.Organizations) != expected {
			t.Fatalf("publishable count: %v %v", publishable, err)
		}
		me, err := userService.GetMe(ctx, connect.NewRequest(&usersv1.GetMeRequest{}))
		if err != nil || len(me.Msg.Permissions.ManagedClubIds) != expected {
			t.Fatalf("managed org count: %v %v", me, err)
		}
	}
	check(1)
	exec("UPDATE organization_members SET valid_until=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE user_id = ?", userID)
	check(0)
}
