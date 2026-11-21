package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const memberSyncTopic = "internal.member.sync"
const platformRoleSyncTopic = "internal.platform.role.sync"

// Err exposes a failed enqueue even if a legacy service logs its publish error.
func (p *Publisher) Err() error {
	if p == nil {
		return nil
	}
	return p.failure
}

type OutboxMessage struct {
	bun.BaseModel `bun:"table:domain_event_outbox,alias:outbox"`
	ID            int64             `bun:"id,pk,autoincrement"`
	UUID          string            `bun:"uuid,notnull,unique"`
	Topic         string            `bun:"topic,notnull"`
	Payload       json.RawMessage   `bun:"payload,type:jsonb,notnull"`
	Metadata      map[string]string `bun:"metadata,type:jsonb,notnull"`
	Attempts      int               `bun:"attempts,notnull,default:0"`
	AvailableAt   time.Time         `bun:"available_at,notnull,default:current_timestamp"`
	CreatedAt     time.Time         `bun:"created_at,notnull,default:current_timestamp"`
}

// NewDurablePublisher stores messages in PostgreSQL. Mutations must use
// InTransaction with their own transaction so data and message commit together.
func NewDurablePublisher(db *bun.DB, transport message.Publisher) *Publisher {
	return &Publisher{pub: transport, store: db}
}

func (p *Publisher) InTransaction(tx bun.IDB) *Publisher {
	if p == nil {
		return nil
	}
	if p.store == nil {
		return p
	}
	return &Publisher{pub: p.pub, store: tx}
}

func enqueue(ctx context.Context, db bun.IDB, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}
	metadata := map[string]string{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(metadata))
	record := &OutboxMessage{UUID: watermill.NewUUID(), Topic: topic, Payload: data, Metadata: metadata}
	if _, err := db.NewInsert().Model(record).Exec(ctx); err != nil {
		return fmt.Errorf("enqueue %s: %w", topic, err)
	}
	return nil
}

func EnqueueMemberSync(ctx context.Context, tx bun.IDB, orgID, userID int32) error {
	return enqueue(ctx, tx, memberSyncTopic, memberSync{OrganizationID: orgID, UserID: userID})
}

func EnqueuePlatformRoleSync(ctx context.Context, tx bun.IDB, userID int32) error {
	return enqueue(ctx, tx, platformRoleSyncTopic, platformRoleSync{UserID: userID})
}

type platformRoleSync struct {
	UserID int32 `json:"user_id"`
}

type memberSync struct {
	OrganizationID int32 `json:"organization_id"`
	UserID         int32 `json:"user_id"`
}

type OutboxDispatcher struct {
	db               *bun.DB
	transport        message.Publisher
	syncMember       func(context.Context, int32, int32) error
	syncPlatformRole func(context.Context, int32) error
}

func NewOutboxDispatcher(db *bun.DB, transport message.Publisher, syncMember func(context.Context, int32, int32) error, syncPlatformRole func(context.Context, int32) error) *OutboxDispatcher {
	return &OutboxDispatcher{db: db, transport: transport, syncMember: syncMember, syncPlatformRole: syncPlatformRole}
}

// DispatchOne retains failures with backoff and a stable UUID. Transport delivery
// is at least once: a crash between publish and commit may replay the same UUID.
func (w *OutboxDispatcher) DispatchOne(ctx context.Context) error {
	_, err := w.dispatchOne(ctx)
	return err
}

func (w *OutboxDispatcher) dispatchOne(ctx context.Context) (bool, error) {
	delivered := false
	var deliveryErr error
	err := w.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var row OutboxMessage
		err := tx.NewSelect().Model(&row).Where("available_at <= NOW()").OrderExpr("id ASC").Limit(1).For("UPDATE SKIP LOCKED").Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select outbox: %w", err)
		}
		delivered = true
		if row.Topic == platformRoleSyncTopic {
			var job platformRoleSync
			if err := json.Unmarshal(row.Payload, &job); err != nil {
				deliveryErr = err
			} else if w.syncPlatformRole == nil {
				deliveryErr = errors.New("platform role reconciler unavailable")
			} else {
				deliveryErr = w.syncPlatformRole(ctx, job.UserID)
			}
		} else if row.Topic == memberSyncTopic {
			var job memberSync
			if err := json.Unmarshal(row.Payload, &job); err != nil {
				deliveryErr = err
			} else if w.syncMember == nil {
				deliveryErr = errors.New("membership reconciler unavailable")
			} else {
				deliveryErr = w.syncMember(ctx, job.OrganizationID, job.UserID)
			}
		} else {
			msg := message.NewMessageWithContext(ctx, row.UUID, message.Payload(row.Payload))
			msg.Metadata = message.Metadata(row.Metadata)
			deliveryErr = w.transport.Publish(row.Topic, msg)
		}
		if deliveryErr != nil {
			delay := time.Second << min(row.Attempts, 10)
			_, err := tx.NewUpdate().Model(&row).Set("attempts = attempts + 1").Set("available_at = ?", time.Now().Add(delay)).WherePK().Exec(ctx)
			return err
		}
		_, err = tx.NewDelete().Model(&row).WherePK().Exec(ctx)
		return err
	})
	if err != nil {
		return delivered, fmt.Errorf("dispatch outbox: %w", err)
	}
	return delivered, deliveryErr
}

func (w *OutboxDispatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for batch := 0; batch < 100 && ctx.Err() == nil; batch++ {
				attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				delivered, err := w.dispatchOne(attemptCtx)
				cancel()
				if err != nil && ctx.Err() == nil {
					slog.ErrorContext(ctx, "outbox delivery failed; retained for retry", "error", err)
				}
				if !delivered {
					break
				}
			}
		}
	}
}
