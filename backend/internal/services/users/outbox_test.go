//go:build integration

package users

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	usersv1 "github.com/studyverse/ems-backend/gen/users/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestUserMutationsCommitOutboxAndRollbackEnqueueFailures(t *testing.T) {
	db := newTestDB(t)
	kratosID := "outbox-admin"
	seedAuthUser(t, db, kratosID)
	ctx := authedContext(kratosID)
	capture := &capturePublisher{}
	svc := NewService(db, nil, authz.NewEnforcer(db, testsupport.AllowedPermissions(t)), nil,
		events.NewDurablePublisher(db, capture), "", "")
	created, err := svc.CreateUser(ctx, connect.NewRequest(&usersv1.CreateUserRequest{
		Username: "outbox-user", Email: "outbox-user@example.test", Password: "synthetic",
	}))
	if err != nil {
		t.Fatal(err)
	}
	username := "updated-user"
	if _, err := svc.UpdateUser(ctx, connect.NewRequest(&usersv1.UpdateUserRequest{Id: created.Msg.User.Id, Username: &username})); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteUser(ctx, connect.NewRequest(&usersv1.DeleteUserRequest{Id: created.Msg.User.Id})); err != nil {
		t.Fatal(err)
	}
	var rows []events.OutboxMessage
	if err := db.NewSelect().Model(&rows).OrderExpr("id ASC").Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Topic != events.TopicUserCreated || rows[1].Topic != events.TopicUserUpdated || rows[2].Topic != events.TopicUserDeleted {
		t.Fatalf("unexpected durable events: %+v", rows)
	}
	if capture.msg != nil {
		t.Fatal("mutation must enqueue without contacting the broker")
	}
	target := models.User{Username: "preserved-user", Email: "preserved@example.test", Password: "synthetic"}
	if _, err := db.NewInsert().Model(&target).Returning("*").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE domain_event_outbox ADD CONSTRAINT reject_new_events CHECK (false) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateUser(ctx, connect.NewRequest(&usersv1.CreateUserRequest{
		Username: "must-rollback", Email: "rollback@example.test", Password: "synthetic",
	})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("create should reject failed enqueue, got %v", err)
	}
	username = "must-rollback"
	if _, err := svc.UpdateUser(ctx, connect.NewRequest(&usersv1.UpdateUserRequest{Id: target.ID, Username: &username})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("update should reject failed enqueue, got %v", err)
	}
	if _, err := svc.DeleteUser(ctx, connect.NewRequest(&usersv1.DeleteUserRequest{Id: target.ID})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("delete should reject failed enqueue, got %v", err)
	}
	if err := db.NewSelect().Model(&target).WherePK().Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if target.Username != "preserved-user" {
		t.Fatalf("update escaped rollback: %q", target.Username)
	}
	exists, err := db.NewSelect().Model((*models.User)(nil)).Where("email = ?", "rollback@example.test").Exists(ctx)
	if err != nil || exists {
		t.Fatalf("creation escaped rollback: exists=%v err=%v", exists, err)
	}
}
