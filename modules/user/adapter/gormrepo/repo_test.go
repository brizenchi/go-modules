package gormrepo

import (
	"context"
	"testing"

	"github.com/brizenchi/go-modules/modules/user/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRepoCreateAndFind(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	ctx := context.Background()

	user := &domain.User{
		Email:           "User@Example.com",
		Provider:        "email",
		ProviderSubject: "",
		Role:            "ADMIN",
		Plan:            "PRO",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if got.Email != "user@example.com" {
		t.Fatalf("email = %q, want normalized", got.Email)
	}
	if got.ProviderSubject != got.Email {
		t.Fatalf("provider_subject = %q, want email fallback", got.ProviderSubject)
	}
	if got.Role != domain.RoleAdmin {
		t.Fatalf("role = %q, want %q", got.Role, domain.RoleAdmin)
	}
	if got.Plan != domain.PlanPro {
		t.Fatalf("plan = %q, want %q", got.Plan, domain.PlanPro)
	}
}

func TestRepoAddCredits(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	ctx := context.Background()

	user := &domain.User{Email: "credits@example.com"}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.AddCredits(ctx, user.ID, 25); err != nil {
		t.Fatalf("AddCredits: %v", err)
	}

	got, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Credits != 25 {
		t.Fatalf("credits = %d, want 25", got.Credits)
	}
}

