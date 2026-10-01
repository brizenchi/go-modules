package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// newTracingRT wraps next with the official otelhttp transport. Every
// attempt becomes a client span named after the HTTP method, carrying
// semantic-convention attributes (http.request.method, server.address,
// url.full with credentials redacted, http.response.status_code), and
// W3C Trace Context / Baggage headers are injected so downstream services
// continue the trace. http.client.request.duration is recorded through
// the global MeterProvider.
func newTracingRT(next http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(next)
}

// logRT writes one structured slog record per attempt, using the same
// field schema as foundation/ginx.AccessLog with component=http_client.
// The query string is never logged because it may carry credentials.
type logRT struct {
	next http.RoundTripper
}

func (l logRT) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := l.next.RoundTrip(req)
	duration := time.Since(start)

	attrs := []any{
		"component", "http_client",
		"operation", "request",
		"method", req.Method,
		"host", req.URL.Host,
		"path", req.URL.Path,
		"duration_ms", duration.Milliseconds(),
	}
	ctx := req.Context()
	switch {
	case err != nil:
		attrs = append(attrs, "outcome", "failure", "error", err)
		slog.ErrorContext(ctx, "http client request", attrs...)
	case resp.StatusCode >= 500:
		attrs = append(attrs, "outcome", "failure", "status_code", resp.StatusCode)
		slog.ErrorContext(ctx, "http client request", attrs...)
	case resp.StatusCode >= 400:
		attrs = append(attrs, "outcome", "failure", "status_code", resp.StatusCode)
		slog.WarnContext(ctx, "http client request", attrs...)
	default:
		attrs = append(attrs, "outcome", "success", "status_code", resp.StatusCode)
		slog.InfoContext(ctx, "http client request", attrs...)
	}
	return resp, err
}
