// Package domain holds the provider-agnostic types for the referral module.
package domain

import "time"

// Code is a referral invitation code owned by a single user.
//
// One user has one Code (unique by user_id). The Value is unique across
// users — collisions are detected by the storage layer.
type Code struct {
	UserID    string
	Value     string
	CreatedAt time.Time
}

// Status enumerates a referral's lifecycle.
//
//	pending    — the referee has signed up but no qualifying event yet
//	activated  — the referee has met the activation criterion (typically
//	             first paid subscription); reward (if any) was granted
//	expired    — the activation window passed without qualifying;
//	             no reward will be granted
type Status string

const (
	StatusPending   Status = "pending"
	StatusActivated Status = "activated"
	StatusExpired   Status = "expired"
)

// Referral is one referrer→referee link.
//
// It is created when a referee signs up and supplies a referral code
// (or has the code attached via cookie/state); it transitions to
// activated when the host's billing/business logic calls ActivateReferral.
type Referral struct {
	ID            uint64
	Code          string    // the code value used for attribution (denormalized)
	ReferrerID    string    // user who shared the code
	RefereeID     string    // user who used the code
	Status        Status
	ActivatedAt   *time.Time
	ExpiresAt     *time.Time // host-defined activation deadline
	RewardCredits int        // optional accounting field; the actual reward is granted by host listener
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Stats is a per-user aggregation for the dashboard.
type Stats struct {
	TotalReferred      int
	Activated          int
	Pending            int
	TotalRewardCredits int
}
