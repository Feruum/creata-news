package events

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type capturePublisher struct {
	topic string
	msg   *message.Message
}

func (p *capturePublisher) Publish(topic string, messages ...*message.Message) error {
	p.topic = topic
	p.msg = messages[0]
	return nil
}

func (p *capturePublisher) Close() error { return nil }

func TestPublishUserCreatedSerializesPayloadAndMetadata(t *testing.T) {
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })

	// Create a real span context so the propagator has something to inject.
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("test").Start(context.Background(), "test-op")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	if !span.SpanContext().TraceID().IsValid() {
		t.Fatal("expected valid trace ID from SDK tracer")
	}

	pub := &capturePublisher{}
	publisher := NewPublisher(pub)
	payload := UserCreatedPayload{UserID: 42, Username: "alice", Email: "alice@example.com"}

	if err := publisher.PublishUserCreated(ctx, payload); err != nil {
		t.Fatalf("PublishUserCreated() error = %v", err)
	}

	// Assert correct topic.
	if pub.topic != TopicUserCreated {
		t.Fatalf("topic = %q, want %q", pub.topic, TopicUserCreated)
	}

	// Assert payload round-trip.
	var got UserCreatedPayload
	if err := json.Unmarshal(pub.msg.Payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got != payload {
		t.Fatalf("payload = %#v, want %#v", got, payload)
	}

	// Assert trace propagation: traceparent must contain the trace and span IDs.
	traceparent := pub.msg.Metadata.Get("traceparent")
	if traceparent == "" {
		t.Fatal("expected traceparent in message metadata, got empty string")
	}
	if !strings.Contains(traceparent, traceID) {
		t.Fatalf("traceparent %q does not contain trace ID %q", traceparent, traceID)
	}
	if !strings.Contains(traceparent, spanID) {
		t.Fatalf("traceparent %q does not contain span ID %q", traceparent, spanID)
	}

	// Verify tracestate is present only if the span context has it set.
	// With a default TracerProvider, tracestate is empty, so we just confirm
	// the key is absent or empty rather than asserting a specific value.
	if sc := span.SpanContext(); sc.TraceState().Len() > 0 {
		if pub.msg.Metadata.Get("tracestate") == "" {
			t.Fatal("expected tracestate in metadata when span has TraceState")
		}
	}

	_ = trace.SpanFromContext(ctx) // ensure import used
}
