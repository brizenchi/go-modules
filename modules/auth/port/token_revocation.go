package port

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

type ContextTokenVerifier interface {
	ParseContext(ctx context.Context, value string) (*domain.Identity, error)
}

type TokenRevoker interface {
	Revoke(ctx context.Context, value string) error
}

type TokenRevocationStore interface {
	IsRevoked(ctx context.Context, tokenHash string) (bool, error)
	RevokeToken(ctx context.Context, tokenHash string, expiresAt time.Time) error
}
