//go:build integration

package events

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	authzed "github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type badgeWorkflowFixture struct {
	db      *bun.DB
	service *Service
	editor  context.Context
	admin   context.Context
	staff   context.Context
	adminID int32
	orgID   int32
}

func newBadgeWorkflowFixture(t *testing.T) badgeWorkflowFixture {
	t.Helper()
	if os.Getenv("RUN_SPICEDB_INTEGRATION_TESTS") != "1" {
		t.Skip("badge authorization requires an explicitly disposable SpiceDB server")
	}
	endpoint, key := os.Getenv("SPICEDB_TEST_ENDPOINT"), os.Getenv("SPICEDB_TEST_PRESHARED_KEY")
	if endpoint == "" || key == "" || os.Getenv("DATABASE_URL") == "" {
		t.Fatal("badge workflow requires disposable PostgreSQL and SpiceDB configuration")
	}
	ctx := context.Background()
	raw, err := authzed.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpcutil.WithInsecureBearerToken(key))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("../../../../../infra/spicedb/schema.zed")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		_, err = raw.WriteSchema(ctx, &pb.WriteSchemaRequest{Schema: string(schema)})
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("load isolated authorization schema: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	name := fmt.Sprintf("badges-%d", time.Now().UnixNano())
	client, err := perms.NewClient(endpoint, key, true, false, name)
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
	kind := &models.OrganizationType{Title: name}
	insert(kind)
	base := int32(1_000_000_000) + rand.Int32N(900_000_000)
	active := models.OrganizationStatusActive
	org := &models.Organization{ID: base, Title: name, OrganizationTypeID: kind.ID, Status: &active}
	insert(org)
	users := make([]*models.User, 3)
	for i, role := range []string{"none", "administrator", "staff"} {
		identity := fmt.Sprintf("%s-%d", name, i)
		users[i] = &models.User{ID: base + int32(i), KratosID: &identity, Username: identity, Email: identity + "@example.invalid", Password: "test-only"}
		insert(users[i])
		if _, err := db.ExecContext(ctx, "INSERT INTO platform_user_roles(user_id,role) VALUES(?,?)", users[i].ID, role); err != nil {
			t.Fatal(err)
		}
	}
	role := &models.RoleTemplate{OrganizationTypeID: kind.ID, Key: name, DisplayName: "Badge editor", IsActive: true}
	insert(role)
	insert(&models.RoleTemplateCapability{RoleTemplateID: role.ID, Capability: models.Capability("create_event")})
	insert(&models.OrganizationMember{OrganizationID: org.ID, UserID: users[0].ID, RoleTemplateID: role.ID, AssignedAt: time.Now()})
	if err := client.LinkOrganizationToPlatform(ctx, fmt.Sprint(org.ID)); err != nil {
		t.Fatal(err)
	}
	if err := client.SetOrganizationMember(ctx, fmt.Sprint(org.ID), fmt.Sprint(users[0].ID), []string{perms.CapabilityCreateEvent}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetPlatformRole(ctx, fmt.Sprint(users[1].ID), "administrator"); err != nil {
		t.Fatal(err)
	}
	if err := client.SetPlatformRole(ctx, fmt.Sprint(users[2].ID), "staff"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for resource, id := range map[string]string{"platform": name, "organization": fmt.Sprint(org.ID)} {
			if _, err := raw.DeleteRelationships(cleanupCtx, &pb.DeleteRelationshipsRequest{RelationshipFilter: &pb.RelationshipFilter{ResourceType: resource, OptionalResourceId: id}}); err != nil {
				t.Error(err)
			}
		}
	})
	return badgeWorkflowFixture{
		db: db, service: NewService(db, client, authz.NewEnforcer(db, client), nil, nil, "", ""),
		editor: authedContext(*users[0].KratosID), admin: authedContext(*users[1].KratosID), staff: authedContext(*users[2].KratosID),
		adminID: users[1].ID, orgID: org.ID,
	}
}

func (f badgeWorkflowFixture) create(t *testing.T, socGpa, iros bool) *eventsv1.Event {
	t.Helper()
	address := "Test venue"
	response, err := f.service.CreateEvent(f.editor, connect.NewRequest(&eventsv1.CreateEventRequest{
		Title: "Badge approval workflow", Description: "Synthetic badge workflow event", OrganizationId: f.orgID,
		StartTime: time.Now().Add(time.Hour).Format(time.RFC3339), EndTime: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_EXTERNAL, ExternalAddress: &address},
		SocGpaRequested:    &socGpa, IrosRequested: &iros,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return response.Msg.Event
}

func TestLiveEventBadgeRequestApprovalWorkflow(t *testing.T) {
	f := newBadgeWorkflowFixture(t)
	event := f.create(t, true, false)
	pending, approved := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_PENDING, eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED
	if event.SocGpaBadge.Status != pending || event.IrosBadge.Status != eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_NOT_REQUESTED {
		t.Fatal("creation granted an unreviewed badge")
	}
	for _, ctx := range []context.Context{f.editor, f.staff} {
		_, err := f.service.ReviewEventBadges(ctx, connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: event.Id, SocGpaStatus: &approved}))
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("non-administrator reviewed a badge: %v", err)
		}
	}
	_, err := f.service.ReviewEventBadges(f.admin, connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: event.Id, SocGpaStatus: &approved, IrosStatus: &approved}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unrequested badge review accepted: %v", err)
	}
	stored := &models.Event{ID: event.Id}
	if err := f.db.NewSelect().Model(stored).WherePK().Scan(f.admin); err != nil {
		t.Fatal(err)
	}
	if stored.SocGpaBadgeStatus != badgePending || stored.SocGpaBadgeReviewedBy != nil {
		t.Fatal("failed combined review partially approved a badge")
	}
	response, err := f.service.ReviewEventBadges(f.admin, connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: event.Id, SocGpaStatus: &approved}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Event.SocGpaBadge.GetReviewedBy() != f.adminID || response.Msg.Event.SocGpaBadge.ReviewedAt == nil {
		t.Fatal("reviewer metadata missing")
	}
	for _, ctx := range []context.Context{context.Background(), f.editor, f.admin} {
		detail, err := f.service.GetEvent(ctx, connect.NewRequest(&eventsv1.GetEventRequest{Id: event.Id}))
		if err != nil {
			t.Fatal(err)
		}
		if detail.Msg.Event.SocGpaBadge.Status != approved || detail.Msg.Event.IrosBadge.Status != eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_NOT_REQUESTED {
			t.Fatal("detail lost independent badge decisions")
		}
		if detail.Msg.CanReviewBadges != (ctx == f.admin) {
			t.Fatal("incorrect review capability")
		}
	}
	public, err := f.service.ListEvents(context.Background(), connect.NewRequest(&eventsv1.ListEventsRequest{}))
	if err != nil || len(public.Msg.Events) != 1 || public.Msg.Events[0].SocGpaBadge.Status != approved {
		t.Fatalf("public list lost approval: %v", err)
	}
	requested, removed := true, false
	updated, err := f.service.UpdateEvent(f.editor, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.Id, IrosRequested: &requested}))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Msg.Event.SocGpaBadge.Status != approved || updated.Msg.Event.IrosBadge.Status != pending {
		t.Fatal("second request changed first approval")
	}
	rejected := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_REJECTED
	if _, err := f.service.ReviewEventBadges(f.admin, connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: event.Id, IrosStatus: &rejected})); err != nil {
		t.Fatal(err)
	}
	oldClient, err := f.service.UpdateEvent(f.editor, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.Id}))
	if err != nil {
		t.Fatal(err)
	}
	if oldClient.Msg.Event.SocGpaBadge.Status != approved || oldClient.Msg.Event.IrosBadge.Status != rejected {
		t.Fatal("omitted request flags reset existing decisions")
	}
	removedResponse, err := f.service.UpdateEvent(f.editor, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.Id, SocGpaRequested: &removed}))
	if err != nil {
		t.Fatal(err)
	}
	if removedResponse.Msg.Event.SocGpaBadge.Status != eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_NOT_REQUESTED || removedResponse.Msg.Event.SocGpaBadge.ReviewedBy != nil {
		t.Fatal("withdrawn badge retained approval")
	}
	repeated, err := f.service.UpdateEvent(f.editor, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.Id, SocGpaRequested: &requested, IrosRequested: &requested}))
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Msg.Event.SocGpaBadge.Status != pending || repeated.Msg.Event.IrosBadge.Status != pending {
		t.Fatal("re-request inherited a prior decision")
	}
}

func TestLiveEventBadgeReviewRollsBackOutboxFailure(t *testing.T) {
	f := newBadgeWorkflowFixture(t)
	event := f.create(t, true, true)
	if _, err := f.db.ExecContext(f.admin, "ALTER TABLE domain_event_outbox ADD CONSTRAINT badge_test_outbox_failure CHECK(false)"); err != nil {
		t.Fatal(err)
	}
	f.service.publisher = domainevents.NewDurablePublisher(f.db, nil)
	approved := eventsv1.EventBadgeStatus_EVENT_BADGE_STATUS_APPROVED
	_, err := f.service.ReviewEventBadges(f.admin, connect.NewRequest(&eventsv1.ReviewEventBadgesRequest{EventId: event.Id, SocGpaStatus: &approved, IrosStatus: &approved}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("failed outbox reported review success: %v", err)
	}
	stored := &models.Event{ID: event.Id}
	if err := f.db.NewSelect().Model(stored).WherePK().Scan(f.admin); err != nil {
		t.Fatal(err)
	}
	if stored.SocGpaBadgeStatus != badgePending || stored.IrosBadgeStatus != badgePending || stored.SocGpaBadgeReviewedBy != nil || stored.IrosBadgeReviewedBy != nil {
		t.Fatal("review escaped transaction rollback")
	}
}
