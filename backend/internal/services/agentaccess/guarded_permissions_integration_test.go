//go:build integration

package agentaccess

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	agentaccessv1 "github.com/studyverse/ems-backend/gen/agentaccess/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"testing"
)

type inheritedDocumentGraph struct {
	pb.UnimplementedPermissionsServiceServer
	orgID string
}

func (*inheritedDocumentGraph) CheckPermission(_ context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	result := pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	if req.Resource.ObjectType == "platform" {
		result = pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: result}, nil
}
func (graph *inheritedDocumentGraph) ReadRelationships(req *pb.ReadRelationshipsRequest, stream pb.PermissionsService_ReadRelationshipsServer) error {
	filter := req.RelationshipFilter
	relation, subjectType, subjectID := "parent_organization", "organization", graph.orgID
	if filter.ResourceType == "agent_session" {
		relation, subjectType, subjectID = "source_document", "document", "audit-doc"
	}
	return stream.Send(&pb.ReadRelationshipsResponse{Relationship: &pb.Relationship{Resource: &pb.ObjectReference{ObjectType: filter.ResourceType, ObjectId: filter.OptionalResourceId}, Relation: relation, Subject: &pb.SubjectReference{Object: &pb.ObjectReference{ObjectType: subjectType, ObjectId: subjectID}}}})
}
func TestAgentAccessRejectsStaleOrganizationMembership(t *testing.T) {
	ctx := context.Background()
	db := testsupport.ClonedDB(t)
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
	insert("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-doc-user','audit-doc-user','audit-doc-user@test.local','test') RETURNING id", &userID)
	insert("INSERT INTO organization_types(title) VALUES('Audit document type') RETURNING id", &orgTypeID)
	insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit document org',?) RETURNING id", &orgID, orgTypeID)
	insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'doc-reader','Reader',true) RETURNING id", &roleID, orgTypeID)
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?)", orgID, userID, roleID)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, &inheritedDocumentGraph{orgID: fmt.Sprint(orgID)})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db, client)
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-doc-user")
	request := connect.NewRequest(&agentaccessv1.CheckAccessRequest{ResourceType: agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_DOCUMENT, ResourceId: "audit-doc", Permission: agentaccessv1.AgentPermission_AGENT_PERMISSION_VIEW})
	response, err := service.CheckAccess(ctx, request)
	if err != nil || !response.Msg.Allowed {
		t.Fatalf("current membership denied: %v", err)
	}
	exec("UPDATE organization_members SET valid_until=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE user_id = ?", userID)
	response, err = service.CheckAccess(ctx, request)
	if err != nil || response.Msg.Allowed {
		t.Fatalf("expired document membership allowed: %v", err)
	}
	exec("DELETE FROM organization_members WHERE user_id = ?", userID)
	response, err = service.CheckAccess(ctx, request)
	if err != nil || response.Msg.Allowed {
		t.Fatalf("revoked document membership allowed: %v", err)
	}
	request.Msg.ResourceType = agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_SESSION
	request.Msg.ResourceId = "audit-session"
	request.Msg.Permission = agentaccessv1.AgentPermission_AGENT_PERMISSION_USE_DOCUMENT
	response, err = service.CheckAccess(ctx, request)
	if err != nil || response.Msg.Allowed {
		t.Fatalf("session can use revoked document: %v", err)
	}
}
