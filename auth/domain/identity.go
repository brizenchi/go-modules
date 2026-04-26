// Package domain holds the provider-agnostic types for the auth module.
package domain

import "time"

// Identity is the auth module's view of an authenticated principal.
//
// It is intentionally minimal — the host project's full User model lives
// in its own package and implements port.UserStore to bridge the two.
type Identity struct {
	UserID    string
	Email     string
	Username  string
	AvatarURL string
	Provider  Provider // empty for email-code auth
	Subject   string   // provider-side stable identifier (sub claim)
	Role      Role
	IsNew     bool // true on the request that created the user
}

// Provider names supported identity providers.
type Provider string

const (
	ProviderEmail     Provider = "email"     // passwordless email code
	ProviderGoogle    Provider = "google"    // Google OAuth
	ProviderAnthropic Provider = "anthropic" // Anthropic OAuth (alias)
)

func (p Provider) Valid() bool {
	switch p {
	case ProviderEmail, ProviderGoogle, ProviderAnthropic:
		return true
	}
	return false
}

// Role is the coarse-grained authorization level the auth module knows about.
//
// Anything finer (per-resource permissions) is the host application's
// concern; auth only certifies who, not what they can do.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// OAuthProfile is the normalized user profile returned by an OAuth provider.
type OAuthProfile struct {
	Provider  Provider
	Subject   string // provider's "sub" / unique id
	Email     string
	Username  string
	AvatarURL string
}

// Token is an issued access token (JWT or otherwise) bound to an Identity.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// WSTicket is a short-lived single-use credential for opening a
// privileged WebSocket connection (e.g. terminal access to a bot).
type WSTicket struct {
	Value     string
	UserID    string
	Scope     map[string]string // free-form: {"bot_id": "...", "project_id": "..."}
	ExpiresAt time.Time
}
