// Package event defines auth domain events. Listeners subscribe to
// these to drive project-specific side effects (provision relay key,
// auto-create project, send welcome email, ...).
package event

import (
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

// Kind identifies a domain event type.
type Kind string

const (
	// KindUserSignedUp fires once per new user, immediately after creation.
	// Listeners typically: provision a relay key, create a default project,
	// send a welcome email, attribute a referral.
	KindUserSignedUp Kind = "user.signed_up"

	// KindUserLoggedIn fires on every successful login (including the
	// first one — listeners should subscribe to either this OR
	// KindUserSignedUp depending on whether they want first-login or
	// every-login behavior).
	KindUserLoggedIn Kind = "user.logged_in"
)

// Envelope wraps every event with provenance.
type Envelope struct {
	Kind       Kind
	UserID     string
	OccurredAt time.Time
	Payload    any
}

// UserSignedUp is the payload for KindUserSignedUp.
type UserSignedUp struct {
	Identity domain.Identity
}

// UserLoggedIn is the payload for KindUserLoggedIn.
type UserLoggedIn struct {
	Identity domain.Identity
	Provider domain.Provider
}
