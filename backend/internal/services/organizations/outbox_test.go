//go:build integration

package organizations

import (
	"testing"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestOrganizationMutationsCommitOutboxAndRollbackSlugsOnEnqueueFailure(t *testing.T) {
	db := newTestDB(t)
	kratosID := "outbox-staff"
	seedAuthUser(t, db, kratosID)
	ctx := authedContext(kratosID)
	if _, err := db.ExecContext(ctx, "INSERT INTO organization_types(id,title) VALUES (1,'Test') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	capture := &capturePublisher{}
	enforcer := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	svc := NewService(db, nil, enforcer, nil, events.NewDurablePublisher(db, capture), nil)
	created, err := svc.CreateOrganization(ctx, connect.NewRequest(&eventsv1.CreateOrganizationRequest{
		Title: "Outbox Club", OrganizationTypeId: 1,
	}))
	if err != nil {
		t.Fatal(err)
	}
	title := "Updated Club"
	if _, err := svc.UpdateOrganization(ctx, connect.NewRequest(&eventsv1.UpdateOrganizationRequest{Id: created.Msg.Organization.Id, Title: &title})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteOrganization(ctx, connect.NewRequest(&eventsv1.DeleteOrganizationRequest{Id: created.Msg.Organization.Id})); err != nil {
		t.Fatal(err)
	}
	var rows []events.OutboxMessage
	if err := db.NewSelect().Model(&rows).OrderExpr("id ASC").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Topic != events.TopicOrganizationCreated || rows[1].Topic != events.TopicOrganizationUpdated || rows[2].Topic != events.TopicOrganizationDeleted {
		t.Fatalf("unexpected durable events: %+v", rows)
	}
	if capture.msg != nil {
		t.Fatal("mutation must enqueue without contacting the broker")
	}
	plain := NewService(db, nil, enforcer, nil, nil, nil)
	target, err := plain.CreateOrganization(ctx, connect.NewRequest(&eventsv1.CreateOrganizationRequest{Title: "Preserved Club", OrganizationTypeId: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE domain_event_outbox ADD CONSTRAINT reject_new_events CHECK (false) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateOrganization(ctx, connect.NewRequest(&eventsv1.CreateOrganizationRequest{Title: "Must Rollback", OrganizationTypeId: 1})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("create should reject failed enqueue, got %v", err)
	}
	title = "Must Rollback"
	if _, err := svc.UpdateOrganization(ctx, connect.NewRequest(&eventsv1.UpdateOrganizationRequest{Id: target.Msg.Organization.Id, Title: &title})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("update should reject failed enqueue, got %v", err)
	}
	if _, err := svc.DeleteOrganization(ctx, connect.NewRequest(&eventsv1.DeleteOrganizationRequest{Id: target.Msg.Organization.Id})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("delete should reject failed enqueue, got %v", err)
	}
	var org models.Organization
	if err := db.NewSelect().Model(&org).Where("id = ?", target.Msg.Organization.Id).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if org.Title != "Preserved Club" {
		t.Fatalf("update escaped rollback: %q", org.Title)
	}
	var current models.OrganizationSlug
	if err := db.NewSelect().Model(&current).Where("organization_id = ? AND is_current = true", org.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if current.Slug != target.Msg.Organization.Slug {
		t.Fatalf("slug escaped rollback: %q", current.Slug)
	}
	exists, err := db.NewSelect().Model((*models.Organization)(nil)).Where("title = ?", "Must Rollback").Exists(ctx)
	if err != nil || exists {
		t.Fatalf("creation escaped rollback: exists=%v err=%v", exists, err)
	}
}
