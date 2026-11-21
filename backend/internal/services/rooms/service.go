package rooms

import (
	"context"
	"log/slog"

	"github.com/studyverse/ems-backend/gen/rooms/v1/roomsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ roomsv1connect.RoomsServiceHandler = (*Service)(nil)

type Service struct {
	roomsv1connect.UnimplementedRoomsServiceHandler
	db        bun.IDB
	perms     *perms.Client
	authz     *authz.Enforcer
	publisher *domainevents.Publisher
}

func NewService(db *bun.DB, permsClient *perms.Client, authzEnforcer *authz.Enforcer, publisher *domainevents.Publisher) *Service {
	return &Service{db: db, perms: permsClient, authz: authzEnforcer, publisher: publisher}
}

func (s *Service) logPublishFailure(ctx context.Context, topic string, err error) {
	slog.WarnContext(ctx, "failed to publish event", "topic", topic, "error", err)
	trace.SpanFromContext(ctx).AddEvent("publish_failed", trace.WithAttributes(
		attribute.String("event.topic", topic),
		attribute.String("error.message", err.Error()),
	))
}
