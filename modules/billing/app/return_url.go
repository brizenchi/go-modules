package app

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

// OriginReturnURLValidator allows absolute http(s) URLs only when their exact
// scheme, hostname and explicit/default port match a host-configured origin.
type OriginReturnURLValidator struct {
	origins map[string]struct{}
}

func NewOriginReturnURLValidator(configuredURLs ...string) (*OriginReturnURLValidator, error) {
	validator := &OriginReturnURLValidator{origins: make(map[string]struct{}, len(configuredURLs))}
	for _, configured := range configuredURLs {
		origin, err := normalizedHTTPOrigin(configured)
		if err != nil {
			return nil, fmt.Errorf("billing: invalid allowed return origin: %w", err)
		}
		validator.origins[origin] = struct{}{}
	}
	if len(validator.origins) == 0 {
		return nil, fmt.Errorf("billing: at least one return URL origin is required")
	}
	return validator, nil
}

func (v *OriginReturnURLValidator) ValidateReturnURL(rawURL string) error {
	origin, err := normalizedHTTPOrigin(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidReturnURL, err)
	}
	if v == nil {
		return domain.ErrInvalidReturnURL
	}
	if _, ok := v.origins[origin]; !ok {
		return domain.ErrInvalidReturnURL
	}
	return nil
}

func normalizedHTTPOrigin(rawURL string) (string, error) {
	if rawURL == "" || rawURL != strings.TrimSpace(rawURL) || strings.HasPrefix(rawURL, "//") {
		return "", fmt.Errorf("absolute URL required")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed == nil || parsed.IsAbs() == false || parsed.Host == "" {
		return "", fmt.Errorf("absolute URL required")
	}
	if parsed.User != nil || parsed.Opaque != "" {
		return "", fmt.Errorf("userinfo and opaque URLs are forbidden")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("http(s) URL required")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", fmt.Errorf("hostname required")
	}
	port := parsed.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return scheme + "://" + net.JoinHostPort(hostname, port), nil
}

func validateReturnURL(validator interface{ ValidateReturnURL(string) error }, rawURL string) error {
	if validator != nil {
		return validator.ValidateReturnURL(rawURL)
	}
	// Backward-compatible fallback still rejects malformed and dangerous URL
	// forms. Production hosts should inject OriginReturnURLValidator.
	if _, err := normalizedHTTPOrigin(rawURL); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidReturnURL, err)
	}
	return nil
}
