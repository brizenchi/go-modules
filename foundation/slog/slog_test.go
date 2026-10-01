package slog

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestSetup_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})
	slog.Info("hello", "k", "v")
	out := buf.String()
	if !strings.Contains(out, `"msg":"hello"`) || !strings.Contains(out, `"k":"v"`) {
		t.Errorf("output not JSON-shaped: %q", out)
	}
}

func TestSetup_TextOutput(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatText, Output: &buf})
	slog.Info("hello", "k", "v")
	out := buf.String()
	if strings.Contains(out, `"msg"`) {
		t.Errorf("expected text format, got JSON: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("missing message: %q", out)
	}
}

func TestSetup_LevelFilter(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "warn", Format: FormatJSON, Output: &buf})
	slog.Info("nope")
	slog.Warn("yes")
	out := buf.String()
	if strings.Contains(out, "nope") {
		t.Error("info log leaked through level=warn filter")
	}
	if !strings.Contains(out, "yes") {
		t.Error("warn log dropped under level=warn")
	}
}

func TestSetup_DefaultAttrsAlwaysEmitted(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{
		Level:    "info",
		Format:   FormatJSON,
		Output:   &buf,
		Defaults: map[string]any{"service": "billing"},
	})
	slog.Info("x")
	if !strings.Contains(buf.String(), `"service":"billing"`) {
		t.Errorf("default attr missing: %q", buf.String())
	}
}

func TestWith_GinContextReadsRequestID(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("request_id", "rid-123")

	With(c).Info("hi")
	if !strings.Contains(buf.String(), `"request_id":"rid-123"`) {
		t.Errorf("missing request_id: %q", buf.String())
	}
}

func TestWith_FallsBackToContextValue(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := context.WithValue(context.Background(), RequestIDKey, "rid-456")
	c.Request = httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	With(c).Info("hi")
	if !strings.Contains(buf.String(), `"request_id":"rid-456"`) {
		t.Errorf("missing request_id from ctx: %q", buf.String())
	}
}

func TestWith_NilContextSafe(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})
	With(nil).Info("hi") // must not panic
}

func TestSetup_ContextAttrsInjectedAutomatically(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "rid-ctx")
	ctx = context.WithValue(ctx, ProjectKey, "proj-1")
	ctx = context.WithValue(ctx, EnvKey, "prod")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-9")
	ctx = context.WithValue(ctx, UserIDKey, "user-7")

	slog.InfoContext(ctx, "hello")
	out := buf.String()
	for _, want := range []string{
		`"request_id":"rid-ctx"`,
		`"project":"proj-1"`,
		`"env":"prod"`,
		`"tenant_id":"tenant-9"`,
		`"user_id":"user-7"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %q", want, out)
		}
	}
}

func TestSetup_ContextAttrsIncludeTraceAndSpan(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	ctx, span := startSpan(t)
	defer span.End()

	slog.InfoContext(ctx, "hello")
	out := buf.String()
	if !strings.Contains(out, `"trace_id":"`) {
		t.Fatalf("missing trace_id in %q", out)
	}
	if !strings.Contains(out, `"span_id":"`) {
		t.Fatalf("missing span_id in %q", out)
	}
	if !strings.Contains(out, `"trace_flags":"01"`) {
		t.Fatalf("missing trace_flags in %q", out)
	}
}

func startSpan(t *testing.T) (context.Context, oteltrace.Span) {
	t.Helper()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return tp.Tracer("svc").Start(context.Background(), "test")
}

func TestSetup_CallerAttrsAreNotDuplicated(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	ctx, span := startSpan(t)
	defer span.End()
	ctx = ContextWithRequestID(ctx, "rid-ctx")
	traceID := span.SpanContext().TraceID().String()

	slog.InfoContext(ctx, "explicit", "request_id", "rid-ctx", "trace_id", traceID)
	slog.Default().With("request_id", "rid-ctx").InfoContext(ctx, "bound")

	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if n := strings.Count(line, `"request_id"`); n != 1 {
			t.Fatalf("request_id appears %d times in %q", n, line)
		}
		if n := strings.Count(line, `"trace_id"`); n != 1 {
			t.Fatalf("trace_id appears %d times in %q", n, line)
		}
	}
}

func TestWith_DoesNotDuplicateWhenLoggingWithContext(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx, span := startSpan(t)
	defer span.End()
	c.Request = httptest.NewRequest("GET", "/", nil).WithContext(ContextWithRequestID(ctx, "rid-1"))

	With(c).InfoContext(c.Request.Context(), "hi")
	out := buf.String()
	if strings.Count(out, `"request_id"`) != 1 || strings.Count(out, `"span_id"`) != 1 {
		t.Fatalf("duplicate correlation fields: %q", out)
	}
}

func TestRequestIDContextRoundTrip(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "rid-9")
	if got := RequestIDFromContext(ctx); got != "rid-9" {
		t.Fatalf("RequestIDFromContext() = %q", got)
	}
	if got := RequestIDFromContext(nil); got != "" { //nolint:staticcheck // nil ctx must be safe
		t.Fatalf("RequestIDFromContext(nil) = %q", got)
	}
}

func TestSetup_RedactsSecretKeys(t *testing.T) {
	var buf bytes.Buffer
	Setup(Config{Level: "info", Format: FormatJSON, Output: &buf, RedactKeys: []string{"stripe_key"}})

	slog.Info("login",
		"password", "hunter2",
		"Authorization", "Bearer abc",
		"X-Api-Key", "k",
		"stripe_key", "sk_live",
		slog.Group("req", "cookie", "sid=1"),
		"email", "a@example.com",
	)
	out := buf.String()
	for _, leaked := range []string{"hunter2", "Bearer abc", `"k"`, "sk_live", "sid=1"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("secret %s leaked: %q", leaked, out)
		}
	}
	if !strings.Contains(out, `"email":"a@example.com"`) {
		t.Fatalf("non-secret field was redacted: %q", out)
	}
}
