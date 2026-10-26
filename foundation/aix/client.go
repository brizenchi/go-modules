package aix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client sends stateless completion requests through an injected HTTP client.
type Client struct {
	cfg  Config
	http *http.Client
}

// New requires the host's HTTP client so its tracing and transport are retained.
// The copied client refuses redirects; this adapter adds no retries.
func New(cfg Config, client *http.Client) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("%w: HTTP client required", ErrInvalidConfig)
	}
	copyClient := *client
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{cfg: cfg.normalized(), http: &copyClient}, nil
}

// Complete makes one stateless, non-streaming model request. The caller owns
// authorization, prompt construction, persistence, metering and retry policy.
func (c *Client) Complete(ctx context.Context, messages []Message) (Completion, error) {
	if err := validateMessages(messages); err != nil {
		return Completion{}, err
	}
	body, err := json.Marshal(struct {
		Model           string    `json:"model"`
		Messages        []Message `json:"messages"`
		MaxTokens       int       `json:"max_tokens"`
		ReasoningEffort string    `json:"reasoning_effort,omitempty"`
		Stream          bool      `json:"stream"`
	}{c.cfg.Model, messages, c.cfg.MaxOutputTokens, c.cfg.ReasoningEffort, false})
	if err != nil {
		return Completion{}, fmt.Errorf("encode model input: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Completion{}, fmt.Errorf("%w: cannot construct request", ErrInvalidConfig)
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Completion{}, fmt.Errorf("aix: complete: %w", ctx.Err())
		}
		return Completion{}, fmt.Errorf("%w: transport failed", ErrProvider)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Completion{}, fmt.Errorf("%w: HTTP %d", ErrProvider, response.StatusCode)
	}
	const maxResponse = 2 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil || len(data) > maxResponse {
		if ctx.Err() != nil {
			return Completion{}, fmt.Errorf("aix: complete: %w", ctx.Err())
		}
		return Completion{}, fmt.Errorf("%w: invalid response size", ErrProvider)
	}
	var result struct {
		Model   string `json:"model"`
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return Completion{}, fmt.Errorf("%w: invalid completion", ErrProvider)
	}
	if result.Model == "" {
		result.Model = c.cfg.Model
	}
	return Completion{Content: result.Choices[0].Message.Content, Model: result.Model, FinishReason: result.Choices[0].FinishReason}, nil
}

var _ Generator = (*Client)(nil)
