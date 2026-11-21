//go:build integration

package events

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/google/uuid"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Exercise the real writer and policy after removing the creator's organization
// grant, so inherited permissions cannot mask a UUID/numeric namespace regression.
func TestCreateEventUsesNumericCreatorAndPreservesMutationPermissions(t *testing.T) {
	if os.Getenv("RUN_SPICEDB_INTEGRATION_TESTS") != "1" {
		t.Skip("explicit disposable SpiceDB opt-in required")
	}
	endpoint, key := os.Getenv("SPICEDB_TEST_ENDPOINT"), os.Getenv("SPICEDB_TEST_PRESHARED_KEY")
	if endpoint == "" || key == "" || os.Getenv("DATABASE_URL") == "" {
		t.Fatal("test requires disposable SpiceDB endpoint/key and migrated DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	schemaPath := os.Getenv("SPICEDB_TEST_SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = filepath.Join("..", "..", "..", "..", "..", "infra", "spicedb", "schema.zed")
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := authzed.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpcutil.WithInsecureBearerToken(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.WriteSchema(ctx, &pb.WriteSchemaRequest{Schema: string(schema)}); err != nil {
		t.Fatal(err)
	}
	platformID := fmt.Sprintf("creator-namespace-%d", time.Now().UnixNano())
	graph, err := perms.NewClient(endpoint, key, true, false, platformID)
	if err != nil {
		t.Fatal(err)
	}
	db := testsupport.ClonedDB(t)
	insert := func(model interface{}) {
		t.Helper()
		if _, err := db.NewInsert().Model(model).Returning("*").Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Distinct positive int32 IDs prevent tuples left by other tests on the same
	// disposable SpiceDB server from accidentally granting these fixture users.
	base := int32(1_000_000_000) + rand.Int32N(900_000_000)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for resource, ids := range map[string][]string{
			"platform": {platformID}, "organization": {fmt.Sprint(base)}, "event": {fmt.Sprint(base), fmt.Sprint(base + 1)},
		} {
			for _, id := range ids {
				if _, err := raw.DeleteRelationships(cleanupCtx, &pb.DeleteRelationshipsRequest{
					RelationshipFilter: &pb.RelationshipFilter{ResourceType: resource, OptionalResourceId: id},
				}); err != nil {
					t.Errorf("clean up owned %s:%s: %v", resource, id, err)
				}
			}
		}
	})
	creatorIdentity, strangerIdentity, adminIdentity := uuid.NewString(), uuid.NewString(), uuid.NewString()
	creator := &models.User{ID: base, KratosID: &creatorIdentity, Username: platformID + "-creator", Email: platformID + "-creator@test.invalid", Password: "test-only"}
	stranger := &models.User{ID: base + 1, KratosID: &strangerIdentity, Username: platformID + "-stranger", Email: platformID + "-stranger@test.invalid", Password: "test-only"}
	admin := &models.User{ID: base + 2, KratosID: &adminIdentity, Username: platformID + "-admin", Email: platformID + "-admin@test.invalid", Password: "test-only"}
	for _, user := range []*models.User{creator, stranger, admin} {
		insert(user)
		if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,'none')", user.ID); err != nil {
			t.Fatal(err)
		}
	}
	kind := &models.OrganizationType{Title: platformID}
	insert(kind)
	org := &models.Organization{ID: base, Title: platformID, OrganizationTypeID: kind.ID}
	insert(org)
	role := &models.RoleTemplate{OrganizationTypeID: kind.ID, Key: platformID, DisplayName: "Creator", IsActive: true}
	insert(role)
	insert(&models.RoleTemplateCapability{RoleTemplateID: role.ID, Capability: models.Capability("create_event")})
	membership := &models.OrganizationMember{OrganizationID: org.ID, UserID: creator.ID, RoleTemplateID: role.ID, AssignedAt: time.Now()}
	insert(membership)
	if _, err := db.ExecContext(ctx, "SELECT setval(pg_get_serial_sequence('events', 'id'), ?, false)", base); err != nil {
		t.Fatal(err)
	}
	enforcer := authz.NewEnforcer(db, graph)
	if err := graph.LinkOrganizationToPlatform(ctx, fmt.Sprint(org.ID)); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.SyncOrganizationMember(ctx, org.ID, creator.ID); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.AssignPlatformRole(ctx, admin.ID, "administrator"); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, graph, enforcer, nil, nil, "", "")
	actor := func(identity string) context.Context { return context.WithValue(ctx, auth.UserIDKey, identity) }
	create := func(title string) int32 {
		t.Helper()
		status := eventsv1.EventStatus_EVENT_STATUS_DRAFT
		response, err := svc.CreateEvent(actor(creatorIdentity), connect.NewRequest(&eventsv1.CreateEventRequest{
			Title: title, Description: `"fixture"`, OrganizationId: org.ID, Status: &status,
			StartTime: time.Now().Add(time.Hour).Format(time.RFC3339), EndTime: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
		}))
		if err != nil {
			t.Fatalf("ordinary organization creator could not create event: %v", err)
		}
		if response.Msg.Event.UserId != creator.ID {
			t.Fatalf("event creator = %d, want backend users.id %d", response.Msg.Event.UserId, creator.ID)
		}
		return response.Msg.Event.Id
	}
	creatorEvent := create(platformID + "-creator-event")
	adminEvent := create(platformID + "-admin-event")
	if _, err := db.NewDelete().Model(membership).WherePK().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err := enforcer.SyncOrganizationMember(ctx, org.ID, creator.ID); err != nil {
		t.Fatal(err)
	}
	check := func(user, resource, id, permission string, want bool) {
		t.Helper()
		allowed, err := graph.CheckPermission(ctx, user, resource, id, permission)
		if err != nil || allowed != want {
			t.Fatalf("%s %s:%s %s = %v, want %v: %v", user, resource, id, permission, allowed, want, err)
		}
	}
	check(fmt.Sprint(creator.ID), "organization", fmt.Sprint(org.ID), "create_event", false)
	check(fmt.Sprint(creator.ID), "platform", platformID, "manage_system", false)
	for _, permission := range []string{"edit", "delete", "manage_registrations", "check_in"} {
		check(fmt.Sprint(creator.ID), "event", fmt.Sprint(creatorEvent), permission, true)
		check(creatorIdentity, "event", fmt.Sprint(creatorEvent), permission, false)
		check(fmt.Sprint(stranger.ID), "event", fmt.Sprint(creatorEvent), permission, false)
		check(fmt.Sprint(admin.ID), "event", fmt.Sprint(creatorEvent), permission, true)
	}
	update := func(identity string, id int32) error {
		_, err := svc.UpdateEvent(actor(identity), connect.NewRequest(&eventsv1.UpdateEventRequest{Id: id, Description: ptrString(`"updated"`)}))
		return err
	}
	remove := func(identity string, id int32) error {
		_, err := svc.DeleteEvent(actor(identity), connect.NewRequest(&eventsv1.DeleteEventRequest{Id: id}))
		return err
	}
	if err := update(strangerIdentity, creatorEvent); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("another ordinary user could modify event: %v", err)
	}
	if err := remove(strangerIdentity, creatorEvent); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("another ordinary user could delete event: %v", err)
	}
	if err := update(creatorIdentity, creatorEvent); err != nil {
		t.Fatalf("ordinary creator denied modification after org grant removed: %v", err)
	}
	if err := update(adminIdentity, adminEvent); err != nil {
		t.Fatalf("administrator denied modification: %v", err)
	}
	if err := remove(adminIdentity, adminEvent); err != nil {
		t.Fatalf("administrator denied deletion: %v", err)
	}
	if err := remove(creatorIdentity, creatorEvent); err != nil {
		t.Fatalf("ordinary creator denied deletion after org grant removed: %v", err)
	}
}
