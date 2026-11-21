package events

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	redisstream "github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	wmiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/redis/go-redis/v9"
)

// Bus encapsulates the Watermill router, publisher, four subscriber groups,
// and an optional Redis client for STICK commands.
type Bus struct {
	router      *message.Router
	publisher   message.Publisher
	searchSub   message.Subscriber
	telegramSub message.Subscriber
	activitySub message.Subscriber
	historySub  message.Subscriber
	stickClient *redis.Client
}

// NewGoChannelBus creates a Bus backed by an in-memory GoChannel for testing.
// Persistent mode ensures messages published before subscription are retained.
func NewGoChannelBus(logger watermill.LoggerAdapter) (*Bus, error) {
	pubSub := gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}
	return &Bus{
		router:      router,
		publisher:   pubSub,
		searchSub:   pubSub,
		telegramSub: pubSub,
		activitySub: pubSub,
		historySub:  pubSub,
	}, nil
}

// NewBus creates a production Bus backed by Redis Streams with separate clients
// for the publisher, each subscriber group, and STICK operations.
func NewBus(
	logger watermill.LoggerAdapter,
	pubClient, searchClient, telegramClient, activityClient, historyClient, stickClient *redis.Client,
) (*Bus, error) {
	pub, err := redisstream.NewPublisher(redisstream.PublisherConfig{Client: pubClient}, logger)
	if err != nil {
		return nil, fmt.Errorf("new redis publisher: %w", err)
	}

	newSub := func(client *redis.Client, group string) (message.Subscriber, error) {
		return redisstream.NewSubscriber(redisstream.SubscriberConfig{
			Client:                        client,
			ConsumerGroup:                 group,
			Consumer:                      watermill.NewUUID(),
			OldestId:                      "0",
			DisableIndefiniteInitialBlock: true,
		}, logger)
	}

	searchSub, err := newSub(searchClient, "search-indexer")
	if err != nil {
		return nil, fmt.Errorf("new search subscriber: %w", err)
	}
	telegramSub, err := newSub(telegramClient, "telegram-notifier")
	if err != nil {
		return nil, fmt.Errorf("new telegram subscriber: %w", err)
	}
	activitySub, err := newSub(activityClient, "activity-logger")
	if err != nil {
		return nil, fmt.Errorf("new activity subscriber: %w", err)
	}
	historySub, err := newSub(historyClient, "membership-historian")
	if err != nil {
		return nil, fmt.Errorf("new history subscriber: %w", err)
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}

	poisonMiddleware, err := wmiddleware.PoisonQueue(pub, TopicPoison)
	if err != nil {
		return nil, fmt.Errorf("poison queue middleware: %w", err)
	}

	router.AddMiddleware(
		wmiddleware.CorrelationID,
		wmiddleware.Recoverer,
		wmiddleware.Retry{MaxRetries: 3}.Middleware,
		poisonMiddleware,
	)

	return &Bus{
		router:      router,
		publisher:   pub,
		searchSub:   searchSub,
		telegramSub: telegramSub,
		activitySub: activitySub,
		historySub:  historySub,
		stickClient: stickClient,
	}, nil
}

// InitStick runs the Dragonfly STICK command on all topic streams and the
// poison queue to prevent eviction. No-op when stickClient is nil (test mode).
func (b *Bus) InitStick(ctx context.Context) error {
	if b.stickClient == nil {
		return nil
	}
	for _, topic := range append(AllTopics, TopicPoison) {
		if err := b.stickClient.Do(ctx, "STICK", topic).Err(); err != nil {
			return fmt.Errorf("STICK %s: %w", topic, err)
		}
	}
	return nil
}

// Run starts the Watermill router. It blocks until the context is cancelled
// or the router encounters a fatal error.
func (b *Bus) Run(ctx context.Context) error { return b.router.Run(ctx) }

// Ready verifies that the durable event transport is reachable.
func (b *Bus) Ready(ctx context.Context) error {
	if b.stickClient == nil {
		return nil
	}
	return b.stickClient.Ping(ctx).Err()
}

// Close gracefully shuts down the router.
func (b *Bus) Close() error { return b.router.Close() }

// Router returns the underlying Watermill router for handler registration.
func (b *Bus) Router() *message.Router { return b.router }

// Publisher returns the message publisher for domain event emission.
func (b *Bus) Publisher() message.Publisher { return b.publisher }

// SearchSub returns the subscriber for the search-indexer consumer group.
func (b *Bus) SearchSub() message.Subscriber { return b.searchSub }

// TelegramSub returns the subscriber for the telegram-notifier consumer group.
func (b *Bus) TelegramSub() message.Subscriber { return b.telegramSub }

// ActivitySub returns the subscriber for the activity-logger consumer group.
func (b *Bus) ActivitySub() message.Subscriber { return b.activitySub }

// HistorySub returns the subscriber for the membership-historian consumer group.
func (b *Bus) HistorySub() message.Subscriber { return b.historySub }
