package events

import (
	"context"
	"log/slog"

	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ eventsv1connect.EventsServiceHandler = (*Service)(nil)
var _ eventsv1connect.OrganizationSubscriptionsServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedEventsServiceHandler
	db                     bun.IDB
	perms                  *perms.Client
	authz                  *authz.Enforcer
	storage                storage.Provider
	publisher              *domainevents.Publisher
	mapAppURL              string
	internalReminderSecret string
}

func NewService(
	db *bun.DB,
	permsClient *perms.Client,
	authzEnforcer *authz.Enforcer,
	storageProvider storage.Provider,
	publisher *domainevents.Publisher,
	mapAppURL string,
	internalReminderSecret string,
) *Service {
	return &Service{
		db:                     db,
		perms:                  permsClient,
		authz:                  authzEnforcer,
		storage:                storageProvider,
		publisher:              publisher,
		mapAppURL:              mapAppURL,
		internalReminderSecret: internalReminderSecret,
	}
}

func (s *Service) logPublishFailure(ctx context.Context, topic string, err error) {
	slog.WarnContext(ctx, "failed to publish event", "topic", topic, "error", err)
	trace.SpanFromContext(ctx).AddEvent("publish_failed", trace.WithAttributes(
		attribute.String("event.topic", topic),
		attribute.String("error.message", err.Error()),
	))
}
