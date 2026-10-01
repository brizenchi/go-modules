package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fslog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/gin-gonic/gin"
)

func decodeLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		out = append(out, rec)
	}
	return out
}

// TestRouterPanicIsCorrelatedAcrossLogsAndTrace guards the middleware order:
// a panicking handler must yield a 500, a panic record and an ERROR access
// record that share one request_id and trace_id.
func TestRouterPanicIsCorrelatedAcrossLogsAndTrace(t *testing.T) {
	var buf bytes.Buffer
	fslog.Setup(fslog.Config{Format: fslog.FormatJSON, Output: &buf})
	shutdown, err := tracing.Setup(tracing.Config{ServiceName: "observability-test", SampleRate: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer tracing.Shutdown(context.Background(), shutdown)

	engine := BuildRouter(RouterConfig{ServiceName: "observability-test", AllowedOrigins: []string{"https://app.example.com"}}, nil)
	engine.GET("/boom", func(*gin.Context) { panic("boom") })

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set("X-Request-ID", "rid-router")
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", res.Code)
	}
	var panicRec, accessRec map[string]any
	for _, rec := range decodeLogLines(t, buf.String()) {
		switch rec["msg"] {
		case "panic recovered":
			panicRec = rec
		case "http request":
			accessRec = rec
		}
	}
	if panicRec == nil || accessRec == nil {
		t.Fatalf("missing panic or access record: %s", buf.String())
	}
	if accessRec["level"] != "ERROR" || accessRec["status_code"] != float64(500) {
		t.Fatalf("access record = %v", accessRec)
	}
	for _, key := range []string{"request_id", "trace_id"} {
		if panicRec[key] == nil || panicRec[key] != accessRec[key] {
			t.Fatalf("%s not shared: panic=%v access=%v", key, panicRec[key], accessRec[key])
		}
	}
	if accessRec["request_id"] != "rid-router" {
		t.Fatalf("request_id = %v", accessRec["request_id"])
	}
}

func TestRouterPreflightAndHealthAreNotLogged(t *testing.T) {
	var buf bytes.Buffer
	fslog.Setup(fslog.Config{Format: fslog.FormatJSON, Output: &buf})

	engine := BuildRouter(RouterConfig{ServiceName: "observability-test", AllowedOrigins: []string{"https://app.example.com"}}, nil)

	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/notes", nil)
	preflight.Header.Set("Origin", "https://app.example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	engine.ServeHTTP(httptest.NewRecorder(), preflight)
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	if strings.Contains(buf.String(), `"msg":"http request"`) {
		t.Fatalf("preflight or health produced access logs: %s", buf.String())
	}
}
