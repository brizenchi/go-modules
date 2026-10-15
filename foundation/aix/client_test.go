package aix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testConfig() Config { return Config{APIKey: "sk_test_not-a-real-key", Model: "example-model"} }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestCompletePassesExplicitInputAndContextWithoutKeepingConversationState(t *testing.T) {
	type contextKey struct{}
	ctx, cancel := context.WithDeadline(context.WithValue(context.Background(), contextKey{}, "trace-example"), time.Now().Add(time.Minute))
	defer cancel()
	calls := 0
	client, err := New(testConfig(), &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPost || req.URL.String() != "https://api.openai.com/v1/chat/completions" || req.Header.Get("Authorization") != "Bearer sk_test_not-a-real-key" {
			t.Fatal("incorrect provider request")
		}
		if req.Context().Value(contextKey{}) != "trace-example" {
			t.Fatal("caller context lost")
		}
		if deadline, ok := req.Context().Deadline(); !ok || time.Until(deadline) > 45*time.Second {
			t.Fatal("missing request deadline")
		}
		var payload struct {
			Model     string
			Messages  []Message
			MaxTokens int `json:"max_tokens"`
			Stream    bool
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != "example-model" || payload.MaxTokens != 1024 || payload.Stream || len(payload.Messages) != 3 || payload.Messages[0].Role != "system" || payload.Messages[2].Content != "Another input" {
			t.Fatal("input was changed or product-specific alternation was imposed")
		}
		return response(200, `{"choices":[{"message":{"role":"assistant","content":"A summary"}}]}`), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		result, err := client.Complete(ctx, []Message{{Role: "system", Content: "Summarize"}, {Role: "user", Content: "First input"}, {Role: "user", Content: "Another input"}})
		if err != nil || result.Content != "A summary" || result.Model != "example-model" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want one per explicit operation", calls)
	}
}

func TestInvalidConfigurationAndMessagesNeverCallProvider(t *testing.T) {
	for _, base := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com?key=value", "https://example.com?", "https://example.com#fragment", "://invalid"} {
		cfg := testConfig()
		cfg.BaseURL = base
		if _, err := New(cfg, &http.Client{}); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("base %q: %v", base, err)
		}
	}
	for _, configure := range []func(*Config){
		func(c *Config) { c.APIKey = "" }, func(c *Config) { c.APIKey = "key\r\nheader" },
		func(c *Config) { c.Model = "" }, func(c *Config) { c.Model = strings.Repeat("m", 201) },
		func(c *Config) { c.MaxOutputTokens = -1 }, func(c *Config) { c.MaxOutputTokens = 4097 },
	} {
		cfg := testConfig()
		configure(&cfg)
		if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("error=%v", err)
		}
	}
	if _, err := New(testConfig(), nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	client, err := New(testConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected provider call")
		return nil, errors.New("unreachable")
	})})
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([]Message, 129)
	tooLarge := []Message{{Role: "user", Content: strings.Repeat("x", 64001)}}
	totalTooLarge := make([]Message, 5)
	for i := range totalTooLarge {
		totalTooLarge[i] = Message{Role: "user", Content: strings.Repeat("x", 64000)}
	}
	for _, messages := range [][]Message{nil, {{Role: "tool", Content: "x"}}, {{Role: "user", Content: " "}}, {{Role: "user", Content: string([]byte{0xff})}}, tooMany, tooLarge, totalTooLarge} {
		if _, err := client.Complete(context.Background(), messages); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestProviderFailuresAreBoundedSanitizedAndNotRetried(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"upstream error", 429, "sk_test_not-a-real-key private prompt"},
		{"malformed", 200, "not json"}, {"empty choices", 200, `{"choices":[]}`},
		{"empty content", 200, `{"choices":[{"message":{"content":" "}}]}`},
		{"oversize", 200, strings.Repeat("x", 2*1024*1024+1)},
		{"redirect", 302, "private prompt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				result := response(tc.status, tc.body)
				result.Header.Set("Location", "https://other.example.com")
				return result, nil
			})}
			client, err := New(testConfig(), httpClient)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Complete(context.Background(), []Message{{Role: "user", Content: "private prompt"}})
			if !errors.Is(err, ErrProvider) || calls != 1 || strings.Contains(err.Error(), "private prompt") || strings.Contains(err.Error(), "sk_test_") || httpClient.CheckRedirect != nil {
				t.Fatalf("calls=%d, err=%v", calls, err)
			}
		})
	}
}

func TestCancellationAndTransportFailurePreserveOnlySafeErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := New(testConfig(), &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(ctx, []Message{{Role: "user", Content: "input"}}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	client, err = New(testConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport detail") })})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(context.Background(), []Message{{Role: "developer", Content: "input"}}); !errors.Is(err, ErrProvider) || strings.Contains(err.Error(), "private transport") {
		t.Fatal(err)
	}
}

type readFunc func([]byte) (int, error)

func (f readFunc) Read(p []byte) (int, error) { return f(p) }

func TestCancellationWhileReadingCompletionRemainsDetectable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := New(testConfig(), &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(readFunc(func([]byte) (int, error) {
			cancel()
			return 0, context.Canceled
		}))}, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(ctx, []Message{{Role: "user", Content: "input"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want cancellation", err)
	}
}
