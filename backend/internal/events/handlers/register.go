package handlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/search"
	"github.com/uptrace/bun"
)

// SearchClient defines the search operations used by domain event handlers.
// This interface allows for easy testing with fakes.
type SearchClient interface {
	IndexUser(ctx context.Context, doc *search.UserDocument) error
	IndexOrganization(ctx context.Context, doc *search.OrganizationDocument) error
	IndexEvent(ctx context.Context, doc *search.EventDocument) error
	DeleteDocument(ctx context.Context, indexName string, id int32) error
}

// HandlerDeps holds shared dependencies for all domain event handlers.
type HandlerDeps struct {
	DB                         *bun.DB
	SearchClient               SearchClient
	TelegramBotURL             string
	InternalNotificationSecret string
	MapAppURL                  string
	Perms                      *perms.Client
	OnUserCreated              func()
}

// registerNoPublisher is a small helper that registers a no-publisher handler
// and immediately wires TraceMiddleware + SlogMiddleware onto it.
func registerNoPublisher(
	router *message.Router,
	name string,
	topic string,
	sub message.Subscriber,
	handlerFunc message.NoPublishHandlerFunc,
) {
	h := router.AddNoPublisherHandler(name, topic, sub, handlerFunc)
	h.AddMiddleware(
		events.TraceMiddleware(name),
		events.SlogMiddleware(name),
	)
}

// RegisterHandlers wires all domain event handlers to the Watermill router
// using AddNoPublisherHandler with unique handler names per subscriber group.
func RegisterHandlers(
	router *message.Router,
	deps HandlerDeps,
	searchSub, telegramSub, activitySub, historySub message.Subscriber,
) {
	// --- search-indexer group (9 handlers) ---
	registerNoPublisher(router, "search.user.created", events.TopicUserCreated, searchSub, handleUserCreated(deps))
	registerNoPublisher(router, "search.user.updated", events.TopicUserUpdated, searchSub, handleUserUpdated(deps))
	registerNoPublisher(router, "search.user.deleted", events.TopicUserDeleted, searchSub, handleUserDeleted(deps))
	registerNoPublisher(router, "search.organization.created", events.TopicOrganizationCreated, searchSub, handleOrganizationCreated(deps))
	registerNoPublisher(router, "search.organization.updated", events.TopicOrganizationUpdated, searchSub, handleOrganizationUpdated(deps))
	registerNoPublisher(router, "search.organization.deleted", events.TopicOrganizationDeleted, searchSub, handleOrganizationDeleted(deps))
	registerNoPublisher(router, "search.event.created", events.TopicEventCreated, searchSub, handleEventCreated(deps))
	registerNoPublisher(router, "search.event.updated", events.TopicEventUpdated, searchSub, handleEventUpdated(deps))
	registerNoPublisher(router, "search.event.deleted", events.TopicEventDeleted, searchSub, handleEventDeleted(deps))
	registerNoPublisher(router, "search.event.published", events.TopicEventPublished, searchSub, handleEventPublishedSearch(deps))
	registerNoPublisher(router, "search.event.archived", events.TopicEventArchived, searchSub, handleEventArchivedSearch(deps))

	// --- telegram-notifier group (6 handlers) ---
	registerNoPublisher(router, "telegram.collaboration.invited", events.TopicCollaborationInvited, telegramSub, handleCollaborationInvited(deps))
	registerNoPublisher(router, "telegram.collaboration.accepted", events.TopicCollaborationAccepted, telegramSub, handleCollaborationAccepted(deps))
	registerNoPublisher(router, "telegram.collaboration.declined", events.TopicCollaborationDeclined, telegramSub, handleCollaborationDeclined(deps))
	registerNoPublisher(router, "telegram.booking.approved", events.TopicBookingApproved, telegramSub, handleBookingApproved(deps))
	registerNoPublisher(router, "telegram.booking.rejected", events.TopicBookingRejected, telegramSub, handleBookingRejected(deps))
	registerNoPublisher(router, "telegram.event.published", events.TopicEventPublished, telegramSub, handleEventPublished(deps))

	// --- activity-logger group (14 handlers) ---
	registerNoPublisher(router, "activity.organization.created", events.TopicOrganizationCreated, activitySub, handleOrganizationCreatedActivity(deps))
	registerNoPublisher(router, "activity.organization.updated", events.TopicOrganizationUpdated, activitySub, handleOrganizationUpdatedActivity(deps))
	registerNoPublisher(router, "activity.organization.deleted", events.TopicOrganizationDeleted, activitySub, handleOrganizationDeletedActivity(deps))
	registerNoPublisher(router, "activity.event.created", events.TopicEventCreated, activitySub, handleEventCreatedActivity(deps))
	registerNoPublisher(router, "activity.event.updated", events.TopicEventUpdated, activitySub, handleEventUpdatedActivity(deps))
	registerNoPublisher(router, "activity.event.deleted", events.TopicEventDeleted, activitySub, handleEventDeletedActivity(deps))
	registerNoPublisher(router, "activity.event.published", events.TopicEventPublished, activitySub, handleEventPublishedActivity(deps))
	registerNoPublisher(router, "activity.booking.approved", events.TopicBookingApproved, activitySub, handleBookingApprovedActivity(deps))
	registerNoPublisher(router, "activity.booking.rejected", events.TopicBookingRejected, activitySub, handleBookingRejectedActivity(deps))
	registerNoPublisher(router, "activity.member.added", events.TopicMemberAdded, activitySub, handleMemberAddedActivity(deps))
	registerNoPublisher(router, "activity.member.updated", events.TopicMemberUpdated, activitySub, handleMemberUpdatedActivity(deps))
	registerNoPublisher(router, "activity.member.removed", events.TopicMemberRemoved, activitySub, handleMemberRemovedActivity(deps))
	registerNoPublisher(router, "activity.registration.created", events.TopicRegistrationCreated, activitySub, handleRegistrationCreatedActivity(deps))
	registerNoPublisher(router, "activity.registration.cancelled", events.TopicRegistrationCancelled, activitySub, handleRegistrationCancelledActivity(deps))

	// --- membership-historian group (3 handlers) ---
	registerNoPublisher(router, "history.member.added", events.TopicMemberAdded, historySub, handleMemberAddedHistory(deps))
	registerNoPublisher(router, "history.member.updated", events.TopicMemberUpdated, historySub, handleMemberUpdatedHistory(deps))
	registerNoPublisher(router, "history.member.removed", events.TopicMemberRemoved, historySub, handleMemberRemovedHistory(deps))
}
