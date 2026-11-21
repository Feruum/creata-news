package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Publisher wraps a Watermill message.Publisher with typed domain event methods.
type Publisher struct {
	pub     message.Publisher
	store   bun.IDB
	failure error
}

// NewPublisher creates a typed Publisher around any Watermill message.Publisher.
func NewPublisher(pub message.Publisher) *Publisher {
	return &Publisher{pub: pub}
}

func (p *Publisher) publish(ctx context.Context, topic string, payload any) error {
	if p.store != nil {
		err := enqueue(ctx, p.store, topic, payload)
		if err != nil {
			p.failure = err
		}
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	msg := message.NewMessageWithContext(ctx, watermill.NewUUID(), data)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(msg.Metadata))

	if err := p.pub.Publish(topic, msg); err != nil {
		return fmt.Errorf("publish %s: %w", topic, err)
	}

	return nil
}

func (p *Publisher) PublishUserCreated(ctx context.Context, payload UserCreatedPayload) error {
	return p.publish(ctx, TopicUserCreated, payload)
}

func (p *Publisher) PublishEventArchived(ctx context.Context, payload EventArchivedPayload) error {
	return p.publish(ctx, TopicEventArchived, payload)
}

func (p *Publisher) PublishUserUpdated(ctx context.Context, payload UserUpdatedPayload) error {
	return p.publish(ctx, TopicUserUpdated, payload)
}

func (p *Publisher) PublishUserDeleted(ctx context.Context, payload UserDeletedPayload) error {
	return p.publish(ctx, TopicUserDeleted, payload)
}

func (p *Publisher) PublishOrganizationCreated(ctx context.Context, payload OrganizationCreatedPayload) error {
	return p.publish(ctx, TopicOrganizationCreated, payload)
}

func (p *Publisher) PublishOrganizationUpdated(ctx context.Context, payload OrganizationUpdatedPayload) error {
	return p.publish(ctx, TopicOrganizationUpdated, payload)
}

func (p *Publisher) PublishOrganizationDeleted(ctx context.Context, payload OrganizationDeletedPayload) error {
	return p.publish(ctx, TopicOrganizationDeleted, payload)
}

func (p *Publisher) PublishEventCreated(ctx context.Context, payload EventCreatedPayload) error {
	return p.publish(ctx, TopicEventCreated, payload)
}

func (p *Publisher) PublishEventUpdated(ctx context.Context, payload EventUpdatedPayload) error {
	return p.publish(ctx, TopicEventUpdated, payload)
}

func (p *Publisher) PublishEventDeleted(ctx context.Context, payload EventDeletedPayload) error {
	return p.publish(ctx, TopicEventDeleted, payload)
}

func (p *Publisher) PublishEventPublished(ctx context.Context, payload EventPublishedPayload) error {
	return p.publish(ctx, TopicEventPublished, payload)
}

func (p *Publisher) PublishCollaborationInvited(ctx context.Context, payload CollaborationInvitedPayload) error {
	return p.publish(ctx, TopicCollaborationInvited, payload)
}

func (p *Publisher) PublishCollaborationAccepted(ctx context.Context, payload CollaborationAcceptedPayload) error {
	return p.publish(ctx, TopicCollaborationAccepted, payload)
}

func (p *Publisher) PublishCollaborationDeclined(ctx context.Context, payload CollaborationDeclinedPayload) error {
	return p.publish(ctx, TopicCollaborationDeclined, payload)
}

func (p *Publisher) PublishBookingApproved(ctx context.Context, payload BookingApprovedPayload) error {
	return p.publish(ctx, TopicBookingApproved, payload)
}

func (p *Publisher) PublishBookingRejected(ctx context.Context, payload BookingRejectedPayload) error {
	return p.publish(ctx, TopicBookingRejected, payload)
}

func (p *Publisher) PublishMemberAdded(ctx context.Context, payload MemberAddedPayload) error {
	return p.publish(ctx, TopicMemberAdded, payload)
}

func (p *Publisher) PublishMemberUpdated(ctx context.Context, payload MemberUpdatedPayload) error {
	return p.publish(ctx, TopicMemberUpdated, payload)
}

func (p *Publisher) PublishMemberRemoved(ctx context.Context, payload MemberRemovedPayload) error {
	return p.publish(ctx, TopicMemberRemoved, payload)
}

func (p *Publisher) PublishRegistrationCreated(ctx context.Context, payload RegistrationCreatedPayload) error {
	return p.publish(ctx, TopicRegistrationCreated, payload)
}

func (p *Publisher) PublishRegistrationCancelled(ctx context.Context, payload RegistrationCancelledPayload) error {
	return p.publish(ctx, TopicRegistrationCancelled, payload)
}
