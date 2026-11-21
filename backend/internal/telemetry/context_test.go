package telemetry

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	collectorlog "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectormetric "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type traceReceiver struct {
	collectortrace.UnimplementedTraceServiceServer
	requests chan *collectortrace.ExportTraceServiceRequest
}

func (r *traceReceiver) Export(_ context.Context, request *collectortrace.ExportTraceServiceRequest) (*collectortrace.ExportTraceServiceResponse, error) {
	r.requests <- request
	return &collectortrace.ExportTraceServiceResponse{}, nil
}

type metricReceiver struct {
	collectormetric.UnimplementedMetricsServiceServer
}

func (metricReceiver) Export(context.Context, *collectormetric.ExportMetricsServiceRequest) (*collectormetric.ExportMetricsServiceResponse, error) {
	return &collectormetric.ExportMetricsServiceResponse{}, nil
}

type logReceiver struct {
	collectorlog.UnimplementedLogsServiceServer
}

func (logReceiver) Export(context.Context, *collectorlog.ExportLogsServiceRequest) (*collectorlog.ExportLogsServiceResponse, error) {
	return &collectorlog.ExportLogsServiceResponse{}, nil
}

func TestBrowserTraceContextContinuesThroughConnectAndOutgoingHTTP(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	receiver := &traceReceiver{requests: make(chan *collectortrace.ExportTraceServiceRequest, 4)}
	server := grpc.NewServer()
	collectortrace.RegisterTraceServiceServer(server, receiver)
	collectormetric.RegisterMetricsServiceServer(server, metricReceiver{})
	collectorlog.RegisterLogsServiceServer(server, logReceiver{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	previousTracer, previousMeter, previousPropagator := otel.GetTracerProvider(), otel.GetMeterProvider(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracer)
		otel.SetMeterProvider(previousMeter)
		otel.SetTextMapPropagator(previousPropagator)
	})
	providers, err := New(Config{Enabled: true, ServiceName: "ems-backend", ServiceVersion: "test", Environment: "test", OTLPEndpoint: listener.Addr().String()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := providers.Shutdown(ctx); err != nil {
			t.Errorf("telemetry shutdown: %v", err)
		}
	})
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const traceState = "studyverse=browser"
	outgoingHeaders := make(chan http.Header, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		outgoingHeaders <- request.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote(), otelconnect.WithoutServerPeerAttributes())
	if err != nil {
		t.Fatal(err)
	}
	var handlerContext trace.SpanContext
	router := chi.NewRouter()
	router.Use(otelchi.Middleware("ems-backend", otelchi.WithChiRoutes(router)))
	router.Handle("/test.ContextService/Ping", connect.NewUnaryHandler("/test.ContextService/Ping",
		func(ctx context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			handlerContext = trace.SpanContextFromContext(ctx)
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream.URL, nil)
			if err != nil {
				return nil, err
			}
			response, err := NewHTTPClient().Do(request)
			if err != nil {
				return nil, err
			}
			_ = response.Body.Close()
			return connect.NewResponse(&emptypb.Empty{}), nil
		}, connect.WithInterceptors(interceptor)))
	request := httptest.NewRequest(http.MethodPost, "/test.ContextService/Ping", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	request.Header.Set("Traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	request.Header.Set("Tracestate", traceState)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Connect response: %d %s", response.Code, response.Body.String())
	}
	if handlerContext.TraceID().String() != traceID || handlerContext.TraceState().String() != traceState {
		t.Fatalf("browser context was not preserved: %v", handlerContext)
	}
	select {
	case headers := <-outgoingHeaders:
		if !strings.HasPrefix(headers.Get("Traceparent"), "00-"+traceID+"-") || headers.Get("Tracestate") != traceState {
			t.Fatalf("outgoing context was not propagated: %v", headers)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("outgoing HTTP call was not observed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := providers.TracerProvider.ForceFlush(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case exported := <-receiver.requests:
		spans := 0
		wantTraceID := handlerContext.TraceID()
		for _, resource := range exported.ResourceSpans {
			for _, scope := range resource.ScopeSpans {
				for _, span := range scope.Spans {
					spans++
					if string(span.TraceId) != string(wantTraceID[:]) {
						t.Fatal("exported span belongs to a different trace")
					}
				}
			}
		}
		if spans < 3 {
			t.Fatalf("expected HTTP server, Connect and outgoing HTTP spans, got %d", spans)
		}
	case <-ctx.Done():
		t.Fatal("trace export was not observed")
	}
}
