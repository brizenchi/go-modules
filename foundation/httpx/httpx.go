// Package httpx provides a configurable *http.Client builder with
// optional retry, circuit-breaker, default headers, and per-request
// timeouts.
//
// The middleware chain is implemented as composed http.RoundTripper
// values. Outermost to innermost, when fully configured:
//
//	headersRT → breakerRT → retryRT → tracing → logging → underlying transport
//
// Headers wrap the chain so they're applied even on retries. The
// breaker wraps retry so a tripped breaker short-circuits all retries.
// Tracing and logging sit below retry so every attempt gets its own
// client span and log record, and the log record carries that span's id.
//
// HTTP semantics:
//
//   - 5xx and 429 are treated as retryable failures.
//   - Request bodies are buffered into memory once and replayed on
//     each retry. Don't pair the retry RT with very large uploads.
//   - The breaker counts 5xx as failures by default; user-supplied
//     `IsFailure` on the breaker overrides this.
//   - Per-request timeout uses the client's `Timeout` field. The
//     retry RT does NOT cancel an in-flight attempt — that's the
//     caller's job via context or the client's Timeout.
package httpx

import (
	"net/http"
	"time"

	"github.com/brizenchi/go-modules/foundation/resilience"
)

// Config configures a Client built by NewClient. Zero values mean
// "skip this layer" — a Config{} produces a plain http.Client with
// the default transport.
type Config struct {
	// Timeout is the http.Client.Timeout — total time budget per call,
	// including connect, write, server processing, redirects, and read.
	// 0 = no timeout. Strongly recommend setting this for production use.
	Timeout time.Duration

	// Retry, when non-nil, wraps the transport with retry logic for
	// transient HTTP failures (5xx, 429) and network errors.
	Retry *resilience.Policy

	// Breaker, when non-nil, wraps the transport with a circuit breaker.
	Breaker *resilience.Breaker

	// Headers added to every outgoing request. Caller-supplied headers
	// on the request take precedence (these only set when absent).
	Headers map[string]string

	// Tracing instruments outgoing requests with OpenTelemetry: one
	// client span and http.client.request.duration measurement per
	// attempt, plus W3C Trace Context / Baggage header injection so
	// downstream services continue the distributed trace.
	Tracing bool

	// Logging writes one slog record per attempt (method, host, path,
	// status, duration). The query string is never logged.
	Logging bool

	// Transport overrides the inner RoundTripper. Defaults to a tuned
	// http.Transport (see DefaultTransport).
	Transport http.RoundTripper
}

// NewClient builds an http.Client from cfg.
func NewClient(cfg Config) *http.Client {
	rt := cfg.Transport
	if rt == nil {
		rt = DefaultTransport()
	}
	if cfg.Logging {
		rt = logRT{next: rt}
	}
	if cfg.Tracing {
		rt = newTracingRT(rt)
	}
	if cfg.Retry != nil {
		rt = retryRT{next: rt, policy: *cfg.Retry}
	}
	if cfg.Breaker != nil {
		rt = breakerRT{next: rt, breaker: cfg.Breaker}
	}
	if len(cfg.Headers) > 0 {
		rt = headersRT{next: rt, headers: copyHeaders(cfg.Headers)}
	}
	return &http.Client{Transport: rt, Timeout: cfg.Timeout}
}

// DefaultTransport returns an http.Transport tuned for typical
// service-to-service traffic. Callers can clone and adjust.
func DefaultTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 10
	t.IdleConnTimeout = 90 * time.Second
	t.TLSHandshakeTimeout = 10 * time.Second
	t.ExpectContinueTimeout = 1 * time.Second
	return t
}

func copyHeaders(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
