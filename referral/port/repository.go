// Package port defines the interfaces the referral module depends on.
package port

import (
	"context"

	"github.com/brizenchi/go-modules/referral/domain"
)

// CodeRepository persists referral codes (one per user).
type CodeRepository interface {
	// FindByUser returns the code owned by a user, or domain.ErrNotFound.
	FindByUser(ctx context.Context, userID string) (*domain.Code, error)

	// FindByValue resolves a code value to its owner. Returns
	// domain.ErrNotFound if the code does not exist.
	FindByValue(ctx context.Context, value string) (*domain.Code, error)

	// Create stores a new code. Returns domain.ErrCodeCollision when
	// the value already exists for a different user (callers should
	// retry with a freshly generated value).
	Create(ctx context.Context, code domain.Code) error
}

// ReferralRepository persists referrer→referee links.
type ReferralRepository interface {
	// FindByReferee returns the (single) referral whose RefereeID
	// matches, or domain.ErrNotFound. A user can only have one referrer.
	FindByReferee(ctx context.Context, refereeID string) (*domain.Referral, error)

	// Create stores a new referral. Should fail with
	// domain.ErrAlreadyAttributed when refereeID already has a referral.
	Create(ctx context.Context, r domain.Referral) (*domain.Referral, error)

	// Activate transitions a Pending referral to Activated, recording
	// the reward credits and ActivatedAt. Returns domain.ErrAlreadyActivated
	// when called twice on the same referral, domain.ErrNotFound when
	// the referee has no referral.
	Activate(ctx context.Context, refereeID string, rewardCredits int) (*domain.Referral, error)

	// ListByReferrer returns paginated referrals where the user is the
	// referrer, plus the total count.
	ListByReferrer(ctx context.Context, referrerID string, page, limit int) ([]domain.Referral, int, error)

	// StatsByReferrer returns aggregate counts for the user's dashboard.
	StatsByReferrer(ctx context.Context, referrerID string) (*domain.Stats, error)
}
