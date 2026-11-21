//go:build integration

package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/testsupport"
	"github.com/uptrace/bun"
)

type outboxTransport struct {
	fail bool
	ids  []string
}

func (p *outboxTransport) Publish(_ string, msgs ...*message.Message) error {
	for _, msg := range msgs {
		p.ids = append(p.ids, msg.UUID)
	}
	if p.fail {
		return errors.New("broker unavailable")
	}
	return nil
}

type batchTransport struct{ delivered chan struct{} }

func (p *batchTransport) Publish(_ string, msgs ...*message.Message) error {
	for range msgs {
		p.delivered <- struct{}{}
	}
	return nil
}
func (*batchTransport) Close() error { return nil }

func TestDispatcherDrainsReadyBacklogWithoutPerMessageTickerDelay(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := db.NewCreateTable().Model((*OutboxMessage)(nil)).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	transport := &batchTransport{delivered: make(chan struct{}, 20)}
	publisher := NewDurablePublisher(db, transport)
	for i := 0; i < 20; i++ {
		if err := publisher.PublishUserCreated(ctx, UserCreatedPayload{UserID: int32(i)}); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{})
	go func() { NewOutboxDispatcher(db, transport, nil, nil).Start(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for i := 0; i < 20; i++ {
		select {
		case <-transport.delivered:
		case <-deadline.C:
			t.Fatalf("only %d/20 ready events delivered before deadline", i)
		}
	}
}
func (*outboxTransport) Close() error { return nil }

func TestOutboxRollsBackWithMutationAndRetriesStableMessage(t *testing.T) {
	db := testsupport.IsolatedDB(t)
	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*OutboxMessage)(nil)).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE mutations (id int PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	transport := &outboxTransport{fail: true}
	publisher := NewDurablePublisher(db, transport)
	rollback := errors.New("abort mutation")
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO mutations VALUES (1)"); err != nil {
			return err
		}
		if err := publisher.InTransaction(tx).PublishUserCreated(ctx, UserCreatedPayload{UserID: 1}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error: %v", err)
	}
	count, err := db.NewSelect().Model((*OutboxMessage)(nil)).Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("rolled-back outbox count=%d err=%v", count, err)
	}
	var mutations int
	if err := db.NewRaw("SELECT count(*) FROM mutations").Scan(ctx, &mutations); err != nil || mutations != 0 {
		t.Fatalf("mutation did not roll back: %d %v", mutations, err)
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO mutations VALUES (2)"); err != nil {
			return err
		}
		return publisher.InTransaction(tx).PublishUserCreated(ctx, UserCreatedPayload{UserID: 2})
	})
	if err != nil {
		t.Fatal(err)
	}
	worker := NewOutboxDispatcher(db, transport, nil, nil)
	if err := worker.DispatchOne(ctx); err == nil {
		t.Fatal("expected broker failure")
	}
	count, err = db.NewSelect().Model((*OutboxMessage)(nil)).Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("failed delivery was lost: count=%d err=%v", count, err)
	}
	transport.fail = false
	if _, err := db.NewUpdate().Model((*OutboxMessage)(nil)).Set("available_at = NOW()").Where("1=1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err := worker.DispatchOne(ctx); err != nil {
		t.Fatal(err)
	}
	if len(transport.ids) != 2 || transport.ids[0] != transport.ids[1] {
		t.Fatalf("retry UUID changed: %v", transport.ids)
	}
	count, err = db.NewSelect().Model((*OutboxMessage)(nil)).Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("successful delivery remains: count=%d err=%v", count, err)
	}
}
