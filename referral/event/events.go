// Package event defines referral domain events.
package event

import (
	"time"

	"github.com/brizenchi/go-modules/referral/domain"
)

type Kind string

const (
	// KindReferralRegistered fires when a new referrer→referee link is
	// created (i.e. a new user supplied a valid referral code at signup).
	// Status at this point is Pending.
	KindReferralRegistered Kind = "referral.registered"

	// KindReferralActivated fires when a previously-pending referral
	// transitions to Activated. Listeners typically grant a reward
	// (e.g. add credits to the referrer's wallet via the billing module).
	KindReferralActivated Kind = "referral.activated"
)

// Envelope wraps every event with provenance.
type Envelope struct {
	Kind       Kind
	OccurredAt time.Time
	Payload    any
}

// ReferralRegistered is the payload for KindReferralRegistered.
type ReferralRegistered struct {
	Referral domain.Referral
}

// ReferralActivated is the payload for KindReferralActivated.
type ReferralActivated struct {
	Referral domain.Referral
}
