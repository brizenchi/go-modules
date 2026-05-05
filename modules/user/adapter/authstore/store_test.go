package authstore

import (
	"context"
	"testing"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuthTestRepo(t *testing.T) *gormrepo.Repo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gormrepo.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return gormrepo.New(db)
}

func TestFindOrCreateByEmail(t *testing.T) {
	users := newAuthTestRepo(t)
	store := New(users)
	ctx := context.Background()

	id, err := store.FindOrCreateByEmail(ctx, "User@Example.com")
	if err != nil {
		t.Fatalf("FindOrCreateByEmail: %v", err)
	}
	if !id.IsNew {
		t.Fatal("expected new user")
	}
	if id.Email != "user@example.com" {
		t.Fatalf("email = %q, want normalized", id.Email)
	}
}

func TestFindOrCreateFromOAuth(t *testing.T) {
	users := newAuthTestRepo(t)
	store := New(users)
	ctx := context.Background()

	id, err := store.FindOrCreateFromOAuth(ctx, authdomain.OAuthProfile{
		Provider:  authdomain.ProviderGoogle,
		Subject:   "sub-1",
		Email:     "oauth@example.com",
		Username:  "oauth-user",
		AvatarURL: "https://cdn.example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("FindOrCreateFromOAuth: %v", err)
	}
	if !id.IsNew {
		t.Fatal("expected new oauth user")
	}
	if id.Provider != authdomain.ProviderGoogle {
		t.Fatalf("provider = %q, want google", id.Provider)
	}
}
