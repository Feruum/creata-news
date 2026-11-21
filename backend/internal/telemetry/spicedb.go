package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const spicedbTracerName = "ems-backend/spicedb"

type SpiceDBTracer struct {
	tracer trace.Tracer
}

func NewSpiceDBTracer() *SpiceDBTracer {
	return &SpiceDBTracer{
		tracer: otel.Tracer(spicedbTracerName),
	}
}

func (t *SpiceDBTracer) StartCheckPermissionSpan(ctx context.Context, userID, resourceType, resourceID, permission string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "spicedb.check_permission",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("authz.system", "spicedb"),
			attribute.String("authz.operation", "check_permission"),
			attribute.String("authz.user_id", userID),
			attribute.String("authz.resource_type", resourceType),
			attribute.String("authz.resource_id", resourceID),
			attribute.String("authz.permission", permission),
		),
	)
}

func (t *SpiceDBTracer) StartWriteRelationshipsSpan(ctx context.Context, count int) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "spicedb.write_relationships",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("authz.system", "spicedb"),
			attribute.String("authz.operation", "write_relationships"),
			attribute.Int("authz.relationship_count", count),
		),
	)
}

func (t *SpiceDBTracer) StartLookupResourcesSpan(ctx context.Context, userID, resourceType, permission string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "spicedb.lookup_resources",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("authz.system", "spicedb"),
			attribute.String("authz.operation", "lookup_resources"),
			attribute.String("authz.user_id", userID),
			attribute.String("authz.resource_type", resourceType),
			attribute.String("authz.permission", permission),
		),
	)
}

func (t *SpiceDBTracer) EndSpan(span trace.Span, err error, allowed ...bool) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
		if len(allowed) > 0 {
			span.SetAttributes(attribute.Bool("authz.allowed", allowed[0]))
		}
	}
	span.End()
}
