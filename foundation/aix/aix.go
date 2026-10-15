// Package aix provides stateless, server-side text generation over a compatible
// Chat Completions API. It owns no users, conversations, persistence or billing.
package aix

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
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
	// MaxOutputTokens defaults to 1024 and must not exceed 4096.
	MaxOutputTokens int
}

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
	return c
}

// Validate checks transport configuration without making any outbound call.
func (c Config) Validate() error {
	c = c.normalized()
	if c.APIKey == "" || strings.ContainsAny(c.APIKey, "\r\n\x00") || c.Model == "" || len(c.Model) > 200 || c.MaxOutputTokens < 1 || c.MaxOutputTokens > 4096 {
		return fmt.Errorf("%w: key, model or output limit", ErrInvalidConfig)
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
