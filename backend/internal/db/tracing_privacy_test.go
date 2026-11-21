//go:build integration

package db

import (
	"context"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDatabaseTracingDoesNotExportBoundSecrets(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL requires disposable database")
	}
	ctx := context.Background()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	defer otel.SetTracerProvider(old)
	defer provider.Shutdown(ctx)
	ctx, parent := provider.Tracer("db-privacy-regression").Start(ctx, "database privacy check")
	defer parent.End()
	// Configure the provider before creating the pool: pgx captures it at NewTracer.
	pool, err := NewPool(ctx, dsn, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	db := NewBunDB(pool, true)
	defer db.Close()
	secrets := []string{"private-email@test.local", "private-password-digest-sentinel", "private-session-digest-sentinel"}
	for _, secret := range secrets {
		var value string
		if err := db.NewRaw("SELECT ?::text", secret).Scan(ctx, &value); err != nil {
			t.Fatal("sensitive fixture query failed")
		}
		if value != secret {
			t.Fatal("fixture query did not return expected value")
		}
	}
	// Exercise server preparation and preserve useful non-sensitive SQL diagnostics.
	if _, err := pool.Exec(ctx, "SELECT 1/0"); err == nil {
		t.Fatal("expected diagnostic fixture failure")
	}
	var pgxSpan, bunSpan, errorStatus, sqlState bool
	for _, span := range recorder.Ended() {
		scope := span.InstrumentationScope().Name
		pgxSpan = pgxSpan || strings.Contains(scope, "otelpgx")
		bunSpan = bunSpan || strings.Contains(scope, "bun")
		errorStatus = errorStatus || span.Status().Code == codes.Error
		check := func(value string) {
			t.Helper()
			for _, secret := range secrets {
				if strings.Contains(value, secret) {
					t.Fatal("database tracing exported a sensitive SQL value")
				}
			}
		}
		check(span.Name())
		check(span.Status().Description)
		for _, attr := range span.Attributes() {
			check(attr.Value.Emit())
			if string(attr.Key) == "pgx.sql_state" && attr.Value.AsString() == "22012" {
				sqlState = true
			}
		}
		for _, event := range span.Events() {
			check(event.Name)
			for _, attr := range event.Attributes {
				check(attr.Value.Emit())
			}
		}
	}
	if !pgxSpan || !bunSpan {
		t.Fatal("both database tracing layers must be enabled")
	}
	if !errorStatus || !sqlState {
		t.Fatal("safe SQL error diagnostics were lost")
	}
}
