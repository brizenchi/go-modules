package ginx

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog logs one structured slog record per request: method, path,
// status, duration, request id (if RequestID middleware ran first).
//
// Skip noisy paths via SkipPaths (exact match).
type AccessLogConfig struct {
	SkipPaths []string
}

// AccessLog returns the middleware. Empty config = no skipping.
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
