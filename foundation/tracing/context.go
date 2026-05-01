package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceID returns the hex trace ID from ctx's active span, or "".
func TraceID(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanID returns the hex span ID from ctx's active span, or "".
func SpanID(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasSpanID() {
		return sc.SpanID().String()
	}
	return ""
}
