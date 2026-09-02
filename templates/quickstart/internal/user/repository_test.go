package user

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGrantCreditsIsIdempotentBySource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	account := &User{Email: "owner@example.com"}
	if err := repo.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := repo.GrantCredits(context.Background(), account.ID, "stripe", "evt_123", 100); err != nil {
			t.Fatalf("GrantCredits #%d: %v", i+1, err)
		}
	}

	got, err := repo.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Credits != 100 {
		t.Fatalf("credits=%d, want 100", got.Credits)
	}
	var grants int64
	if err := db.Model(&CreditGrant{}).Count(&grants).Error; err != nil {
		t.Fatal(err)
	}
	if grants != 1 {
		t.Fatalf("grants=%d, want 1", grants)
	}
}
