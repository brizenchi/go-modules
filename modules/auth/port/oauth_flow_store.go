package port

import (
	"context"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

// OAuthFlowStore binds an OAuth state to the browser that initiated it.
//
// Consume must atomically delete exactly one unexpired record whose provider,
// state hash, and browser-binding hash all match. A missing, expired, already
// consumed, or mismatched flow returns domain.ErrInvalidState.
type OAuthFlowStore interface {
	SaveOAuthFlow(ctx context.Context, flow domain.OAuthFlow) error
	ConsumeOAuthFlow(ctx context.Context, provider domain.Provider, stateHash, bindingHash string) (*domain.OAuthFlow, error)
}
