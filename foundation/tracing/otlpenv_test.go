package tracing

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
)

// fakeCollector records "path auth" for every OTLP request it receives.
type fakeCollector struct {
	mu   sync.Mutex
	hits []string
	srv  *httptest.Server
}

func newFakeCollector(t *testing.T) *fakeCollector {
	t.Helper()
	fc := &fakeCollector{}
	fc.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fc.mu.Lock()
		fc.hits = append(fc.hits, r.URL.Path+" "+r.Header.Get("Authorization"))
		fc.mu.Unlock()
	}))
	t.Cleanup(fc.srv.Close)
	return fc
}

func (fc *fakeCollector) received(path string) string {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	for _, h := range fc.hits {
		if p, auth, _ := strings.Cut(h, " "); p == path {
			return auth
		}
	}
	return "<none>"
}

func exportOnce(t *testing.T, cfg Config) {
	t.Helper()
	shutdown, err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	_, span := otel.Tracer("t").Start(context.Background(), "op")
	span.End()
	counter, _ := otel.Meter("t").Int64Counter("probe")
	counter.Add(context.Background(), 1)
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown() error = %v", err)
	}
}

// TestStandardOTELEnvironment mirrors a typical Grafana Cloud setup that
// works for any OpenTelemetry SDK.
func TestStandardOTELEnvironment(t *testing.T) {
	fc := newFakeCollector(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", fc.srv.URL+"/otlp")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Basic%20c2VjcmV0")

	exportOnce(t, Config{ServiceName: "svc", SampleRate: 1, Metrics: true})

	if got := fc.received("/otlp/v1/traces"); got != "Basic c2VjcmV0" {
		t.Errorf("traces auth = %q", got)
	}
	if got := fc.received("/otlp/v1/metrics"); got != "Basic c2VjcmV0" {
		t.Errorf("metrics auth = %q", got)
	}
}

func TestConfigEndpointTakesPrecedenceButKeepsEnvHeaders(t *testing.T) {
	fc := newFakeCollector(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1/unused")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Basic%20ZW52")

	exportOnce(t, Config{ServiceName: "svc", SampleRate: 1, Endpoint: fc.srv.URL, URLPath: "/custom/v1/traces"})

	if got := fc.received("/custom/v1/traces"); got != "Basic ZW52" {
		t.Errorf("traces auth = %q", got)
	}
}

func TestResolveTargetProtocolFromEnvironment(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://collector:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	got := resolveTarget(Config{})
	if !got.enabled || !got.fromEnv || got.protocol != "grpc" {
		t.Fatalf("resolveTarget() = %+v", got)
	}
	if resolveTarget(Config{Endpoint: "x:4318"}).fromEnv {
		t.Fatal("explicit Endpoint must win over the environment")
	}
}

func TestDiagnosticsForMissingOrQuotedAuth(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", `"Authorization=Basic%20abc"`)
	if hasAuthHeader(Config{}) {
		t.Fatal("quoted header must not count as configured")
	}
	shutdown, err := Setup(Config{ServiceName: "svc"})
	if err != nil {
		t.Fatal(err)
	}
	_ = shutdown(context.Background())

	out := buf.String()
	for _, want := range []string{
		`"variable":"OTEL_EXPORTER_OTLP_HEADERS"`,
		`no Authorization header configured`,
		`"auth_header":false`,
		`"endpoint_source":"OTEL_EXPORTER_OTLP_*"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}
