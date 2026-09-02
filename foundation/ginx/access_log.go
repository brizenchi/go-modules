package ginx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLogConfig controls the access-log middleware.
type AccessLogConfig struct {
	// SkipPaths are exact-match paths excluded from logging (e.g. "/health").
	SkipPaths []string
}

// AccessLog returns a middleware that logs one structured slog record per
// request using the stable cross-project HTTP field schema.
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
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()
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
		}
		if rid := c.GetString(string(RequestIDKey)); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		if tid := c.GetString("trace_id"); tid != "" {
			attrs = append(attrs, "trace_id", tid, "span_id", c.GetString("span_id"))
		}
		ctx := c.Request.Context()
		switch {
		case status >= 500:
			slog.ErrorContext(ctx, "http request", attrs...)
		case status >= 400:
			slog.WarnContext(ctx, "http request", attrs...)
		default:
			slog.InfoContext(ctx, "http request", attrs...)
		}
	}
}
