package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

type inboxReceipt struct {
	bun.BaseModel `bun:"table:domain_event_inbox"`
	Consumer      string    `bun:"consumer,pk"`
	MessageID     string    `bun:"message_id,pk"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// The receipt and consumer write commit together. The separate receipt table
// also works with activity_log's legacy PostgreSQL INSERT partition rules.
func consumeOnce(ctx context.Context, db *bun.DB, consumer, messageID string, write func(bun.Tx) error) error {
	if messageID == "" {
		return fmt.Errorf("consumer %s requires a message UUID", consumer)
	}
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		receipt := &inboxReceipt{Consumer: consumer, MessageID: messageID}
		result, err := tx.NewInsert().Model(receipt).On("CONFLICT (consumer,message_id) DO NOTHING").Exec(ctx)
		if err != nil {
			return fmt.Errorf("record consumer receipt: %w", err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		return write(tx)
	})
}

func insertMembershipHistory(deps HandlerDeps, msg *message.Message, record *models.MembershipHistory) error {
	record.SourceEventID = &msg.UUID
	return consumeOnce(msg.Context(), deps.DB, "membership-history", msg.UUID, func(tx bun.Tx) error {
		_, err := tx.NewInsert().Model(record).Exec(msg.Context())
		return err
	})
}
