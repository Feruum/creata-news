package main

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/studyverse/ems-backend/internal/config"
	"github.com/studyverse/ems-backend/internal/events"
)

func newTestInjector(t *testing.T) do.Injector {
	t.Helper()
	injector := do.New()

	do.ProvideValue(injector, &config.Config{
		RedisURL: "redis://:secret@localhost:6379/0",
	})

	do.Provide(injector, func(i do.Injector) (*events.Bus, error) {
		cfg := do.MustInvoke[*config.Config](i)
		logger := watermill.NewSlogLoggerWithLevelMapping(slog.Default(), nil)

		newRedis := func() (*redis.Client, error) {
			opts, err := redis.ParseURL(cfg.RedisURL)
			if err != nil {
				return nil, fmt.Errorf("parse redis URL: %w", err)
			}
			return redis.NewClient(opts), nil
		}

		pubClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		searchClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		telegramClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		activityClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		historyClient, err := newRedis()
		if err != nil {
			return nil, err
		}
		stickClient, err := newRedis()
		if err != nil {
			return nil, err
		}

		return events.NewBus(logger, pubClient, searchClient, telegramClient, activityClient, historyClient, stickClient)
	})

	do.Provide(injector, func(i do.Injector) (*events.Publisher, error) {
		bus := do.MustInvoke[*events.Bus](i)
		return events.NewPublisher(bus.Publisher()), nil
	})

	return injector
}

func TestInjectorProvidesEventBusAndPublisher(t *testing.T) {
	injector := newTestInjector(t)
	if _, err := do.Invoke[*events.Bus](injector); err != nil {
		t.Fatalf("invoke bus: %v", err)
	}
	if _, err := do.Invoke[*events.Publisher](injector); err != nil {
		t.Fatalf("invoke publisher: %v", err)
	}
}

// TestBusErrorPropagation verifies that a non-context.Canceled error from
// bus.Run is forwarded through busErrCh so the main goroutine can trigger
// shutdown. This pattern is wired in main() via startBus().
func TestBusErrorPropagation(t *testing.T) {
	logger := watermill.NewSlogLoggerWithLevelMapping(slog.Default(), nil)
	bus, err := events.NewGoChannelBus(logger)
	if err != nil {
		t.Fatal(err)
	}

	busErrCh := make(chan error, 1)
	startBus(bus, busErrCh)

	// Give the router a moment to start, then close it to simulate a fatal exit.
	time.Sleep(50 * time.Millisecond)
	if err := bus.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-busErrCh:
		// context.Canceled is expected after Close(); that should NOT propagate.
		if err != nil {
			t.Fatalf("unexpected non-nil error on busErrCh: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bus exit on busErrCh")
	}
}

// TestBusErrorPropagation_FatalError verifies that a genuine (non-context.Canceled)
// error from bus.Run IS forwarded through busErrCh.
func TestBusErrorPropagation_FatalError(t *testing.T) {
	busErrCh := make(chan error, 1)
	wantErr := errors.New("redis connection lost")
	startBusRunner(busErrCh, func() error {
		return wantErr
	})

	select {
	case gotErr := <-busErrCh:
		if !errors.Is(gotErr, wantErr) {
			t.Fatalf("got %v, want %v", gotErr, wantErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: fatal bus error was not propagated")
	}
}
