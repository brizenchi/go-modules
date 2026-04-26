package port

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
)

// EmailCodeIssuer issues a fresh verification code, persists it, and
// arranges for it to be delivered to the email address.
//
// It is responsible for rate-limiting (per-minute and per-day caps),
// code generation, and the email delivery itself (typically by calling
// pkg/email).
type EmailCodeIssuer interface {
	Issue(ctx context.Context, email string) (*domain.CodeIssueResult, error)
}

// EmailCodeVerifier consumes a code issued by EmailCodeIssuer and
// reports whether it matches.
//
// On success the code MUST be invalidated (single-use semantics).
// On failure the implementation should track attempt counts and
// invalidate the code after too many failures.
type EmailCodeVerifier interface {
	Verify(ctx context.Context, email, code string) error
}

// CodeRateLimitStore is the persistence layer used by an EmailCodeIssuer
// implementation. Hosts can swap Redis/in-memory/Memcached.
//
// Operations are scoped to (email, day_bucket) for the day cap and to
// (email) alone for the live code + per-minute throttle.
type CodeRateLimitStore interface {
	// SaveCode persists the latest code for the email, overwriting any
	// previous code, and stamps lastSentAt.
	SaveCode(ctx context.Context, email, code string, expiresAt, lastSentAt time.Time) error

	// LoadCode returns the latest code (or "" if none/expired) and the
	// last-sent timestamp.
	LoadCode(ctx context.Context, email string) (code string, lastSentAt time.Time, err error)

	// DeleteCode invalidates the live code (called on success and on max attempts).
	DeleteCode(ctx context.Context, email string) error

	// IncrAttempts increments the failed-verification attempt counter
	// and returns the new value.
	IncrAttempts(ctx context.Context, email string) (int, error)

	// IncrDailyCount increments the per-day issuance counter for an
	// email under the given dayBucket key (e.g. "2026-04-26") and
	// returns the new value.
	IncrDailyCount(ctx context.Context, email, dayBucket string) (int, error)
}
