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
// request: method, path, status, duration, request_id, trace_id, span_id.
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
		latency := time.Since(start)
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if rid := c.GetString(string(RequestIDKey)); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		if tid := c.GetString("trace_id"); tid != "" {
			attrs = append(attrs, "trace_id", tid, "span_id", c.GetString("span_id"))
		}
		switch {
		case c.Writer.Status() >= 500:
			slog.Error("http request", attrs...)
		case c.Writer.Status() >= 400:
			slog.Warn("http request", attrs...)
		default:
			slog.Info("http request", attrs...)
		}
	}
}
