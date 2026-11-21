//go:build integration

package roletemplates

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
)

type roleTemplateMutationFixture struct {
	ctx         context.Context
	db          *bun.DB
	service     *Service
	graph       *templateGraph
	actorID     int32
	typeID      int32
	otherTypeID int32
	orgID       int32
	otherOrgID  int32
}

func newRoleTemplateMutationFixture(t *testing.T) *roleTemplateMutationFixture {
	t.Helper()
	db := auditTemplateDB(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	graph := &templateGraph{}
	pb.RegisterPermissionsServiceServer(server, graph)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	client, err := perms.NewClient(listener.Addr().String(), "test", true, false, "default")
	if err != nil {
		t.Fatal(err)
	}
	actor := fmt.Sprintf("role-mutation-actor-%d", time.Now().UnixNano())
	fixture := &roleTemplateMutationFixture{
		ctx:     context.WithValue(context.Background(), auth.UserIDKey, actor),
		db:      db,
		service: NewService(db, authz.NewEnforcer(db, client)),
		graph:   graph,
	}
	fixture.mustID(t, "INSERT INTO users(kratos_id,username,email,password) VALUES(?,?,?,'test') RETURNING id", &fixture.actorID, actor, actor, actor+"@test.local")
	fixture.mustID(t, "INSERT INTO organization_types(title) VALUES('Role mutation type') RETURNING id", &fixture.typeID)
	fixture.mustID(t, "INSERT INTO organization_types(title) VALUES('Other role mutation type') RETURNING id", &fixture.otherTypeID)
	fixture.mustID(t, "INSERT INTO organizations(title,organization_type_id) VALUES('Role mutation organization',?) RETURNING id", &fixture.orgID, fixture.typeID)
	fixture.mustID(t, "INSERT INTO organizations(title,organization_type_id) VALUES('Other role mutation organization',?) RETURNING id", &fixture.otherOrgID, fixture.typeID)
	return fixture
}

func (fixture *roleTemplateMutationFixture) mustID(t *testing.T, query string, target *int32, args ...any) {
	t.Helper()
	if err := fixture.db.NewRaw(query, args...).Scan(fixture.ctx, target); err != nil {
		t.Fatal(err)
	}
}

func (fixture *roleTemplateMutationFixture) insertTemplate(t *testing.T, key string, typeID int32, orgID, parentID, maxHolders *int32) int32 {
	t.Helper()
	var id int32
	fixture.mustID(t, `INSERT INTO role_templates(organization_type_id,organization_id,key,display_name,parent_template_id,max_holders,is_active)
		VALUES(?,?,?,?,?,?,true) RETURNING id`, &id, typeID, orgID, key, key, parentID, maxHolders)
	return id
}

func (fixture *roleTemplateMutationFixture) loadTemplate(t *testing.T, id int32) models.RoleTemplate {
	t.Helper()
	var template models.RoleTemplate
	if err := fixture.db.NewSelect().Model(&template).Where("id = ?", id).Scan(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	return template
}

func requireRoleTemplateInt32(t *testing.T, field string, got, want *int32) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("%s null state differs: got %v, want %v", field, got, want)
		}
		return
	}
	if *got != *want {
		t.Fatalf("%s got %d, want %d", field, *got, *want)
	}
}

func TestUpdateRoleTemplatePersistsNullableFields(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	parentID := fixture.insertTemplate(t, "nullable-parent", fixture.typeID, nil, nil, nil)
	limit, replacement := int32(3), int32(4)
	name := "Updated role"
	for _, test := range []*struct {
		name       string
		request    eventsv1.UpdateRoleTemplateRequest
		wantParent *int32
		wantLimit  *int32
	}{
		{name: "old request preserves omitted values", request: eventsv1.UpdateRoleTemplateRequest{DisplayName: &name}, wantParent: &parentID, wantLimit: &limit},
		{name: "empty request preserves omitted values", wantParent: &parentID, wantLimit: &limit},
		{name: "clear only parent", request: eventsv1.UpdateRoleTemplateRequest{ClearParentTemplate: true}, wantLimit: &limit},
		{name: "clear only limit", request: eventsv1.UpdateRoleTemplateRequest{ClearMaxHolders: true}, wantParent: &parentID},
		{name: "clear both", request: eventsv1.UpdateRoleTemplateRequest{ClearParentTemplate: true, ClearMaxHolders: true}},
		{name: "set values", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &parentID, MaxHolders: &replacement}, wantParent: &parentID, wantLimit: &replacement},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := fixture.insertTemplate(t, test.name, fixture.typeID, &fixture.orgID, &parentID, &limit)
			if _, err := fixture.db.ExecContext(fixture.ctx, "INSERT INTO role_template_capabilities(role_template_id,capability) VALUES(?,'create_event')", id); err != nil {
				t.Fatal(err)
			}
			test.request.Id = id
			response, err := fixture.service.UpdateRoleTemplate(fixture.ctx, connect.NewRequest(&test.request))
			if err != nil {
				t.Fatal(err)
			}
			stored := fixture.loadTemplate(t, id)
			requireRoleTemplateInt32(t, "stored parent", stored.ParentTemplateID, test.wantParent)
			requireRoleTemplateInt32(t, "stored limit", stored.MaxHolders, test.wantLimit)
			requireRoleTemplateInt32(t, "response parent", response.Msg.RoleTemplate.ParentTemplateId, test.wantParent)
			requireRoleTemplateInt32(t, "response limit", response.Msg.RoleTemplate.MaxHolders, test.wantLimit)
			if test.request.DisplayName != nil && stored.DisplayName != name {
				t.Fatal("old request did not update its display name")
			}
			if caps := response.Msg.RoleTemplate.Capabilities; len(caps) != 1 || caps[0] != eventsv1.Capability_CAPABILITY_CREATE_EVENT {
				t.Fatalf("template capabilities changed: %v", caps)
			}
		})
	}
	if len(fixture.graph.snapshot()) != 0 {
		t.Fatal("nullable-field updates unexpectedly synchronized capabilities")
	}
}

func TestUpdateRoleTemplateRejectsInvalidParentChangesAtomically(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	globalID := fixture.insertTemplate(t, "parent-global", fixture.typeID, nil, nil, nil)
	localID := fixture.insertTemplate(t, "parent-local", fixture.typeID, &fixture.orgID, nil, nil)
	otherOrgID := fixture.insertTemplate(t, "parent-other-org", fixture.typeID, &fixture.otherOrgID, nil, nil)
	otherTypeID := fixture.insertTemplate(t, "parent-other-type", fixture.otherTypeID, nil, nil, nil)
	missingID, zero, negative, limit := int32(2147483647), int32(0), int32(-1), int32(3)
	for _, test := range []*struct {
		name    string
		global  bool
		request eventsv1.UpdateRoleTemplateRequest
	}{
		{name: "parent clear conflict", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &globalID, ClearParentTemplate: true}},
		{name: "limit clear conflict", request: eventsv1.UpdateRoleTemplateRequest{MaxHolders: &limit, ClearMaxHolders: true}},
		{name: "zero parent", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &zero}},
		{name: "negative parent", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &negative}},
		{name: "zero limit", request: eventsv1.UpdateRoleTemplateRequest{MaxHolders: &zero}},
		{name: "missing parent", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &missingID}},
		{name: "other organization", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &otherOrgID}},
		{name: "other organization type", request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &otherTypeID}},
		{name: "global with local parent", global: true, request: eventsv1.UpdateRoleTemplateRequest{ParentTemplateId: &localID}},
		{name: "self parent"},
		{name: "ancestor cycle"},
	} {
		t.Run(test.name, func(t *testing.T) {
			orgID := &fixture.orgID
			if test.global {
				orgID = nil
			}
			id := fixture.insertTemplate(t, test.name, fixture.typeID, orgID, &globalID, &limit)
			test.request.Id = id
			name := "Must roll back"
			test.request.DisplayName = &name
			if test.name == "self parent" {
				test.request.ParentTemplateId = &id
			}
			if test.name == "ancestor cycle" {
				descendant := fixture.insertTemplate(t, "descendant", fixture.typeID, orgID, &id, nil)
				test.request.ParentTemplateId = &descendant
			}
			if _, err := fixture.service.UpdateRoleTemplate(fixture.ctx, connect.NewRequest(&test.request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("invalid parent update got %v", err)
			}
			stored := fixture.loadTemplate(t, id)
			requireRoleTemplateInt32(t, "parent after rejected update", stored.ParentTemplateID, &globalID)
			requireRoleTemplateInt32(t, "limit after rejected update", stored.MaxHolders, &limit)
			if stored.DisplayName != test.name {
				t.Fatal("invalid request changed the display name")
			}
		})
	}
}

func TestCreateRoleTemplateValidatesParentScope(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	globalID := fixture.insertTemplate(t, "create-global", fixture.typeID, nil, nil, nil)
	localID := fixture.insertTemplate(t, "create-local", fixture.typeID, &fixture.orgID, nil, nil)
	otherOrgID := fixture.insertTemplate(t, "create-other-org", fixture.typeID, &fixture.otherOrgID, nil, nil)
	otherTypeID := fixture.insertTemplate(t, "create-other-type", fixture.otherTypeID, nil, nil, nil)
	cycleA := fixture.insertTemplate(t, "create-cycle-a", fixture.typeID, nil, nil, nil)
	cycleB := fixture.insertTemplate(t, "create-cycle-b", fixture.typeID, nil, &cycleA, nil)
	if _, err := fixture.db.ExecContext(fixture.ctx, "UPDATE role_templates SET parent_template_id = ? WHERE id = ?", cycleB, cycleA); err != nil {
		t.Fatal(err)
	}
	zero, negative, missingID := int32(0), int32(-1), int32(2147483647)
	for _, test := range []struct {
		name     string
		orgID    *int32
		parentID *int32
		invalid  bool
	}{
		{name: "global uses global parent", parentID: &globalID},
		{name: "local uses global parent", orgID: &fixture.orgID, parentID: &globalID},
		{name: "local uses same organization parent", orgID: &fixture.orgID, parentID: &localID},
		{name: "no parent", orgID: &fixture.orgID},
		{name: "global cannot use local", parentID: &localID, invalid: true},
		{name: "different organization", orgID: &fixture.orgID, parentID: &otherOrgID, invalid: true},
		{name: "different organization type", orgID: &fixture.orgID, parentID: &otherTypeID, invalid: true},
		{name: "ancestor cycle", orgID: &fixture.orgID, parentID: &cycleA, invalid: true},
		{name: "zero parent", orgID: &fixture.orgID, parentID: &zero, invalid: true},
		{name: "negative parent", orgID: &fixture.orgID, parentID: &negative, invalid: true},
		{name: "missing parent", orgID: &fixture.orgID, parentID: &missingID, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := &eventsv1.CreateRoleTemplateRequest{OrganizationTypeId: fixture.typeID, OrganizationId: test.orgID, Key: test.name, DisplayName: test.name, ParentTemplateId: test.parentID, IsActive: true, Capabilities: []eventsv1.Capability{eventsv1.Capability_CAPABILITY_CREATE_EVENT}}
			response, err := fixture.service.CreateRoleTemplate(fixture.ctx, connect.NewRequest(request))
			if test.invalid {
				if connect.CodeOf(err) != connect.CodeInvalidArgument {
					t.Fatalf("invalid create got %v", err)
				}
				var count int
				if err := fixture.db.NewRaw("SELECT COUNT(*) FROM role_templates WHERE organization_type_id = ? AND key = ?", fixture.typeID, test.name).Scan(fixture.ctx, &count); err != nil || count != 0 {
					t.Fatalf("invalid create persisted a role: count=%d err=%v", count, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			stored := fixture.loadTemplate(t, response.Msg.RoleTemplate.Id)
			requireRoleTemplateInt32(t, "created parent", stored.ParentTemplateID, test.parentID)
			if len(response.Msg.RoleTemplate.Capabilities) != 1 {
				t.Fatal("create did not preserve capabilities")
			}
		})
	}
}

func TestUpdateRoleTemplateSerializesConcurrentParentChanges(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	first := fixture.insertTemplate(t, "concurrent-first", fixture.typeID, &fixture.orgID, nil, nil)
	second := fixture.insertTemplate(t, "concurrent-second", fixture.typeID, &fixture.orgID, nil, nil)
	tx, err := fixture.db.BeginTx(fixture.ctx, &sql.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := lockRoleTemplateHierarchy(fixture.ctx, tx, fixture.typeID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(fixture.ctx, 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := fixture.service.UpdateRoleTemplate(ctx, connect.NewRequest(&eventsv1.UpdateRoleTemplateRequest{Id: second, ParentTemplateId: &first}))
		done <- err
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
waitForHierarchyLock:
	for {
		select {
		case err := <-done:
			t.Fatalf("parent update escaped hierarchy serialization: %v", err)
		case <-ticker.C:
			var waiting bool
			if err := fixture.db.NewRaw(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database() AND wait_event_type='Lock'
				AND query LIKE '%pg_advisory_xact_lock%')`).Scan(ctx, &waiting); err != nil {
				t.Fatal(err)
			}
			if waiting {
				break waitForHierarchyLock
			}
		case <-ctx.Done():
			t.Fatal("parent update did not wait on the hierarchy lock")
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE role_templates SET parent_template_id = ? WHERE id = ?", second, first); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("waiting update did not reject the newly committed cycle: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("waiting parent update did not finish")
	}
	storedFirst := fixture.loadTemplate(t, first)
	storedSecond := fixture.loadTemplate(t, second)
	requireRoleTemplateInt32(t, "first parent", storedFirst.ParentTemplateID, &second)
	requireRoleTemplateInt32(t, "second parent", storedSecond.ParentTemplateID, nil)
}

func TestUpdateRoleTemplateRejectsOtherTypeWithoutLockingIt(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	id := fixture.insertTemplate(t, "scoped-lock-child", fixture.typeID, &fixture.orgID, nil, nil)
	otherTypeID := fixture.insertTemplate(t, "scoped-lock-parent", fixture.otherTypeID, nil, nil, nil)
	tx, err := fixture.db.BeginTx(fixture.ctx, &sql.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	var lockedID int32
	if err := tx.NewRaw("SELECT id FROM role_templates WHERE id = ? FOR UPDATE", otherTypeID).Scan(fixture.ctx, &lockedID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(fixture.ctx, time.Second)
	defer cancel()
	_, err = fixture.service.UpdateRoleTemplate(ctx, connect.NewRequest(&eventsv1.UpdateRoleTemplateRequest{Id: id, ParentTemplateId: &otherTypeID}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("cross-type parent waited on an unrelated row instead of being rejected: %v", err)
	}
}

func TestUpdateRoleTemplateClearKeepsActivationReconciliation(t *testing.T) {
	fixture := newRoleTemplateMutationFixture(t)
	parent := fixture.insertTemplate(t, "activation-parent", fixture.typeID, nil, nil, nil)
	limit := int32(1)
	id := fixture.insertTemplate(t, "activation-role", fixture.typeID, &fixture.orgID, &parent, &limit)
	if _, err := fixture.db.ExecContext(fixture.ctx, "INSERT INTO organization_members(organization_id,user_id,role_template_id,assigned_at) VALUES(?,?,?,CURRENT_TIMESTAMP)", fixture.orgID, fixture.actorID, id); err != nil {
		t.Fatal(err)
	}
	inactive := false
	response, err := fixture.service.UpdateRoleTemplate(fixture.ctx, connect.NewRequest(&eventsv1.UpdateRoleTemplateRequest{Id: id, ClearParentTemplate: true, ClearMaxHolders: true, IsActive: &inactive}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.RoleTemplate.IsActive || response.Msg.RoleTemplate.ParentTemplateId != nil || response.Msg.RoleTemplate.MaxHolders != nil {
		t.Fatalf("combined clear and deactivation was not saved: %+v", response.Msg.RoleTemplate)
	}
	if len(fixture.graph.snapshot()) != 1 {
		t.Fatalf("activation change reconciled %d members, want 1", len(fixture.graph.snapshot()))
	}
}
