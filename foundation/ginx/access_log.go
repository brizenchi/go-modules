package ginx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// AccessLogConfig controls the access-log middleware.
type AccessLogConfig struct {
	// SkipPaths are exact-match paths excluded from logging (e.g. "/health").
	SkipPaths []string
}

// AccessLog returns a middleware that logs one structured slog record per
// request using the stable cross-project HTTP field schema:
//
//	component, operation, outcome, method, path, route, status_code,
//	duration_ms, client_ip, user_agent, request_size, response_size,
//	request_id, trace_id, span_id
//
// The query string is never logged because it may carry tokens. 5xx is
// logged at ERROR, 4xx at WARN, everything else at INFO. Gin errors
// attached through c.Error are included as "errors".
//
// Place it after RequestID and tracing.Trace so the correlation fields are
// available, and before Recover so recovered panics are logged as 500.
// If a panic still reaches this middleware it is logged as a 500 and
// re-raised.
func AccessLog(cfg AccessLogConfig) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if _, ok := skip[path]; ok {
			c.Next()
			return
		}
		start := time.Now()
		completed := false
		defer func() {
			status := c.Writer.Status()
			if !completed {
				status = http.StatusInternalServerError
			}
			logAccess(c, path, status, time.Since(start))
		}()
		c.Next()
		completed = true
	}
}

func logAccess(c *gin.Context, path string, status int, duration time.Duration) {
	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	outcome := "success"
	if status >= 400 {
		outcome = "failure"
	}
	attrs := []any{
		"component", "http",
		"operation", "request",
		"outcome", outcome,
		"method", c.Request.Method,
		"path", path,
		"route", route,
		"status_code", status,
		"duration_ms", duration.Milliseconds(),
		"client_ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
		"request_size", max(c.Request.ContentLength, 0),
		"response_size", max(c.Writer.Size(), 0),
	}
	// foundation/slog also derives these from ctx and skips keys that
	// are already present; adding them here keeps them in the record
	// with a plain slog handler too.
	if rid := c.GetString(string(RequestIDKey)); rid != "" {
		attrs = append(attrs, "request_id", rid)
	}
	ctx := c.Request.Context()
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		attrs = append(attrs, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}
	if len(c.Errors) > 0 {
		attrs = append(attrs, "errors", c.Errors.String())
	}
	switch {
	case status >= 500:
		slog.ErrorContext(ctx, "http request", attrs...)
	case status >= 400:
		slog.WarnContext(ctx, "http request", attrs...)
	default:
		slog.InfoContext(ctx, "http request", attrs...)
	}
}
