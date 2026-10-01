package tracing

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestSetupRequiresServiceName(t *testing.T) {
	_, err := Setup(Config{})
	if err == nil {
		t.Fatal("expected error when service name is empty")
	}
}

func TestSetupWithoutEndpointReturnsNoopShutdown(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
	if otel.GetTextMapPropagator() == nil {
		t.Fatal("expected global propagator to be initialized")
	}
}

func TestSetupWithoutEndpointStillCreatesTraceIDsWhenSampled(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc", SampleRate: 1})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown(context.Background(), shutdown)

	_, span := otel.Tracer("svc").Start(context.Background(), "test", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	if !span.SpanContext().HasTraceID() {
		t.Fatal("expected valid trace id even when exporter is disabled")
	}
	span.End()
}

func TestShutdownHandlesError(t *testing.T) {
	var called bool
	Shutdown(context.Background(), func(context.Context) error {
		called = true
		return errors.New("boom")
	})
	if !called {
		t.Fatal("expected shutdown function to be called")
	}
}

func TestSetupMetricsWithoutEndpointIsNoop(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc", Metrics: true})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

func TestSetupWithMetricsExporter(t *testing.T) {
	shutdown, err := Setup(Config{
		ServiceName: "svc",
		Endpoint:    "127.0.0.1:4318",
		Insecure:    true,
		Metrics:     true,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = shutdown(ctx) // nothing listens; only construction is under test
}

func TestSetupInstallsBaggagePropagator(t *testing.T) {
	shutdown, err := Setup(Config{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown(context.Background(), shutdown)

	fields := otel.GetTextMapPropagator().Fields()
	want := map[string]bool{"traceparent": false, "baggage": false}
	for _, f := range fields {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for f, ok := range want {
		if !ok {
			t.Errorf("propagator missing %s (fields: %v)", f, fields)
		}
	}
}

func TestMetricsURLPath(t *testing.T) {
	for _, tc := range []struct {
		cfg  Config
		want string
	}{
		{Config{}, "/v1/metrics"},
		{Config{URLPath: "/api/default/v1/traces"}, "/api/default/v1/metrics"},
		{Config{URLPath: "/custom"}, "/v1/metrics"},
		{Config{URLPath: "/api/default/v1/traces", MetricsURLPath: "/m"}, "/m"},
	} {
		if got := metricsURLPath(tc.cfg); got != tc.want {
			t.Errorf("metricsURLPath(%+v) = %q, want %q", tc.cfg, got, tc.want)
		}
	}
}

func TestResourceHonoursEnvironment(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "team=growth")
	res, err := newResource(Config{ServiceName: "svc", Project: "proj", Environment: "prod", ServiceVersion: "1.2.3"})
	if err != nil {
		t.Fatalf("newResource() error = %v", err)
	}
	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.Emit()
	}
	for key, want := range map[string]string{
		"service.name":                "svc",
		"service.version":             "1.2.3",
		"service.namespace":           "proj",
		"deployment.environment.name": "prod",
		"team":                        "growth",
		"telemetry.sdk.language":      "go",
	} {
		if got[key] != want {
			t.Errorf("%s = %q, want %q", key, got[key], want)
		}
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	for _, tc := range []struct {
		in           Config
		wantEndpoint string
		wantInsecure bool
		wantErr      bool
	}{
		{in: Config{Endpoint: "localhost:4318"}, wantEndpoint: "localhost:4318"},
		{in: Config{Endpoint: "localhost:4318", Insecure: true}, wantEndpoint: "localhost:4318", wantInsecure: true},
		{in: Config{Endpoint: "http://collector:5080"}, wantEndpoint: "collector:5080", wantInsecure: true},
		{in: Config{Endpoint: "https://otel.example.com", Insecure: true}, wantEndpoint: "otel.example.com"},
		{in: Config{Endpoint: "ftp://x:1"}, wantErr: true},
		{in: Config{Endpoint: "http://"}, wantErr: true},
	} {
		got, err := normalizeEndpoint(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizeEndpoint(%q) expected error", tc.in.Endpoint)
			}
			continue
		}
		if err != nil || got.Endpoint != tc.wantEndpoint || got.Insecure != tc.wantInsecure {
			t.Errorf("normalizeEndpoint(%q) = (%q, insecure=%v, %v)", tc.in.Endpoint, got.Endpoint, got.Insecure, err)
		}
	}
}
