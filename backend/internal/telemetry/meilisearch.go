package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const meilisearchTracerName = "ems-backend/meilisearch"

type MeilisearchTracer struct {
	tracer trace.Tracer
}

func NewMeilisearchTracer() *MeilisearchTracer {
	return &MeilisearchTracer{
		tracer: otel.Tracer(meilisearchTracerName),
	}
}

func (t *MeilisearchTracer) StartIndexSpan(ctx context.Context, operation, indexName string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "meilisearch."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "meilisearch"),
			attribute.String("db.operation", operation),
			attribute.String("meilisearch.index", indexName),
		),
	)
}

func (t *MeilisearchTracer) StartSearchSpan(ctx context.Context, indexName, query string, limit int64) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "meilisearch.search",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "meilisearch"),
			attribute.String("db.operation", "search"),
			attribute.String("meilisearch.index", indexName),
			attribute.String("meilisearch.query", query),
			attribute.Int64("meilisearch.limit", limit),
		),
	)
}

func (t *MeilisearchTracer) EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

func (t *MeilisearchTracer) SetHits(span trace.Span, hits int64) {
	span.SetAttributes(attribute.Int64("meilisearch.hits", hits))
}
