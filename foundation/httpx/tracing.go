package httpx

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// tracingRT injects W3C Trace Context headers (traceparent, tracestate)
// into every outgoing request so downstream services can continue the
// trace. When no span is active in the request context this is a no-op.
type tracingRT struct {
	next http.RoundTripper
}

func (t tracingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))
	return t.next.RoundTrip(req)
}
