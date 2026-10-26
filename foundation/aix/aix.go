// Package aix provides stateless, server-side text generation over a compatible
// Chat Completions API. It owns no users, conversations, persistence or billing.
package aix

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// ErrInvalidConfig identifies missing or unsupported provider configuration.
	ErrInvalidConfig = errors.New("aix: invalid configuration")
	// ErrInvalidRequest identifies invalid or oversized explicit model input.
	ErrInvalidRequest = errors.New("aix: invalid request")
	// ErrProvider identifies a failed or invalid upstream response, without its body.
	ErrProvider = errors.New("aix: provider unavailable")
)

// Config is deployment-owned. APIKey must never be exposed to the browser.
type Config struct {
	// BaseURL is an HTTPS API base, defaulting to https://api.openai.com/v1.
	BaseURL string
	// APIKey is the server-side provider credential.
	APIKey string `json:"-"`
	// Model is the provider's compatible text model identifier.
	Model string
	// MaxOutputTokens defaults to 1024 and must not exceed 65536. Some providers
	// count reasoning tokens toward this limit.
	MaxOutputTokens int
	// ReasoningEffort is sent as reasoning_effort when set: none, minimal, low,
	// medium or high. Empty leaves the provider's default.
	ReasoningEffort string
	// Timeout bounds each request, defaulting to 45 seconds and at most 10 minutes.
	Timeout time.Duration
}

const (
	defaultTimeout  = 45 * time.Second
	maxTimeout      = 10 * time.Minute
	maxOutputTokens = 65536
)

var reasoningEfforts = map[string]bool{"": true, "none": true, "minimal": true, "low": true, "medium": true, "high": true}

func (c Config) normalized() Config {
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com/v1"
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
	if c.MaxOutputTokens == 0 {
		c.MaxOutputTokens = 1024
	}
	c.ReasoningEffort = strings.TrimSpace(c.ReasoningEffort)
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	return c
}

// Validate checks transport configuration without making any outbound call.
func (c Config) Validate() error {
	c = c.normalized()
	if c.APIKey == "" || strings.ContainsAny(c.APIKey, "\r\n\x00") || c.Model == "" || len(c.Model) > 200 || c.MaxOutputTokens < 1 || c.MaxOutputTokens > maxOutputTokens {
		return fmt.Errorf("%w: key, model or output limit", ErrInvalidConfig)
	}
	if !reasoningEfforts[c.ReasoningEffort] || c.Timeout < time.Second || c.Timeout > maxTimeout {
		return fmt.Errorf("%w: reasoning effort or timeout", ErrInvalidConfig)
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return fmt.Errorf("%w: HTTPS API base required without credentials, query or fragment", ErrInvalidConfig)
	}
	return nil
}

// Message is explicit model input, not a persisted customer conversation.
type Message struct {
	// Role is system, developer, user or assistant.
	Role string `json:"role"`
	// Content is non-empty UTF-8 text explicitly supplied by the caller.
	Content string `json:"content"`
}

// Completion contains the text result of one model request.
type Completion struct {
	// Content is the returned text; no persistence is implied.
	Content string
	// Model is the returned model identifier, falling back to configuration.
	Model string
	// FinishReason is the provider's stop reason; "length" means the output was cut off.
	FinishReason string
}

// Generator is injected into the host's own business service and can be faked.
type Generator interface {
	// Complete generates text using the caller's context and explicit input.
	Complete(context.Context, []Message) (Completion, error)
}

func validateMessages(messages []Message) error {
	if len(messages) == 0 || len(messages) > 128 {
		return ErrInvalidRequest
	}
	total := 0
	for _, message := range messages {
		switch message.Role {
		case "system", "developer", "user", "assistant":
		default:
			return ErrInvalidRequest
		}
		if !utf8.ValidString(message.Content) || strings.TrimSpace(message.Content) == "" || len(message.Content) > 64000 {
			return ErrInvalidRequest
		}
		total += len(message.Content)
	}
	if total > 256000 {
		return ErrInvalidRequest
	}
	return nil
}
