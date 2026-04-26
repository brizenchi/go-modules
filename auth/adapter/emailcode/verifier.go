package emailcode

import (
	"context"
	"strings"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/port"
)

// Verifier implements port.EmailCodeVerifier.
type Verifier struct {
	cfg   Config
	store port.CodeRateLimitStore
}

func NewVerifier(cfg Config, store port.CodeRateLimitStore) *Verifier {
	return &Verifier{cfg: cfg.withDefaults(), store: store}
}

// Verify checks the supplied code. On success, the code is invalidated.
// Returns:
//   - ErrInvalidEmail when email is empty
//   - ErrInvalidCode when no live code exists or the supplied code mismatches
//   - ErrCodeMaxAttempts after too many failed verifications (code is
//     also invalidated to force the caller to request a fresh one).
func (v *Verifier) Verify(ctx context.Context, email, code string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	code = strings.TrimSpace(code)
	if email == "" || code == "" {
		return domain.ErrInvalidCode
	}

	stored, _, err := v.store.LoadCode(ctx, email)
	if err != nil {
		return err
	}
	if stored == "" {
		return domain.ErrInvalidCode
	}
	if stored != code {
		attempts, err := v.store.IncrAttempts(ctx, email)
		if err != nil {
			return err
		}
		if attempts >= v.cfg.MaxAttempts {
			_ = v.store.DeleteCode(ctx, email)
			return domain.ErrCodeMaxAttempts
		}
		return domain.ErrInvalidCode
	}
	if err := v.store.DeleteCode(ctx, email); err != nil {
		return err
	}
	return nil
}

var _ port.EmailCodeVerifier = (*Verifier)(nil)
