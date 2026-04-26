// Package slog is a thin setup helper around log/slog (stdlib).
//
// It does NOT wrap or replace slog — it just standardizes the
// boot-time configuration so every service in the org outputs
// logs in the same format with the same fields.
//
// Usage:
//
//	func main() {
//	    flog.Setup(flog.Config{
//	        Level:  "info",
//	        Format: "json",
//	    })
//	    slog.Info("starting", "port", 8080)
//	}
//
// All other code (including business modules) should keep calling
// log/slog directly — never import this package's helpers in library
// code, only at app boot.
package slog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Format selects the slog handler.
type Format string

const (
	FormatText Format = "text" // human-readable
	FormatJSON Format = "json" // structured for log shippers
)

// Config drives Setup.
type Config struct {
	// Level: "debug" | "info" | "warn" | "error". Default "info".
	Level string
	// Format: "json" (default) or "text".
	Format Format
	// AddSource attaches file:line to every record. Default false.
	AddSource bool
	// Output is where logs go. Defaults to os.Stdout.
	Output io.Writer
	// Default attributes attached to every record (e.g. service name, env).
	Defaults map[string]any
}

// Setup builds an slog.Logger from cfg, attaches default attributes,
// and installs it as the process default via slog.SetDefault.
//
// Returns the same logger so the caller can hold a reference if they
// want to pin it (e.g. inject into a struct).
func Setup(cfg Config) *slog.Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stdout
	}
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}
	var handler slog.Handler
	switch normalizeFormat(cfg.Format) {
	case FormatText:
		handler = slog.NewTextHandler(out, opts)
	default:
		handler = slog.NewJSONHandler(out, opts)
	}

	if len(cfg.Defaults) > 0 {
		attrs := make([]slog.Attr, 0, len(cfg.Defaults))
		for k, v := range cfg.Defaults {
			attrs = append(attrs, slog.Any(k, v))
		}
		handler = handler.WithAttrs(attrs)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// RequestIDKey is the context key under which a request id is stored
// (and pulled back out by With). Compatible with foundation/ginx.
const RequestIDKey ctxKey = "request_id"

type ctxKey string

// With returns a logger with request-scoped attributes (request_id, etc).
//
// Pass a Gin context — RequestIDKey is read from c.Request.Context().
// Returns slog.Default() when no fields are present, so it's always
// safe to call.
func With(c *gin.Context) *slog.Logger {
	if c == nil {
		return slog.Default()
	}
	if rid := requestID(c); rid != "" {
		return slog.Default().With("request_id", rid)
	}
	return slog.Default()
}

func requestID(c *gin.Context) string {
	if v := c.GetString("request_id"); v != "" {
		return v
	}
	if v := c.GetHeader("X-Request-ID"); v != "" {
		return v
	}
	if v, ok := c.Request.Context().Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func normalizeFormat(f Format) Format {
	switch Format(strings.ToLower(string(f))) {
	case FormatText:
		return FormatText
	default:
		return FormatJSON
	}
}

// avoid unused imports if the gin helpers are stripped.
var _ context.Context
