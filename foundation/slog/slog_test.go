package slog

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
