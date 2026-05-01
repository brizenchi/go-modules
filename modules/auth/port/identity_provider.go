// Package port defines the interfaces the auth module depends on.
package port

import (
	"context"
	"net/url"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

// IdentityProvider abstracts an OAuth-style identity provider
// (Google, GitHub, Anthropic, ...).
//
// Implementations must be safe for concurrent use.
type IdentityProvider interface {
	// Name returns the provider identifier; matches domain.Provider.
	Name() domain.Provider

	// AuthorizeURL builds the URL the user is redirected to to begin login.
	// `state` MUST round-trip through the provider unchanged so callbacks
	// can be matched to the original request.
	AuthorizeURL(state string, extra url.Values) (string, error)

	// Exchange completes the callback by trading the authorization code
	// for an access token + normalized profile.
	Exchange(ctx context.Context, callbackQuery url.Values) (*domain.OAuthProfile, error)

	// VerifyState validates that a state value returned in a callback
	// is one this server issued and hasn't expired. Provider implementations
	// can use a stateless signed token (recommended) or a state store.
	VerifyState(state string) error

	// IssueState produces a fresh state value to embed in AuthorizeURL.
	IssueState() (string, error)
}
