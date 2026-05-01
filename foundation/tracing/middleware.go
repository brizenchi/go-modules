package tracing

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// TraceIDKey is the gin-context key for the active trace ID.
	TraceIDKey = "trace_id"
	// SpanIDKey is the gin-context key for the active span ID.
	SpanIDKey = "span_id"
)

// Trace returns a Gin middleware that:
//
//  1. Extracts W3C Trace Context (traceparent) from the inbound request.
//  2. Starts a server span using the global tracer provider.
//  3. Stores trace_id / span_id in the gin context so downstream
//     middleware and handlers can attach them to logs and responses.
//  4. Tags the span with the X-Request-ID when available.
//
// Place it AFTER ginx.RequestID() so the request ID is available as a
// span attribute.
func Trace(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)

	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		spanName := c.Request.Method + " " + c.Request.URL.Path
		if route := c.FullPath(); route != "" {
			spanName = c.Request.Method + " " + route
		}

		ctx, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		sc := span.SpanContext()
		if sc.HasTraceID() {
			traceID := sc.TraceID().String()
			c.Set(TraceIDKey, traceID)
			c.Set("trace_id", traceID)
		}
		if sc.HasSpanID() {
			spanID := sc.SpanID().String()
			c.Set(SpanIDKey, spanID)
			c.Set("span_id", spanID)
		}
		if rid := c.GetString("request_id"); rid != "" {
			span.SetAttributes(attribute.String("http.request_id", rid))
		}

		c.Next()

		status := c.Writer.Status()
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("url.path", c.Request.URL.Path),
			attribute.Int("http.status_code", status),
		)
		if route := c.FullPath(); route != "" {
			span.SetAttributes(attribute.String("http.route", route))
		}
		if status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
		if len(c.Errors) > 0 {
			span.SetStatus(codes.Error, c.Errors.String())
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}
	}
}
