package events

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// MetadataCarrier adapts message.Metadata to the OTel TextMapCarrier interface.
type MetadataCarrier message.Metadata

func (c MetadataCarrier) Get(key string) string { return message.Metadata(c)[key] }
func (c MetadataCarrier) Set(key, value string) { message.Metadata(c)[key] = value }
func (c MetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}

// TraceMiddleware extracts the propagated trace context from message metadata
// and creates a consumer span around the handler invocation.
func TraceMiddleware(operation string) message.HandlerMiddleware {
	tracer := otel.Tracer("ems-backend/internal/events")

	return func(next message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := msg.Context()
			ctx = otel.GetTextMapPropagator().Extract(ctx, MetadataCarrier(msg.Metadata))
			ctx, span := tracer.Start(ctx, operation,
				trace.WithSpanKind(trace.SpanKindConsumer),
				trace.WithAttributes(
				attribute.String("messaging.system", "redis"),
					attribute.String("messaging.destination.name", operation),
				),
			)
			defer span.End()

			msg.SetContext(ctx)
			out, err := next(msg)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return out, err
		}
	}
}

// SlogMiddleware logs handler entry, success, and failure using structured logging.
func SlogMiddleware(handlerName string) message.HandlerMiddleware {
	return func(next message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := msg.Context()
			slog.DebugContext(ctx, "handling domain event", "handler", handlerName, "message_id", msg.UUID)
			out, err := next(msg)
			if err != nil {
				slog.ErrorContext(ctx, "domain event handler failed", "handler", handlerName, "message_id", msg.UUID, "error", err)
				return out, fmt.Errorf("%s: %w", handlerName, err)
			}
			slog.DebugContext(ctx, "domain event handler completed", "handler", handlerName, "message_id", msg.UUID)
			return out, nil
		}
	}
}
