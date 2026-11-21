//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/uptrace/bun"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/models"
)

func TestReplayDoesNotDuplicateActivityOrMembershipHistory(t *testing.T) {
	db := newTestDB(t)
	actor := int32(2)
	payload, _ := json.Marshal(events.MemberAddedPayload{UserID: 1, OrganizationID: 10, OrganizationTitle: "Chess", RoleTemplateID: 5, RoleName: "member", ActorID: &actor})
	msg := message.NewMessage("stable-outbox-uuid", payload)
	msg.SetContext(context.Background())
	for _, handler := range []message.NoPublishHandlerFunc{handleMemberAddedActivity(HandlerDeps{DB: db}), handleMemberAddedHistory(HandlerDeps{DB: db})} {
		if err := handler(msg); err != nil {
			t.Fatal(err)
		}
		if err := handler(msg); err != nil {
			t.Fatal(err)
		}
	}
	for _, model := range []any{(*models.ActivityLog)(nil), (*models.MembershipHistory)(nil)} {
		count, err := db.NewSelect().Model(model).Count(context.Background())
		if err != nil || count != 1 {
			t.Fatalf("replay duplicated %T: %d %v", model, count, err)
		}
	}
}

func TestConsumerFailureRollsBackReceiptAndRemainsRetryable(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "CREATE TABLE consumer_write (id int PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("write failed")
	err := consumeOnce(ctx, db, "test-consumer", "retryable-uuid", func(tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO consumer_write VALUES (1)"); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("expected write failure: %v", err)
	}
	for _, table := range []string{"consumer_write", "domain_event_inbox"} {
		var count int
		if err := db.NewRaw("SELECT count(*) FROM "+table).Scan(ctx, &count); err != nil || count != 0 {
			t.Fatalf("partial %s committed: %d %v", table, count, err)
		}
	}
	write := func(tx bun.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO consumer_write VALUES (1)")
		return err
	}
	for i := 0; i < 2; i++ {
		if err := consumeOnce(ctx, db, "test-consumer", "retryable-uuid", write); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM consumer_write").Scan(ctx, &count); err != nil || count != 1 {
		t.Fatalf("retry result: %d %v", count, err)
	}
}
