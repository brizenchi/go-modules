package user

import (
	"context"
	"errors"
	"testing"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	return NewRepository(db)
}

func TestAuthStoreEmailCreatesHostUser(t *testing.T) {
	repo := newTestRepository(t)
	store := NewAuthStore(repo)

	identity, err := store.FindOrCreateByEmail(context.Background(), " USER@Example.COM ")
	if err != nil {
		t.Fatalf("FindOrCreateByEmail: %v", err)
	}
	if !identity.IsNew || identity.Email != "user@example.com" || identity.Provider != authdomain.ProviderEmail {
		t.Fatalf("unexpected identity: %#v", identity)
	}

	again, err := store.FindOrCreateByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("second FindOrCreateByEmail: %v", err)
	}
	if again.IsNew || again.UserID != identity.UserID {
		t.Fatalf("second identity: %#v", again)
	}

	user, err := repo.FindByID(context.Background(), identity.UserID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if !user.EmailVerified || user.Email != "user@example.com" {
		t.Fatalf("unexpected host user: %#v", user)
	}
}

func TestAuthStoreOAuthLinksExistingEmail(t *testing.T) {
	repo := newTestRepository(t)
	store := NewAuthStore(repo)
	created, err := store.FindOrCreateByEmail(context.Background(), "owner@example.com")
	if err != nil {
		t.Fatalf("create email user: %v", err)
	}

	identity, err := store.FindOrCreateFromOAuth(context.Background(), authdomain.OAuthProfile{
		Provider:  authdomain.ProviderGitHub,
		Subject:   "github-123",
		Email:     "owner@example.com",
		Username:  "Owner",
		AvatarURL: "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("FindOrCreateFromOAuth: %v", err)
	}
	if identity.IsNew || identity.UserID != created.UserID || identity.Provider != authdomain.ProviderGitHub {
		t.Fatalf("unexpected linked identity: %#v", identity)
	}

	linked, err := repo.FindIdentity(context.Background(), authdomain.ProviderGitHub, "github-123")
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}
	if linked.UserID != created.UserID {
		t.Fatalf("oauth link not persisted: %#v", linked)
	}
}

// A Google account whose e-mail changed after signup must reach its original
// user through the linked subject; docs/migrate-from-saascore/user_identities.sql
// creates that link for users from saascore, which matched by subject first.
func TestAuthStoreOAuthPrefersLinkedSubjectOverEmail(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepository(t)
	store := NewAuthStore(repo)
	legacy, err := store.FindOrCreateByEmail(ctx, "old-address@example.com")
	if err != nil {
		t.Fatalf("create legacy user: %v", err)
	}
	if err := repo.LinkIdentity(ctx, legacy.UserID, authdomain.ProviderGoogle, "google-sub-1"); err != nil {
		t.Fatalf("LinkIdentity: %v", err)
	}

	identity, err := store.FindOrCreateFromOAuth(ctx, authdomain.OAuthProfile{
		Provider: authdomain.ProviderGoogle,
		Subject:  "google-sub-1",
		Email:    "new-address@example.com",
	})
	if err != nil {
		t.Fatalf("FindOrCreateFromOAuth: %v", err)
	}
	if identity.IsNew || identity.UserID != legacy.UserID || identity.Email != "new-address@example.com" {
		t.Fatalf("linked subject did not reach the original account: %#v", identity)
	}

	unlinked, err := store.FindOrCreateFromOAuth(ctx, authdomain.OAuthProfile{
		Provider: authdomain.ProviderGoogle,
		Subject:  "google-sub-2",
		Email:    "another-address@example.com",
	})
	if err != nil {
		t.Fatalf("FindOrCreateFromOAuth without link: %v", err)
	}
	if !unlinked.IsNew {
		t.Fatalf("an unlinked subject with an unknown e-mail must create a user: %#v", unlinked)
	}
}

func TestAuthStoreReturnsAuthNotFound(t *testing.T) {
	store := NewAuthStore(newTestRepository(t))
	_, err := store.FindByEmail(context.Background(), "missing@example.com")
	if !errors.Is(err, authdomain.ErrUserNotFound) {
		t.Fatalf("FindByEmail error=%v, want ErrUserNotFound", err)
	}
}

func TestBillingLookupUsesOnlyStableAccountFields(t *testing.T) {
	repo := newTestRepository(t)
	store := NewAuthStore(repo)
	identity, err := store.FindOrCreateByEmail(context.Background(), "billing@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	lookup := NewBillingLookup(repo)
	account, err := lookup.FindBillingAccount(context.Background(), identity.UserID)
	if err != nil {
		t.Fatalf("FindBillingAccount: %v", err)
	}
	if account.UserID != identity.UserID || account.Email != "billing@example.com" {
		t.Fatalf("unexpected account: %#v", account)
	}
	userID, err := lookup.FindUserIDByEmail(context.Background(), " BILLING@example.com ")
	if err != nil || userID != identity.UserID {
		t.Fatalf("FindUserIDByEmail: id=%q err=%v", userID, err)
	}
}
