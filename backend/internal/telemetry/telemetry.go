package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Enabled        bool
}

// Providers holds all three OTEL signal providers. Any field may be nil if telemetry is disabled.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
	config         Config
}

func New(cfg Config) (*Providers, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		slog.Info("Telemetry disabled", "enabled", cfg.Enabled, "endpoint", cfg.OTLPEndpoint)
		return &Providers{config: cfg}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(cfg.OTLPEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, err
	}

	// --- Traces ---
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// --- Metrics ---
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
	)

	// --- Logs ---
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)

	// Set global providers so instrumentation libraries (otelconnect, bunotel, otelhttp, etc.) pick them up
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	// Note: LoggerProvider is NOT set globally — it's passed explicitly to otelslog in main.go

	slog.Info("Telemetry initialized",
		"service", cfg.ServiceName,
		"endpoint", cfg.OTLPEndpoint,
	)

	return &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
		config:         cfg,
	}, nil
}

// Shutdown gracefully shuts down all providers with the given context deadline.
func (p *Providers) Shutdown(ctx context.Context) error {
	var errs []error
	if p.TracerProvider != nil {
		errs = append(errs, p.TracerProvider.Shutdown(ctx))
	}
	if p.MeterProvider != nil {
		errs = append(errs, p.MeterProvider.Shutdown(ctx))
	}
	if p.LoggerProvider != nil {
		errs = append(errs, p.LoggerProvider.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

// Tracer returns a named tracer from the provider, falling back to the global tracer if disabled.
func (p *Providers) Tracer(name string) trace.Tracer {
	if p.TracerProvider == nil {
		return otel.Tracer(name)
	}
	return p.TracerProvider.Tracer(name)
}

// IsEnabled returns true if telemetry providers were successfully initialized.
func (p *Providers) IsEnabled() bool {
	return p.TracerProvider != nil
}
