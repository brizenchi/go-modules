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
	"github.com/gin-gonic/gin"
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
