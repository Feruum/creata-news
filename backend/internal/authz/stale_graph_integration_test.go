//go:build integration

package authz

import (
	"context"
	"database/sql"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/grpc"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type staleGraph struct {
	pb.UnimplementedPermissionsServiceServer
}

func (staleGraph) CheckPermission(_ context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	allowed := pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	if req.Resource.ObjectType == "platform" {
		allowed = pb.CheckPermissionResponse_PERMISSIONSHIP_NO_PERMISSION
	}
	return &pb.CheckPermissionResponse{Permissionship: allowed}, nil
}
func auditAuthzDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL must name a disposable test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.TrimPrefix(parsed.Path, "/")
	parsed.Path = "/postgres"
	admin := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(parsed.String()))), pgdialect.New())
	name := fmt.Sprintf("authz_audit_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(context.Background(), fmt.Sprintf(`CREATE DATABASE "%s" TEMPLATE "%s"`, name, source)); err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + name
	db := bun.NewDB(sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(parsed.String()))), pgdialect.New())
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), fmt.Sprintf(`DROP DATABASE "%s" WITH (FORCE)`, name))
		_ = admin.Close()
	})
	return db
}
func TestStaleGraphCannotRestoreRevokedCapabilities(t *testing.T) {
	ctx := context.Background()
	db := auditAuthzDB(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, staleGraph{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	graph, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	enforcer := NewEnforcer(db, graph)
	var userID, orgID, orgTypeID, roleID, eventID int32
	mustID := func(query string, target *int32, args ...any) {
		t.Helper()
		if err := db.NewRaw(query, args...).Scan(ctx, target); err != nil {
			t.Fatal(err)
		}
	}
	mustID("INSERT INTO users(username,email,password) VALUES('audit-permissions','audit-permissions@test.local','test') RETURNING id", &userID)
	mustID("INSERT INTO organization_types(title) VALUES('Audit permissions type') RETURNING id", &orgTypeID)
	mustID("INSERT INTO organizations(title,organization_type_id) VALUES('Audit permissions org',?) RETURNING id", &orgID, orgTypeID)
	mustID("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'audit','Audit',true) RETURNING id", &roleID, orgTypeID)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", roleID)
	if enforcer.CheckOrgPermission(ctx, userID, orgID, "create_event") {
		t.Fatal("deleted membership authorized from stale graph")
	}
	if len(enforcer.FilterOrganizations(ctx, userID, []string{fmt.Sprint(orgID)}, "create_event")) != 0 {
		t.Fatal("lookup exposes deleted membership")
	}
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_at) VALUES(?,?,?,CURRENT_TIMESTAMP)", orgID, userID, roleID)
	if !enforcer.CheckOrgPermission(ctx, userID, orgID, "create_event") {
		t.Fatal("active membership denied")
	}
	exec("DELETE FROM role_template_capabilities WHERE role_template_id = ?", roleID)
	if enforcer.CheckOrgPermission(ctx, userID, orgID, "create_event") {
		t.Fatal("removed capability authorized from stale graph")
	}
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", roleID)
	exec("UPDATE organization_members SET valid_until=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE user_id = ?", userID)
	if enforcer.CheckOrgPermission(ctx, userID, orgID, "create_event") {
		t.Fatal("expired membership authorized")
	}
	if len(enforcer.FilterOrganizations(ctx, userID, []string{fmt.Sprint(orgID)}, "create_event")) != 0 {
		t.Fatal("lookup exposes expired membership")
	}
	mustID("INSERT INTO events(title,description,user_id,organization_id,start_time,end_time,format,status,structured_location,check_in_secret,check_in_token) VALUES('Audit event','{}',?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP + INTERVAL '1 hour','online','draft','{}','audit-test-secret','audit-token') RETURNING id", &eventID, userID, orgID)
	if !enforcer.CheckResourcePermission(ctx, userID, "event", fmt.Sprint(eventID), "edit") {
		t.Fatal("DB creator denied")
	}
	var collaboratorOrgID, collaboratorUserID int32
	mustID("INSERT INTO users(username,email,password) VALUES('audit-collaborator','audit-collaborator@test.local','test') RETURNING id", &collaboratorUserID)
	mustID("INSERT INTO organizations(title,organization_type_id) VALUES('Audit collaborator org',?) RETURNING id", &collaboratorOrgID, orgTypeID)
	if len(enforcer.FilterOrganizations(ctx, userID, []string{fmt.Sprint(orgID)}, "create_event")) != 0 {
		t.Fatal("lookup exposes deleted membership")
	}
	exec("INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_at) VALUES(?,?,?,CURRENT_TIMESTAMP)", collaboratorOrgID, collaboratorUserID, roleID)
	exec("INSERT INTO event_collaborators(event_id,organization_id) VALUES(?,?)", eventID, collaboratorOrgID)
	if !enforcer.CheckResourcePermission(ctx, collaboratorUserID, "event", fmt.Sprint(eventID), "edit") {
		t.Fatal("current collaborator with create_event denied")
	}
	exec("DELETE FROM event_collaborators WHERE event_id = ?", eventID)
	if enforcer.CheckResourcePermission(ctx, collaboratorUserID, "event", fmt.Sprint(eventID), "edit") {
		t.Fatal("removed collaborator authorized from stale graph")
	}
	// A stale event graph grant cannot stand in for a current org membership.
	if enforcer.CheckResourcePermission(ctx, userID+1000, "event", fmt.Sprint(eventID), "edit") {
		t.Fatal("unrelated event editor authorized from stale graph")
	}
}
