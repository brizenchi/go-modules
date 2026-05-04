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
	"go.opentelemetry.io/otel/trace"
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
	handler = contextHandler{next: handler}

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

// ProjectKey is the context key under which a project id/name may be stored.
const ProjectKey ctxKey = "project"

// EnvKey is the context key under which an environment name may be stored.
const EnvKey ctxKey = "env"

// TenantIDKey is the context key under which a tenant/workspace id may be stored.
const TenantIDKey ctxKey = "tenant_id"

// UserIDKey is the context key under which an authenticated user id may be stored.
const UserIDKey ctxKey = "user_id"

type ctxKey string

type contextHandler struct {
	next slog.Handler
}

func (h contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, attr := range attrsFromContext(ctx) {
		r.AddAttrs(attr)
	}
	return h.next.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{next: h.next.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{next: h.next.WithGroup(name)}
}

// With returns a logger with request-scoped attributes (request_id,
// trace_id, span_id).
//
// Pass a Gin context — RequestIDKey is read from c.Request.Context(),
// trace/span IDs from the active OpenTelemetry span.
// Returns slog.Default() when no fields are present, so it's always
// safe to call.
func With(c *gin.Context) *slog.Logger {
	if c == nil {
		return slog.Default()
	}
	attrs := attrsFromContext(c.Request.Context())
	seen := make(map[string]struct{}, len(attrs))
	for _, attr := range attrs {
		seen[attr.Key] = struct{}{}
	}
	if rid := requestID(c); rid != "" {
		if _, ok := seen["request_id"]; !ok {
			attrs = append(attrs, slog.String("request_id", rid))
		}
	}
	if tid := traceID(c); tid != "" {
		if _, ok := seen["trace_id"]; !ok {
			attrs = append(attrs, slog.String("trace_id", tid))
		}
		if sid := spanID(c); sid != "" {
			if _, ok := seen["span_id"]; !ok {
				attrs = append(attrs, slog.String("span_id", sid))
			}
		}
	}
	if len(attrs) == 0 {
		return slog.Default()
	}
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return slog.Default().With(args...)
}

func traceID(c *gin.Context) string {
	sc := trace.SpanFromContext(c.Request.Context()).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

func spanID(c *gin.Context) string {
	sc := trace.SpanFromContext(c.Request.Context()).SpanContext()
	if sc.HasSpanID() {
		return sc.SpanID().String()
	}
	return ""
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

func attrsFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs := make([]slog.Attr, 0, 6)
	if rid, ok := ctx.Value(RequestIDKey).(string); ok && rid != "" {
		attrs = append(attrs, slog.String("request_id", rid))
	}
	if project, ok := ctx.Value(ProjectKey).(string); ok && project != "" {
		attrs = append(attrs, slog.String("project", project))
	}
	if env, ok := ctx.Value(EnvKey).(string); ok && env != "" {
		attrs = append(attrs, slog.String("env", env))
	}
	if tenantID, ok := ctx.Value(TenantIDKey).(string); ok && tenantID != "" {
		attrs = append(attrs, slog.String("tenant_id", tenantID))
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		attrs = append(attrs, slog.String("trace_id", sc.TraceID().String()))
	}
	if sc.HasSpanID() {
		attrs = append(attrs, slog.String("span_id", sc.SpanID().String()))
	}
	return attrs
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
