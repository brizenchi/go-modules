package ginx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	flog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestCORS_PreflightShortCircuit(t *testing.T) {
	r := newRouter()
	r.Use(CORS(CORSConfig{AllowedOrigins: []string{"https://app"}}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://app")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app" {
		t.Errorf("ACAO = %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_OriginAllowlist(t *testing.T) {
	r := newRouter()
	r.Use(CORS(CORSConfig{AllowedOrigins: []string{"https://allowed"}}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://blocked")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Request still processes (CORS is enforced by the browser, not server),
	// but ACAO should NOT echo the disallowed origin.
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("ACAO leaked for disallowed origin: %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRequestID_GeneratedAndEchoed(t *testing.T) {
	r := newRouter()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if rid := w.Header().Get(HeaderRequestID); len(rid) < 30 {
		t.Errorf("expected uuid request id, got %q", rid)
	}
}

func TestRequestID_HonorsIncoming(t *testing.T) {
	r := newRouter()
	var captured string
	var helper string
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		captured = c.GetString(string(RequestIDKey))
		helper = RequestIDFromContext(c)
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(HeaderRequestID, "rid-explicit")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if captured != "rid-explicit" {
		t.Errorf("captured = %q", captured)
	}
	if helper != "rid-explicit" {
		t.Errorf("helper = %q", helper)
	}
	if w.Header().Get(HeaderRequestID) != "rid-explicit" {
		t.Errorf("response header = %q", w.Header().Get(HeaderRequestID))
	}
}

func TestRecover_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := newRouter()
	r.Use(Recover())
	r.GET("/boom", func(c *gin.Context) { panic("test panic") })

	req := httptest.NewRequest("GET", "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("missing panic log: %q", buf.String())
	}
	if !strings.Contains(w.Body.String(), `"code":500`) {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestSecure_HeadersSet(t *testing.T) {
	r := newRouter()
	r.Use(Secure(SecureConfig{}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("X-Frame-Options = %q", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", w.Header().Get("X-Content-Type-Options"))
	}
}

func TestAccessLog_Logs(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := newRouter()
	r.Use(RequestID())
	r.Use(AccessLog(AccessLogConfig{}))
	r.GET("/users/:id", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/users/user-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	out := buf.String()
	if !strings.Contains(out, `"path":"/users/user-123"`) {
		t.Errorf("missing path in log: %q", out)
	}
	for _, field := range []string{
		`"component":"http"`,
		`"operation":"request"`,
		`"outcome":"success"`,
		`"route":"/users/:id"`,
		`"status_code":200`,
		`"duration_ms":`,
	} {
		if !strings.Contains(out, field) {
			t.Errorf("missing %s in log: %q", field, out)
		}
	}
	if !strings.Contains(out, `"request_id":`) {
		t.Errorf("missing request_id in log: %q", out)
	}
}

func TestAccessLog_SkipPaths(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := newRouter()
	r.Use(AccessLog(AccessLogConfig{SkipPaths: []string{"/health"}}))
	r.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if strings.Contains(buf.String(), `"path":"/health"`) {
		t.Errorf("/health log should be skipped, got %q", buf.String())
	}
}

func TestRequestID_RejectsUnsafeIncoming(t *testing.T) {
	for _, bad := range []string{
		"rid\nforged=1",
		"rid with spaces",
		strings.Repeat("a", MaxRequestIDLength+1),
	} {
		r := newRouter()
		r.Use(RequestID())
		r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

		req := httptest.NewRequest("GET", "/x", nil)
		req.Header[HeaderRequestID] = []string{bad}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if got := w.Header().Get(HeaderRequestID); got == bad || len(got) < 30 {
			t.Errorf("unsafe id %q was not replaced, got %q", bad, got)
		}
	}
}

func TestRequestID_ReachesBusinessLogsThroughContext(t *testing.T) {
	var buf bytes.Buffer
	flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})

	r := newRouter()
	r.Use(RequestID())
	r.GET("/x", func(c *gin.Context) {
		slog.InfoContext(c.Request.Context(), "business event")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(HeaderRequestID, "rid-biz")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), `"request_id":"rid-biz"`) {
		t.Fatalf("business log missing request_id: %q", buf.String())
	}
}

func TestAccessLog_RecordsRecoveredPanicAs500(t *testing.T) {
	var buf bytes.Buffer
	flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})

	r := newRouter()
	r.Use(RequestID(), AccessLog(AccessLogConfig{}), Recover())
	r.GET("/boom", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/boom", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	var access string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if strings.Contains(line, `"msg":"http request"`) {
			access = line
		}
	}
	for _, want := range []string{`"level":"ERROR"`, `"status_code":500`, `"outcome":"failure"`, `"errors":`} {
		if !strings.Contains(access, want) {
			t.Fatalf("access log missing %s: %q", want, access)
		}
	}
	if n := strings.Count(access, `"request_id"`); n != 1 {
		t.Fatalf("request_id appears %d times: %q", n, access)
	}
}

func TestAccessLog_LogsAndRepanicsWithoutRecover(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := newRouter()
	r.Use(AccessLog(AccessLogConfig{}))
	r.GET("/boom", func(c *gin.Context) { panic("boom") })

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic to propagate")
			}
		}()
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/boom", nil))
	}()
	if !strings.Contains(buf.String(), `"status_code":500`) {
		t.Fatalf("panic not logged as 500: %q", buf.String())
	}
}

func TestAccessLog_OmitsQueryString(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	r := newRouter()
	r.Use(AccessLog(AccessLogConfig{}))
	r.GET("/cb", func(c *gin.Context) { c.String(200, "ok") })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/cb?code=secret-code", nil))

	if strings.Contains(buf.String(), "secret-code") {
		t.Fatalf("query string leaked into access log: %q", buf.String())
	}
}

func TestRecover_MarksSpanAsError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	r := newRouter()
	r.Use(func(c *gin.Context) {
		ctx, span := tp.Tracer("test").Start(c.Request.Context(), "server")
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}, Recover())
	r.GET("/boom", func(c *gin.Context) { panic(errors.New("boom")) })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/boom", nil))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("spans = %d", len(spans))
	}
	if spans[0].Status().Code != codes.Error {
		t.Fatalf("span status = %v", spans[0].Status())
	}
	if len(spans[0].Events()) == 0 || spans[0].Events()[0].Name != "exception" {
		t.Fatalf("panic not recorded as exception event: %+v", spans[0].Events())
	}
}

func TestRecover_ReraisesAbortHandler(t *testing.T) {
	r := newRouter()
	r.Use(Recover())
	r.GET("/abort", func(c *gin.Context) { panic(http.ErrAbortHandler) })

	defer func() {
		if recover() != http.ErrAbortHandler { //nolint:errorlint
			t.Fatal("expected http.ErrAbortHandler to be re-raised")
		}
	}()
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/abort", nil))
}
