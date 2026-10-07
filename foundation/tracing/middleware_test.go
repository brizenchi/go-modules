package tracing

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/foundation/ginx"
	flog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTraceSetsTraceIDsForAccessLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shutdown, err := Setup(Config{ServiceName: "svc", SampleRate: 1})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown(context.Background(), shutdown)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := gin.New()
	r.Use(ginx.RequestID(), Trace("svc"), ginx.AccessLog(ginx.AccessLogConfig{}))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	out := buf.String()
	if !strings.Contains(out, `"trace_id":`) {
		t.Fatalf("missing trace_id in access log: %q", out)
	}
	if !strings.Contains(out, `"span_id":`) {
		t.Fatalf("missing span_id in access log: %q", out)
	}
	if !strings.Contains(out, `"request_id":`) {
		t.Fatalf("missing request_id in access log: %q", out)
	}
}

func TestTraceAlsoEnrichesBusinessLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	shutdown, err := Setup(Config{ServiceName: "svc", SampleRate: 1})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	defer Shutdown(context.Background(), shutdown)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := gin.New()
	r.Use(ginx.RequestID(), Trace("svc"), ginx.AccessLog(ginx.AccessLogConfig{}))
	r.GET("/x", func(c *gin.Context) {
		slog.InfoContext(c.Request.Context(), "business event", "component", "handler")
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	out := buf.String()
	if !strings.Contains(out, `"msg":"business event"`) {
		t.Fatalf("missing business log: %q", out)
	}
	if !strings.Contains(out, `"trace_id":`) {
		t.Fatalf("missing trace_id in business/access logs: %q", out)
	}
	if !strings.Contains(out, `"request_id":`) {
		t.Fatalf("missing request_id in business/access logs: %q", out)
	}
}

func newRecordingProvider(t *testing.T, sampleRate float64) *tracetest.SpanRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	shutdown, err := Setup(Config{ServiceName: "svc", SampleRate: sampleRate})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	t.Cleanup(func() { Shutdown(context.Background(), shutdown) })

	recorder := tracetest.NewSpanRecorder()
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.ParentBased(tracesdk.TraceIDRatioBased(sampleRate))),
		tracesdk.WithSpanProcessor(requestIDProcessor{}),
		tracesdk.WithSpanProcessor(recorder),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	})
	return recorder
}

func attrMap(span tracesdk.ReadOnlySpan) map[string]string {
	out := map[string]string{}
	for _, kv := range span.Attributes() {
		out[string(kv.Key)] = kv.Value.String()
	}
	return out
}

func TestMiddlewareUsesHTTPSemanticConventions(t *testing.T) {
	recorder := newRecordingProvider(t, 1)

	r := gin.New()
	r.Use(ginx.RequestID(), Middleware(MiddlewareConfig{ServiceName: "svc"}))
	r.GET("/users/:id", func(c *gin.Context) { c.String(http.StatusNotFound, "missing") })

	req := httptest.NewRequest(http.MethodGet, "/users/42?token=secret", nil)
	req.Header.Set(ginx.HeaderRequestID, "rid-span")
	r.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d", len(spans))
	}
	span := spans[0]
	if span.Name() != "GET /users/:id" {
		t.Fatalf("span name = %q", span.Name())
	}
	if span.SpanKind() != oteltrace.SpanKindServer {
		t.Fatalf("span kind = %v", span.SpanKind())
	}
	attrs := attrMap(span)
	for key, want := range map[string]string{
		"http.request.method":       "GET",
		"http.route":                "/users/:id",
		"http.response.status_code": "404",
		"url.path":                  "/users/42",
		string(RequestIDAttribute):  "rid-span",
	} {
		if attrs[key] != want {
			t.Errorf("%s = %q, want %q (all: %v)", key, attrs[key], want, attrs)
		}
	}
	for _, legacy := range []string{"http.method", "http.status_code"} {
		if _, ok := attrs[legacy]; ok {
			t.Errorf("legacy attribute %s still emitted", legacy)
		}
	}
	if span.Status().Code == codes.Error {
		t.Fatal("4xx must not mark a server span as error")
	}
}

func TestMiddlewareFollowsParentSamplingDecision(t *testing.T) {
	recorder := newRecordingProvider(t, 0)

	r := gin.New()
	r.Use(Middleware(MiddlewareConfig{ServiceName: "svc"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	r.ServeHTTP(httptest.NewRecorder(), req)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("sampled parent must be honoured even with SampleRate=0, spans = %d", len(spans))
	}
	if got := spans[0].SpanContext().TraceID().String(); got != traceID {
		t.Fatalf("trace id = %s, want %s", got, traceID)
	}
}

func TestMiddlewareSkipPaths(t *testing.T) {
	recorder := newRecordingProvider(t, 1)

	r := gin.New()
	r.Use(Middleware(MiddlewareConfig{ServiceName: "svc", SkipPaths: []string{"/health"}}))
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	if n := len(recorder.Ended()); n != 0 {
		t.Fatalf("/health produced %d spans", n)
	}
}

func TestMiddlewareRecordsRecoveredPanicAsError(t *testing.T) {
	recorder := newRecordingProvider(t, 1)
	slog.SetDefault(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))

	r := gin.New()
	r.Use(ginx.RequestID(), Trace("svc"), ginx.AccessLog(ginx.AccessLogConfig{}), ginx.Recover())
	r.GET("/boom", func(c *gin.Context) { panic("boom") })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d", len(spans))
	}
	if spans[0].Status().Code != codes.Error {
		t.Fatalf("status = %+v", spans[0].Status())
	}
	if got := attrMap(spans[0])["http.response.status_code"]; got != "500" {
		t.Fatalf("status code attr = %q", got)
	}
}

func TestAccessLogHasSingleCorrelationFields(t *testing.T) {
	newRecordingProvider(t, 1)
	var buf bytes.Buffer
	flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})

	r := gin.New()
	r.Use(ginx.RequestID(), Trace("svc"), ginx.AccessLog(ginx.AccessLogConfig{}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	out := buf.String()
	for _, key := range []string{`"request_id"`, `"trace_id"`, `"span_id"`} {
		if n := strings.Count(out, key); n != 1 {
			t.Fatalf("%s appears %d times: %q", key, n, out)
		}
	}
}
