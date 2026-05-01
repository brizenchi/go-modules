package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/brizenchi/go-modules/foundation/resilience"
)

// errRetryableStatus is sentinel returned by the retry RT when a 5xx /
// 429 response should trigger a retry. Not exported — callers see the
// underlying http.Response on the final attempt.
var errRetryableStatus = errors.New("httpx: retryable status")

type headersRT struct {
	next    http.RoundTripper
	headers map[string]string
}

func (h headersRT) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range h.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	return h.next.RoundTrip(req)
}

type retryRT struct {
	next   http.RoundTripper
	policy resilience.Policy
}

func (r retryRT) RoundTrip(req *http.Request) (*http.Response, error) {
	// Buffer the body once so retries can replay it.
	var bodyBytes []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("httpx: read body: %w", err)
		}
		bodyBytes = b
	}

	var resp *http.Response
	policy := r.policy
	origRetryable := policy.Retryable
	policy.Retryable = func(err error) bool {
		switch {
		case errors.Is(err, errRetryableStatus):
			return true
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return false
		case origRetryable != nil:
			return origRetryable(err)
		default:
			return true
		}
	}

	err := resilience.Do(req.Context(), func(ctx context.Context) error {
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			resp = nil
		}

		// Reset body for the next attempt.
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}
		attempt := req.Clone(ctx)
		if bodyBytes != nil {
			attempt.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			attempt.ContentLength = int64(len(bodyBytes))
		}

		got, e := r.next.RoundTrip(attempt)
		if e != nil {
			return e
		}
		resp = got
		if shouldRetry(got.StatusCode) {
			return fmt.Errorf("%w (%d)", errRetryableStatus, got.StatusCode)
		}
		return nil
	}, policy)

	if resp != nil {
		if errors.Is(err, errRetryableStatus) || err == nil {
			return resp, nil
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	return nil, err
}

// shouldRetry reports whether the HTTP status code is transient and
// worth retrying. 408 (request timeout), 425 (too early), 429 (rate
// limited), and all 5xx are considered transient.
func shouldRetry(status int) bool {
	switch status {
	case http.StatusRequestTimeout,
		http.StatusTooEarly,
		http.StatusTooManyRequests:
		return true
	}
	return status >= 500 && status <= 599
}

type breakerRT struct {
	next    http.RoundTripper
	breaker *resilience.Breaker
}

func (b breakerRT) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := b.breaker.Do(req.Context(), func(ctx context.Context) error {
		got, e := b.next.RoundTrip(req)
		if e != nil {
			return e
		}
		// 5xx counts as a failure for the breaker — same intuition as
		// retry. The breaker's IsFailure config can override this.
		if got.StatusCode >= 500 && got.StatusCode <= 599 {
			resp = got
			return fmt.Errorf("httpx: server error %d", got.StatusCode)
		}
		resp = got
		return nil
	})
	// When the breaker is open, err == ErrCircuitOpen and resp is nil.
	if errors.Is(err, resilience.ErrCircuitOpen) {
		return nil, err
	}
	if err != nil && resp == nil {
		return nil, err
	}
	return resp, nil
}
