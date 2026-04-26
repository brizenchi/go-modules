package port

import (
	"context"

	"github.com/brizenchi/go-modules/auth/domain"
)

// UserStore is the bridge between the auth module and the host's user table.
//
// Auth never owns a user table — it asks the host to find or create one
// matching the credentials it just verified. Hosts implement this against
// whatever schema they like; auth only cares about UserID and Email.
type UserStore interface {
	// FindByEmail returns the existing user identity for an email,
	// or domain.ErrUserNotFound when missing.
	FindByEmail(ctx context.Context, email string) (*domain.Identity, error)

	// FindOrCreateByEmail returns an existing user identity, creating
	// one if none exists. The IsNew field of the returned Identity must
	// be true exactly when a new user was created.
	FindOrCreateByEmail(ctx context.Context, email string) (*domain.Identity, error)

	// FindOrCreateFromOAuth returns the user identity for an OAuth
	// profile, creating or linking as necessary. Hosts decide how to
	// handle email collisions across providers.
	FindOrCreateFromOAuth(ctx context.Context, profile domain.OAuthProfile) (*domain.Identity, error)

	// FindByID returns the user identity by id.
	FindByID(ctx context.Context, userID string) (*domain.Identity, error)

	// MarkLogin records a successful login (typically updating last_login_at).
	MarkLogin(ctx context.Context, userID string) error
}

// RoleResolver assigns a Role to an Identity at login time.
//
// The default implementation in adapter/role/static returns the role
// specified by a config-driven admin allowlist; production deployments
// can swap in DB-backed RBAC.
type RoleResolver interface {
	Resolve(ctx context.Context, id domain.Identity) (domain.Role, error)
}
