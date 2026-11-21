//go:build integration

package orgmembers

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"net"
	"sync"
	"testing"
	"time"
)

type overlappingMemberGraph struct {
	pb.UnimplementedPermissionsServiceServer
	mu                                        sync.Mutex
	calls                                     int
	last                                      []*pb.RelationshipUpdate
	firstStarted, secondStarted, releaseFirst chan struct{}
}

func (*overlappingMemberGraph) CheckPermission(context.Context, *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	return &pb.CheckPermissionResponse{Permissionship: pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION}, nil
}
func (g *overlappingMemberGraph) WriteRelationships(ctx context.Context, req *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	g.mu.Lock()
	g.calls++
	call := g.calls
	g.mu.Unlock()
	if call == 1 {
		close(g.firstStarted)
		select {
		case <-g.releaseFirst:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if call == 2 {
		close(g.secondStarted)
	}
	g.mu.Lock()
	g.last = req.Updates
	g.mu.Unlock()
	return &pb.WriteRelationshipsResponse{}, nil
}
func TestOverlappingMemberSyncCannotOverwriteNewAssignment(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	insert := func(q string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(q, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	actor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES('audit-overlap-actor','audit-overlap-actor','audit-overlap-actor@test.local','test') RETURNING id")
	target := insert("INSERT INTO users(username,email,password) VALUES('audit-overlap-target','audit-overlap-target@test.local','test') RETURNING id")
	typ := insert("INSERT INTO organization_types(title) VALUES('Audit overlap type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit overlap org',?) RETURNING id", typ)
	oldRole := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'old-overlap','Old',true) RETURNING id", typ)
	newRole := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'new-overlap','New',true) RETURNING id", typ)
	exec("INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event'),(?,'manage_members')", oldRole, newRole)
	member := insert("INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_by) VALUES(?,?,?,?) RETURNING id", org, target, oldRole, actor)
	graph := &overlappingMemberGraph{firstStarted: make(chan struct{}), secondStarted: make(chan struct{}), releaseFirst: make(chan struct{})}
	var release sync.Once
	defer release.Do(func() { close(graph.releaseFirst) })
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	e := authz.NewEnforcer(db, client)
	s := NewService(db, client, e, nil)
	oldFinished := make(chan error, 1)
	go func() { oldFinished <- e.SyncOrganizationMember(ctx, org, target) }()
	select {
	case <-graph.firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("old sync did not start")
	}
	updated := make(chan error, 1)
	go func() {
		_, err := s.UpdateOrganizationMember(context.WithValue(ctx, auth.UserIDKey, "audit-overlap-actor"), connect.NewRequest(&eventsv1.UpdateOrganizationMemberRequest{Id: member, RoleTemplateId: &newRole}))
		updated <- err
	}()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	overlapping := false
	newCompletedFirst := false
	for !overlapping {
		select {
		case <-graph.secondStarted:
			newCompletedFirst = true
			overlapping = true
		case <-ticker.C:
			var waiting bool
			if err := db.NewRaw("SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%pg_advisory_xact_lock%')").Scan(ctx, &waiting); err != nil {
				t.Fatal(err)
			}
			overlapping = waiting
		case <-deadline.C:
			t.Fatal("neither overlapping write nor serialized advisory wait observed")
		}
	}
	if newCompletedFirst {
		if err := <-updated; err != nil {
			t.Fatal(err)
		}
	}
	release.Do(func() { close(graph.releaseFirst) })
	if err := <-oldFinished; err != nil {
		t.Fatal(err)
	}
	if !newCompletedFirst {
		if err := <-updated; err != nil {
			t.Fatal(err)
		}
	}
	graph.mu.Lock()
	defer graph.mu.Unlock()
	for _, update := range graph.last {
		if update.Relationship.Relation == perms.CapabilityManageMembers && update.Operation == pb.RelationshipUpdate_OPERATION_TOUCH {
			return
		}
	}
	t.Fatal(fmt.Sprintf("older role overwrote newer membership projection after %d writes", graph.calls))
}

func TestDelayedRemovalSyncPreservesReaddedMembership(t *testing.T) {
	db := testsupport.ClonedDB(t)
	ctx := context.Background()
	insert := func(query string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(query, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	user := insert("INSERT INTO users(username,email,password) VALUES('audit-readded-user','audit-readded-user@test.local','test') RETURNING id")
	typ := insert("INSERT INTO organization_types(title) VALUES('Audit readded type') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Audit readded org',?) RETURNING id", typ)
	role := insert("INSERT INTO role_templates(organization_type_id,key,display_name,is_active) VALUES(?,'readded','Readded',true) RETURNING id", typ)
	if _, err := db.ExecContext(ctx, "INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'manage_members')", role); err != nil {
		t.Fatal(err)
	}
	// A previous removal has already committed, then the same user is re-added
	// before that removal's immediate graph call finally executes.
	insert("INSERT INTO organization_members(organization_id,user_id,role_template_id) VALUES(?,?,?) RETURNING id", org, user, role)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &delayedRemovalGraph{}
	pb.RegisterPermissionsServiceServer(server, graph)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(db, client, authz.NewEnforcer(db, client), nil)
	if err := service.removeMemberFromSpiceDB(ctx, org, user); err != nil {
		t.Fatal(err)
	}
	graph.mu.Lock()
	defer graph.mu.Unlock()
	for _, update := range graph.updates {
		if update.Relationship.Relation == perms.CapabilityManageMembers && update.Operation == pb.RelationshipUpdate_OPERATION_TOUCH {
			return
		}
	}
	t.Fatal("delayed removal erased the re-added member's current capabilities")
}

type delayedRemovalGraph struct {
	pb.UnimplementedPermissionsServiceServer
	mu      sync.Mutex
	updates []*pb.RelationshipUpdate
}

func (g *delayedRemovalGraph) WriteRelationships(_ context.Context, request *pb.WriteRelationshipsRequest) (*pb.WriteRelationshipsResponse, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.updates = request.Updates
	return &pb.WriteRelationshipsResponse{}, nil
}
