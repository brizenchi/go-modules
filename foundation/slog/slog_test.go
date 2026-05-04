package slog

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	tracingpkg "github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
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

	shutdown, err := tracingpkg.Setup(tracingpkg.Config{ServiceName: "svc", SampleRate: 1})
	if err != nil {
		t.Fatalf("tracing.Setup: %v", err)
	}
	defer tracingpkg.Shutdown(context.Background(), shutdown)

	ctx, span := otel.GetTracerProvider().Tracer("svc").Start(context.Background(), "test", oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
	defer span.End()

	slog.InfoContext(ctx, "hello")
	out := buf.String()
	if !strings.Contains(out, `"trace_id":"`) {
		t.Fatalf("missing trace_id in %q", out)
	}
	if !strings.Contains(out, `"span_id":"`) {
		t.Fatalf("missing span_id in %q", out)
	}
}
