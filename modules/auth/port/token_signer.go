package port

import (
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

// TokenSigner issues and parses session tokens for an Identity.
//
// The reference adapter is HMAC-SHA256 JWT, but other implementations
// (RS256, opaque tokens with a backing store) can be plugged in.
type TokenSigner interface {
	// Issue mints a token that encodes the Identity for the given TTL.
	Issue(identity domain.Identity, ttl time.Duration) (*domain.Token, error)

	// Parse validates a token and returns the embedded Identity.
	Parse(value string) (*domain.Identity, error)
}

// WSTicketSigner issues short-lived single-use tickets that can be
// exchanged for a WebSocket connection (e.g. terminal access).
//
// The Scope map is opaque to auth — callers put resource ids in it and
// downstream handlers pull them back out.
type WSTicketSigner interface {
	Issue(userID string, scope map[string]string, ttl time.Duration) (*domain.WSTicket, error)
	Parse(value string) (*domain.WSTicket, error)
}
