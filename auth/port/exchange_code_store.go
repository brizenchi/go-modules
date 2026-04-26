package port

import (
	"context"

	"github.com/brizenchi/go-modules/auth/domain"
)

// ExchangeCodeStore persists single-use OAuth callback exchange codes.
//
// Implementations must enforce the ExpiresAt deadline and the single-use
// invariant: Consume returns the code if found AND not yet consumed,
// AND deletes/marks it in one atomic step.
type ExchangeCodeStore interface {
	Save(ctx context.Context, code domain.ExchangeCode) error
	Consume(ctx context.Context, code string) (*domain.ExchangeCode, error)
}
