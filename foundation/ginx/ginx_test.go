package ginx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	out := buf.String()
	if !strings.Contains(out, `"path":"/x"`) {
		t.Errorf("missing path in log: %q", out)
	}
	if !strings.Contains(out, `"status":200`) {
		t.Errorf("missing status in log: %q", out)
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
