package httpx

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/foundation/resilience"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func installRecorder(t *testing.T) (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	})
	return recorder, tp
}

func TestTracing_CreatesClientSpanPerAttemptAndInjectsHeaders(t *testing.T) {
	recorder, tp := installRecorder(t)

	var calls int32
	var traceparents []string
	c := NewClient(Config{
		Tracing: true,
		Retry:   policyPtr(resilience.Constant(2, time.Millisecond)),
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			traceparents = append(traceparents, req.Header.Get("traceparent"))
			if atomic.AddInt32(&calls, 1) == 1 {
				return newResponse(req, http.StatusServiceUnavailable, "busy"), nil
			}
			return newResponse(req, http.StatusOK, "ok"), nil
		}),
	})

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://api.example.com/v1/items?token=secret", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	parent.End()

	var clientSpans []sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.SpanKind() == oteltrace.SpanKindClient {
			clientSpans = append(clientSpans, s)
		}
	}
	if len(clientSpans) != 2 {
		t.Fatalf("client spans = %d, want one per attempt", len(clientSpans))
	}
	for i, s := range clientSpans {
		if s.Parent().SpanID() != parent.SpanContext().SpanID() {
			t.Errorf("attempt %d is not a child of the caller span", i)
		}
		if !strings.Contains(traceparents[i], s.SpanContext().SpanID().String()) {
			t.Errorf("attempt %d traceparent %q does not carry its span", i, traceparents[i])
		}
	}
}

func TestLogging_WritesOneRecordPerAttemptWithoutQuery(t *testing.T) {
	_, tp := installRecorder(t)
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	c := NewClient(Config{
		Tracing: true,
		Logging: true,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return newResponse(req, http.StatusBadGateway, "bad"), nil
		}),
	})
	ctx, span := tp.Tracer("test").Start(context.Background(), "parent")
	defer span.End()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.example.com/v1/send?api_key=secret", nil)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()

	out := buf.String()
	for _, want := range []string{
		`"level":"ERROR"`,
		`"component":"http_client"`,
		`"host":"api.example.com"`,
		`"path":"/v1/send"`,
		`"status_code":502`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %q", want, out)
		}
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("query string leaked: %q", out)
	}
}

func policyPtr(p resilience.Policy) *resilience.Policy { return &p }
