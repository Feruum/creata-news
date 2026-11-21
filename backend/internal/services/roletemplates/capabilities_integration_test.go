//go:build integration

package roletemplates

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"google.golang.org/grpc"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type templateGraph struct {
	pb.UnimplementedPermissionsServiceServer
	mu     sync.Mutex
	writes []*pb.WriteRelationshipsRequest
}

func (*templateGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func (g *templateGraph) WriteRelationships(_ context.Context, req *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.writes = append(g.writes, req)
	return &pb.WriteRelationshipsResponse{}, nil
}
func auditTemplateDB(t *testing.T) *bun.DB {
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
	name := fmt.Sprintf("template_audit_%d", time.Now().UnixNano())
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
func TestSetTemplateCapabilitiesRollsBackOnInsertFailureAndReconcilesAllMembers(t *testing.T) {
	ctx := context.Background()
	db := auditTemplateDB(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &templateGraph{}
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	var actorID, orgTypeID, orgID, roleID int32
	mustID := func(query string, target *int32, args ...any) {
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
	mustID("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-template-actor','audit-template-actor','audit-template-actor@test.local','test') RETURNING id", &actorID)
	mustID("INSERT INTO organization_types(title) VALUES('Audit template type') RETURNING id", &orgTypeID)
	mustID("INSERT INTO organizations(title,organization_type_id) VALUES('Audit template org',?) RETURNING id", &orgID, orgTypeID)
	mustID("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'audit-template','Audit',true) RETURNING id", &roleID, orgTypeID)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", roleID)
	targetIDs := []int32{actorID}
	for i := 0; i < 2; i++ {
		var id int32
		mustID("INSERT INTO users(username,email,password) VALUES(?,?,'test') RETURNING id", &id, fmt.Sprintf("audit-target-%d", i), fmt.Sprintf("audit-target-%d@test.local", i))
		targetIDs = append(targetIDs, id)
	}
	for _, id := range targetIDs {
		exec("INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_at) VALUES(?,?,?,CURRENT_TIMESTAMP)", orgID, id, roleID)
	}
	exec("UPDATE organization_members SET valid_until=CURRENT_TIMESTAMP - INTERVAL '1 hour' WHERE user_id = ?", targetIDs[2])
	exec(`CREATE FUNCTION fail_audit_capability() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'forced capability failure'; END; $$ LANGUAGE plpgsql`)
	exec(`CREATE TRIGGER audit_capability_failure BEFORE INSERT ON role_template_capabilities FOR EACH ROW EXECUTE FUNCTION fail_audit_capability()`)
	svc := NewService(db, authz.NewEnforcer(db, client))
	ctx = context.WithValue(ctx, auth.UserIDKey, "audit-template-actor")
	request := connect.NewRequest(&eventsv1.SetTemplateCapabilitiesRequest{RoleTemplateId: roleID, Capabilities: []eventsv1.Capability{eventsv1.Capability_CAPABILITY_MANAGE_MEMBERS}})
	if _, err := svc.SetTemplateCapabilities(ctx, request); err == nil {
		t.Fatal("insert failure was reported as success")
	}
	var original string
	if err := db.NewRaw("SELECT capability FROM role_template_capabilities WHERE role_template_id = ?", roleID).Scan(ctx, &original); err != nil || original != "create_event" {
		t.Fatalf("old capability lost after rollback: %q %v", original, err)
	}
	if len(graph.snapshot()) != 0 {
		t.Fatal("graph changed before database commit")
	}
	exec("DROP TRIGGER audit_capability_failure ON role_template_capabilities")
	if _, err := svc.SetTemplateCapabilities(ctx, request); err != nil {
		t.Fatal(err)
	}
	if len(graph.snapshot()) != 3 {
		t.Fatalf("reconciled %d members, want 3", len(graph.snapshot()))
	}
	touched := map[string]bool{}
	for _, write := range graph.snapshot() {
		for _, update := range write.Updates {
			if update.Relationship.Relation == "can_manage_members" && update.Operation == pb.RelationshipUpdate_OPERATION_TOUCH {
				touched[update.Relationship.Subject.Object.ObjectId] = true
			}
		}
	}
	if !touched[fmt.Sprint(targetIDs[0])] || !touched[fmt.Sprint(targetIDs[1])] || touched[fmt.Sprint(targetIDs[2])] {
		t.Fatalf("wrong active member grants: %v", touched)
	}
}
func (g *templateGraph) snapshot() []*pb.WriteRelationshipsRequest {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]*pb.WriteRelationshipsRequest(nil), g.writes...)
}
