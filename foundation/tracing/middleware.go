package tracing

import (
	"context"
	"net/http"

	flog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// TraceIDKey is the gin-context key for the active trace ID.
	//
	// Deprecated: the middleware no longer copies ids into the gin
	// context. Use TraceID(c.Request.Context()).
	TraceIDKey = "trace_id"
	// SpanIDKey is the gin-context key for the active span ID.
	//
	// Deprecated: use SpanID(c.Request.Context()).
	SpanIDKey = "span_id"

	// RequestIDAttribute is the span attribute that carries the
	// X-Request-ID assigned by foundation/ginx.RequestID.
	RequestIDAttribute = attribute.Key("http.request_id")
)

// MiddlewareConfig configures Middleware.
type MiddlewareConfig struct {
	// ServiceName is the logical server name recorded on spans and metrics.
	ServiceName string
	// SkipPaths are exact-match paths that are neither traced nor
	// measured (e.g. "/health").
	SkipPaths []string
}

// Trace returns Middleware with only the service name set.
func Trace(serviceName string) gin.HandlerFunc {
	return Middleware(MiddlewareConfig{ServiceName: serviceName})
}

// Middleware returns a Gin middleware built on the official otelgin
// instrumentation. For every request it:
//
//  1. Extracts W3C Trace Context and Baggage from the inbound headers.
//  2. Starts a server span named "METHOD /route/:template" with
//     semantic-convention attributes (http.request.method, url.path,
//     http.route, http.response.status_code, client.address, ...).
//  3. Marks 5xx and gin errors (c.Error) as span errors.
//  4. Records http.server.request.duration and request/response body
//     size metrics through the global MeterProvider.
//
// The request id from foundation/ginx.RequestID is copied onto the
// server span as http.request_id by the span processor installed in
// Setup. Place it after ginx.RequestID and before ginx.AccessLog and
// ginx.Recover.
func Middleware(cfg MiddlewareConfig) gin.HandlerFunc {
	var opts []otelgin.Option
	if len(cfg.SkipPaths) > 0 {
		skip := make(map[string]struct{}, len(cfg.SkipPaths))
		for _, p := range cfg.SkipPaths {
			skip[p] = struct{}{}
		}
		opts = append(opts, otelgin.WithFilter(func(r *http.Request) bool {
			_, skipped := skip[r.URL.Path]
			return !skipped
		}))
	}
	return otelgin.Middleware(cfg.ServiceName, opts...)
}

// requestIDProcessor copies the request id stored by ginx.RequestID onto
// server spans, so a request id reported by a user leads straight to its
// trace.
type requestIDProcessor struct{}

func (requestIDProcessor) OnStart(parent context.Context, s tracesdk.ReadWriteSpan) {
	if s.SpanKind() != oteltrace.SpanKindServer {
		return
	}
	if rid := flog.RequestIDFromContext(parent); rid != "" {
		s.SetAttributes(RequestIDAttribute.String(rid))
	}
}

func (requestIDProcessor) OnEnd(tracesdk.ReadOnlySpan)      {}
func (requestIDProcessor) Shutdown(context.Context) error   { return nil }
func (requestIDProcessor) ForceFlush(context.Context) error { return nil }
