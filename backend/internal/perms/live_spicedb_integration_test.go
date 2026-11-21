//go:build integration

package perms_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// This suite writes the real schema and must only target an isolated disposable server.
func TestLiveSpiceDBCurrentAuthorization(t *testing.T) {
	if os.Getenv("RUN_SPICEDB_INTEGRATION_TESTS") != "1" {
		t.Skip("explicit disposable SpiceDB opt-in required")
	}
	endpoint, key := os.Getenv("SPICEDB_TEST_ENDPOINT"), os.Getenv("SPICEDB_TEST_PRESHARED_KEY")
	if endpoint == "" || key == "" || os.Getenv("DATABASE_URL") == "" {
		t.Fatal("live suite requires test endpoint/key and disposable migrated DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schemaPath := os.Getenv("SPICEDB_TEST_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = filepath.Join("..", "..", "..", "..", "infra", "spicedb", "schema.zed")
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := authzed.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpcutil.WithInsecureBearerToken(key))
	if err != nil {
		t.Fatal(err)
	}
	// Writing the schema also provides a bounded readiness check for a newly started server.
	deadline := time.Now().Add(20 * time.Second)
	for {
		_, err = raw.WriteSchema(ctx, &pb.WriteSchemaRequest{Schema: string(schema)})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("write production schema: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	platformID := fmt.Sprintf("audit-%d", time.Now().UnixNano())
	client, err := perms.NewClient(endpoint, key, true, false, platformID)
	if err != nil {
		t.Fatal(err)
	}
	db := testsupport.ClonedDB(t)
	insert := func(model interface{}) {
		t.Helper()
		if _, err := db.NewInsert().Model(model).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	kind := &models.OrganizationType{Title: platformID}
	insert(kind)
	// SQL clones share sequence starting points, but SpiceDB retains tuples.
	// Distinct positive int32 fixture IDs prevent earlier tests' graph grants
	// from accidentally becoming this test's users or host organization.
	base := int32(1_000_000_000) + rand.Int32N(900_000_000)
	org := &models.Organization{ID: base, Title: platformID, OrganizationTypeID: kind.ID}
	insert(org)
	creator := &models.User{ID: base, Username: platformID + "-creator", Email: platformID + "-creator@test.invalid", Password: "test-only"}
	member := &models.User{ID: base + 1, Username: platformID + "-member", Email: platformID + "-member@test.invalid", Password: "test-only"}
	admin := &models.User{ID: base + 2, Username: platformID + "-admin", Email: platformID + "-admin@test.invalid", Password: "test-only"}
	stranger := &models.User{ID: base + 3, Username: platformID + "-stranger", Email: platformID + "-stranger@test.invalid", Password: "test-only"}
	insert(creator)
	insert(member)
	insert(admin)
	insert(stranger)
	role := &models.RoleTemplate{OrganizationTypeID: kind.ID, Key: platformID, DisplayName: "Test editor", IsActive: true}
	insert(role)
	capability := &models.RoleTemplateCapability{RoleTemplateID: role.ID, Capability: models.Capability("create_event")}
	insert(capability)
	membership := &models.OrganizationMember{OrganizationID: org.ID, UserID: member.ID, RoleTemplateID: role.ID, AssignedAt: time.Now()}
	insert(membership)
	event := &models.Event{ID: base, Title: platformID, UserID: creator.ID, OrganizationID: org.ID, Description: json.RawMessage(`{}`), StructuredLocation: json.RawMessage(`{}`), StartTime: time.Now(), EndTime: time.Now().Add(time.Hour), Status: models.EventStatus("draft")}
	event.CheckInSecret = &platformID
	token := fmt.Sprintf("%x", time.Now().UnixNano())
	event.CheckInToken = &token
	insert(event)
	uid, mid, oid, eid := fmt.Sprint(creator.ID), fmt.Sprint(member.ID), fmt.Sprint(org.ID), fmt.Sprint(event.ID)
	registrationID := platformID + "-registration"
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for resource, id := range map[string]string{"platform": platformID, "organization": oid, "event": eid, "event_registration": registrationID} {
			if _, err := raw.DeleteRelationships(cleanupCtx, &pb.DeleteRelationshipsRequest{
				RelationshipFilter: &pb.RelationshipFilter{ResourceType: resource, OptionalResourceId: id},
			}); err != nil {
				t.Errorf("clean up owned %s:%s: %v", resource, id, err)
			}
		}
	})
	graph := func(t *testing.T, user, resource, id, permission string, expected bool) {
		t.Helper()
		allowed, err := client.CheckPermission(ctx, user, resource, id, permission)
		if err != nil || allowed != expected {
			t.Fatalf("%s %s:%s %s = %v, want %v: %v", user, resource, id, permission, allowed, expected, err)
		}
	}
	if err := client.SetupEventRelationships(ctx, eid, oid, uid); err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, client)
	t.Run("numeric ordinary creator can edit delete and manage registrations", func(t *testing.T) {
		for _, permission := range []string{"edit", "delete", "manage_registrations", "check_in"} {
			graph(t, uid, "event", eid, permission, true)
			graph(t, mid, "event", eid, permission, false)
			if !enforcer.CheckResourcePermission(ctx, creator.ID, "event", eid, permission) {
				t.Fatalf("ordinary creator denied %s", permission)
			}
			if enforcer.CheckResourcePermission(ctx, member.ID, "event", eid, permission) {
				t.Fatalf("unrelated ordinary user allowed %s", permission)
			}
		}
		graph(t, uid, "platform", platformID, "manage_system", false)
		graph(t, uid, "organization", oid, "create_event", false)
	})
	t.Run("administrator retains event and registration access", func(t *testing.T) {
		if err := client.LinkOrganizationToPlatform(ctx, oid); err != nil {
			t.Fatal(err)
		}
		if err := enforcer.AssignPlatformRole(ctx, admin.ID, "administrator"); err != nil {
			t.Fatal(err)
		}
		for _, permission := range []string{"edit", "delete", "manage_registrations", "check_in"} {
			graph(t, fmt.Sprint(admin.ID), "event", eid, permission, true)
			if !enforcer.CheckResourcePermission(ctx, admin.ID, "event", eid, permission) {
				t.Fatalf("administrator denied %s", permission)
			}
		}
		if err := client.WriteRelationships(ctx, []perms.Relationship{
			{Resource: "event_registration", ResourceID: registrationID, Relation: "parent_event", SubjectType: "event", SubjectID: eid},
			{Resource: "event_registration", ResourceID: registrationID, Relation: "registrant", SubjectType: "user", SubjectID: mid},
		}); err != nil {
			t.Fatal(err)
		}
		for _, permission := range []string{"view", "cancel"} {
			graph(t, mid, "event_registration", registrationID, permission, true)
			graph(t, uid, "event_registration", registrationID, permission, true)
			graph(t, fmt.Sprint(admin.ID), "event_registration", registrationID, permission, true)
			graph(t, fmt.Sprint(stranger.ID), "event_registration", registrationID, permission, false)
		}
		graph(t, mid, "event_registration", registrationID, "check_in", false)
		graph(t, uid, "event_registration", registrationID, "check_in", true)
		graph(t, fmt.Sprint(admin.ID), "event_registration", registrationID, "check_in", true)
		graph(t, fmt.Sprint(stranger.ID), "event_registration", registrationID, "check_in", false)
	})
	t.Run("legacy UUID tuple does not replace authoritative numeric creator", func(t *testing.T) {
		numeric := perms.Relationship{Resource: "event", ResourceID: eid, Relation: "creator", SubjectType: "user", SubjectID: uid}
		legacy := numeric
		legacy.SubjectID = "58996d9d-218d-42ec-81f2-9d088f4eebd4"
		if err := client.DeleteRelationships(ctx, []perms.Relationship{numeric}); err != nil {
			t.Fatal(err)
		}
		if err := client.WriteRelationships(ctx, []perms.Relationship{legacy}); err != nil {
			t.Fatal(err)
		}
		for _, permission := range []string{"edit", "delete"} {
			graph(t, uid, "event", eid, permission, false)
			if !enforcer.CheckResourcePermission(ctx, creator.ID, "event", eid, permission) {
				t.Fatalf("database creator denied with legacy graph tuple: %s", permission)
			}
			if enforcer.CheckResourcePermission(ctx, stranger.ID, "event", eid, permission) {
				t.Fatalf("unrelated user allowed by legacy graph tuple: %s", permission)
			}
		}
		// Explicitly touching the authoritative tuple before deleting the exact
		// legacy tuple is a repeatable repair; it cannot revoke another creator.
		for i := 0; i < 2; i++ {
			if err := client.WriteRelationships(ctx, []perms.Relationship{numeric}); err != nil {
				t.Fatal(err)
			}
			if err := client.DeleteRelationships(ctx, []perms.Relationship{legacy}); err != nil {
				t.Fatal(err)
			}
		}
		graph(t, uid, "event", eid, "edit", true)
		graph(t, uid, "event", eid, "delete", true)
		graph(t, legacy.SubjectID, "event", eid, "edit", false)
	})
	t.Run("current SQL capability grants host event edit", func(t *testing.T) {
		if err := enforcer.SyncOrganizationMember(ctx, org.ID, member.ID); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "organization", oid, "create_event", true)
		graph(t, mid, "event", eid, "edit", true)
		if !enforcer.CheckOrgPermission(ctx, member.ID, org.ID, "create_event") || !enforcer.CheckResourcePermission(ctx, member.ID, "event", eid, "edit") {
			t.Fatal("current member denied")
		}
	})
	t.Run("atomic capability replacement removes old grants", func(t *testing.T) {
		if err := client.SetOrganizationMember(ctx, oid, mid, []string{perms.CapabilityManageSettings}); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "organization", oid, "manage_settings", true)
		graph(t, mid, "organization", oid, "create_event", false)
		graph(t, mid, "event", eid, "edit", false)
		if err := enforcer.SyncOrganizationMember(ctx, org.ID, member.ID); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "organization", oid, "manage_settings", false)
		graph(t, mid, "organization", oid, "create_event", true)
	})
	denyCurrent := func(t *testing.T) {
		t.Helper()
		graph(t, mid, "organization", oid, "create_event", true)
		graph(t, mid, "event", eid, "edit", true)
		if enforcer.CheckOrgPermission(ctx, member.ID, org.ID, "create_event") || enforcer.CheckResourcePermission(ctx, member.ID, "event", eid, "edit") {
			t.Fatal("stale graph overrode revoked SQL grant")
		}
	}
	t.Run("removed SQL capability denies stale graph", func(t *testing.T) {
		if _, err := db.NewDelete().Model(capability).WherePK().Exec(ctx); err != nil {
			t.Fatal(err)
		}
		denyCurrent(t)
		capability.ID = 0
		insert(capability)
	})
	t.Run("expired SQL membership denies stale graph", func(t *testing.T) {
		expired := time.Now().Add(-time.Hour)
		if _, err := db.NewUpdate().Model(membership).Set("valid_until = ?", expired).WherePK().Exec(ctx); err != nil {
			t.Fatal(err)
		}
		denyCurrent(t)
	})
	t.Run("deleted SQL membership denies stale graph", func(t *testing.T) {
		if _, err := db.NewDelete().Model(membership).WherePK().Exec(ctx); err != nil {
			t.Fatal(err)
		}
		denyCurrent(t)
		if err := enforcer.SyncOrganizationMember(ctx, org.ID, member.ID); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "event", eid, "edit", false)
	})
	t.Run("atomic platform role replacement removes administrator grant", func(t *testing.T) {
		if err := client.SetPlatformRole(ctx, mid, "administrator"); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "platform", platformID, "manage_system", true)
		if err := client.SetPlatformRole(ctx, mid, "staff"); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "platform", platformID, "manage_system", false)
		graph(t, mid, "platform", platformID, "manage_organizations", true)
		if err := client.SetPlatformRole(ctx, mid, ""); err != nil {
			t.Fatal(err)
		}
		graph(t, mid, "platform", platformID, "manage_organizations", false)
	})
}
