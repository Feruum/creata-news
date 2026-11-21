package agentaccess

import (
	"context"
	"github.com/studyverse/ems-backend/internal/perms"
	"testing"
)

type graphFixture struct {
	grants  []perms.Relationship
	session []perms.Relationship
}

func (*graphFixture) CheckPermission(context.Context, string, string, string, string) (bool, error) {
	return true, nil
}
func (*graphFixture) CreateOwnerRelationship(context.Context, string, string, string) error {
	return nil
}
func (*graphFixture) WriteRelationships(context.Context, []perms.Relationship) error { return nil }
func (g *graphFixture) ReadRelationships(_ context.Context, resource, id string) ([]perms.Relationship, error) {
	if resource == "agent_session" {
		return g.session, nil
	}
	return g.grants, nil
}

type orgFixture struct {
	current     bool
	permissions []string
}

func (o *orgFixture) CheckOrgPermission(_ context.Context, userID, orgID int32, permission string) bool {
	o.permissions = append(o.permissions, permission)
	return o.current
}
func TestDocumentInheritanceRequiresCurrentOrganizationPermission(t *testing.T) {
	graph := &graphFixture{grants: []perms.Relationship{{Relation: "parent_organization", SubjectType: "organization", SubjectID: "3"}}}
	orgs := &orgFixture{}
	service := &Service{permissions: graph, organizations: orgs}
	for _, permission := range []string{"view", "edit", "delete", "share"} {
		allowed, err := service.checkPermission(context.Background(), "7", "document", "doc", permission)
		if err != nil || allowed {
			t.Fatalf("stale organization %s grant allowed: %v %v", permission, allowed, err)
		}
	}
	orgs.current = true
	for _, permission := range []string{"view", "edit", "delete", "share"} {
		allowed, err := service.checkPermission(context.Background(), "7", "document", "doc", permission)
		if err != nil || !allowed {
			t.Fatalf("current organization grant denied: %s %v", permission, err)
		}
	}
}
func TestDirectDocumentGrantsSurviveOrganizationRevocation(t *testing.T) {
	for _, test := range []struct {
		relation, permission string
		allowed              bool
	}{{"owner", "delete", true}, {"editor", "edit", true}, {"viewer", "view", true}, {"viewer", "edit", false}, {"editor", "delete", false}} {
		graph := &graphFixture{grants: []perms.Relationship{{Relation: test.relation, SubjectType: "user", SubjectID: "7"}}}
		service := &Service{permissions: graph, organizations: &orgFixture{}}
		allowed, err := service.checkPermission(context.Background(), "7", "document", "doc", test.permission)
		if err != nil || allowed != test.allowed {
			t.Fatalf("%s %s grant: %v %v", test.relation, test.permission, allowed, err)
		}
	}
}
func TestSessionDocumentUseCannotBypassRevokedDocumentMembership(t *testing.T) {
	graph := &graphFixture{grants: []perms.Relationship{{Relation: "parent_organization", SubjectType: "organization", SubjectID: "3"}}, session: []perms.Relationship{{Relation: "source_document", SubjectType: "document", SubjectID: "doc"}}}
	orgs := &orgFixture{}
	service := &Service{permissions: graph, organizations: orgs}
	allowed, err := service.checkPermission(context.Background(), "7", "agent_session", "session", "use_document")
	if err != nil || allowed {
		t.Fatalf("session bypasses revoked membership: %v %v", allowed, err)
	}
	graph.grants = append(graph.grants, perms.Relationship{Relation: "viewer", SubjectType: "user", SubjectID: "7"})
	allowed, err = service.checkPermission(context.Background(), "7", "agent_session", "session", "use_document")
	if err != nil || !allowed {
		t.Fatalf("session cannot use explicitly shared document: %v %v", allowed, err)
	}
}
func TestNilPermissionClientFailsClosed(t *testing.T) {
	service := NewService(nil, nil)
	allowed, err := service.checkPermission(context.Background(), "7", "document", "doc", "view")
	if err == nil || allowed {
		t.Fatal("missing authorization service accepted")
	}
}
