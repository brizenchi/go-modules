package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/foundation/resilience"
)

func TestNewClient_NoMiddleware(t *testing.T) {
	c := NewClient(Config{
		Timeout: 5 * time.Second,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return newResponse(req, http.StatusOK, "hello"), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("body = %q, want hello", body)
	}
}

func TestHeadersRT_AddsDefaultHeaders(t *testing.T) {
	var captured http.Header
	c := NewClient(Config{
		Headers: map[string]string{"X-API-Key": "secret", "User-Agent": "test/1"},
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			captured = req.Header.Clone()
			return newResponse(req, http.StatusOK, ""), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if got := captured.Get("X-API-Key"); got != "secret" {
		t.Errorf("X-API-Key = %q, want secret", got)
	}
	if got := captured.Get("User-Agent"); got != "test/1" {
		t.Errorf("User-Agent = %q, want test/1", got)
	}
}

func TestHeadersRT_RequestHeadersWin(t *testing.T) {
	var captured http.Header
	c := NewClient(Config{
		Headers: map[string]string{"X-API-Key": "default"},
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			captured = req.Header.Clone()
			return newResponse(req, http.StatusOK, ""), nil
		}),
	})

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-API-Key", "override")

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if got := captured.Get("X-API-Key"); got != "override" {
		t.Errorf("X-API-Key = %q, want override", got)
	}
}

func TestRetryRT_5xxThenSuccess(t *testing.T) {
	var calls int32
	policy := resilience.Constant(5, time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			n := atomic.AddInt32(&calls, 1)
			if n < 3 {
				return newResponse(req, http.StatusInternalServerError, ""), nil
			}
			return newResponse(req, http.StatusOK, ""), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestRetryRT_ExhaustsAndReturnsLastResponse(t *testing.T) {
	var calls int32
	policy := resilience.Constant(3, time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return newResponse(req, http.StatusInternalServerError, "boom"), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestRetryRT_429IsRetryable(t *testing.T) {
	var calls int32
	policy := resilience.Constant(5, time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if atomic.AddInt32(&calls, 1) < 2 {
				return newResponse(req, http.StatusTooManyRequests, ""), nil
			}
			return newResponse(req, http.StatusOK, ""), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestRetryRT_4xxNotRetried(t *testing.T) {
	var calls int32
	policy := resilience.Constant(5, time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return newResponse(req, http.StatusBadRequest, ""), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}

func TestRetryRT_BodyReplayedOnRetry(t *testing.T) {
	var bodies []string
	policy := resilience.Constant(3, time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(req.Body)
			bodies = append(bodies, string(b))
			if len(bodies) < 2 {
				return newResponse(req, http.StatusInternalServerError, ""), nil
			}
			return newResponse(req, http.StatusOK, ""), nil
		}),
	})

	resp, err := c.Post("http://example.com", "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if len(bodies) != 2 || bodies[0] != "payload" || bodies[1] != "payload" {
		t.Errorf("bodies = %#v, want both payload", bodies)
	}
}

func TestBreakerRT_TripsAndShortCircuits(t *testing.T) {
	var calls int32
	br := resilience.NewBreaker(resilience.BreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     time.Hour,
	})
	c := NewClient(Config{
		Breaker: br,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return newResponse(req, http.StatusInternalServerError, ""), nil
		}),
	})

	for i := 0; i < 2; i++ {
		resp, err := c.Get("http://example.com")
		if err != nil {
			t.Fatalf("Get #%d: %v", i+1, err)
		}
		resp.Body.Close()
	}

	_, err := c.Get("http://example.com")
	if !errors.Is(err, resilience.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestBreakerRT_PropagatesTransportErrors(t *testing.T) {
	br := resilience.NewBreaker(resilience.BreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     time.Hour,
	})
	c := NewClient(Config{
		Breaker: br,
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed")
		}),
	})

	_, err := c.Get("http://example.com")
	if err == nil || err.Error() != "Get \"http://example.com\": dial failed" {
		t.Fatalf("err = %v", err)
	}
}

func TestRoundTripperOrder_HeadersOnEveryRetry(t *testing.T) {
	var headerSeen []string
	policy := resilience.Constant(3, time.Millisecond)
	c := NewClient(Config{
		Retry:   &policy,
		Headers: map[string]string{"X-API-Key": "k"},
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			headerSeen = append(headerSeen, req.Header.Get("X-API-Key"))
			return newResponse(req, http.StatusInternalServerError, ""), nil
		}),
	})

	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()

	if len(headerSeen) != 3 {
		t.Fatalf("attempts = %d, want 3", len(headerSeen))
	}
	for i, h := range headerSeen {
		if h != "k" {
			t.Errorf("attempt %d X-API-Key = %q, want k", i, h)
		}
	}
}

func TestNewClient_TimeoutEnforced(t *testing.T) {
	c := NewClient(Config{
		Timeout: 50 * time.Millisecond,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return newResponse(req, http.StatusOK, ""), nil
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}),
	})

	_, err := c.Get("http://example.com")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRetryRT_ContextCancelStops(t *testing.T) {
	policy := resilience.Constant(10, 50*time.Millisecond)
	c := NewClient(Config{
		Retry: &policy,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return newResponse(req, http.StatusInternalServerError, ""), nil
		}),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, err := c.Do(req)
	if err == nil {
		t.Fatal("expected ctx error")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}
