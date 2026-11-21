package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/events/handlers"
)

func TestGoChannelBusRunsRegisteredHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus, err := events.NewGoChannelBus(watermill.NopLogger{})
	if err != nil {
		t.Fatalf("NewGoChannelBus() error = %v", err)
	}
	defer bus.Close()

	called := make(chan struct{}, 1)
	deps := handlers.HandlerDeps{
		OnUserCreated: func() { called <- struct{}{} },
	}

	handlers.RegisterHandlers(bus.Router(), deps, bus.SearchSub(), bus.TelegramSub(), bus.ActivitySub(), bus.HistorySub())

	go func() { _ = bus.Run(ctx) }()
	<-bus.Router().Running()

	publisher := events.NewPublisher(bus.Publisher())
	if err := publisher.PublishUserCreated(ctx, events.UserCreatedPayload{UserID: 1, Username: "alice", Email: "alice@example.com"}); err != nil {
		t.Fatalf("PublishUserCreated() error = %v", err)
	}

	select {
	case <-called:
	case <-time.After(5 * time.Second):
		t.Fatal("expected registered search handler to run")
	}
}
