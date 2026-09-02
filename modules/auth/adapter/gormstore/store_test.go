package gormstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestStoreEmailCodeLifecycle(t *testing.T) {
	db := openTestDB(t)
	store := New(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := store.SaveCode(ctx, " User@Example.COM ", "123456", now.Add(time.Minute), now); err != nil {
		t.Fatalf("save code: %v", err)
	}
	code, sentAt, err := store.LoadCode(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("load code: %v", err)
	}
	if code != "123456" || !sentAt.Equal(now) {
		t.Fatalf("unexpected code record: code=%q sent_at=%s", code, sentAt)
	}

	for want := 1; want <= 2; want++ {
		got, err := store.IncrAttempts(ctx, "USER@example.com")
		if err != nil || got != want {
			t.Fatalf("attempt %d: got=%d err=%v", want, got, err)
		}
	}
	for want := 1; want <= 2; want++ {
		got, err := store.IncrDailyCount(ctx, "user@example.com", "2026-08-08")
		if err != nil || got != want {
			t.Fatalf("daily count %d: got=%d err=%v", want, got, err)
		}
	}

	if err := store.DeleteCode(ctx, "user@example.com"); err != nil {
		t.Fatalf("delete code: %v", err)
	}
	code, _, err = store.LoadCode(ctx, "user@example.com")
	if err != nil || code != "" {
		t.Fatalf("deleted code still available: code=%q err=%v", code, err)
	}
}

func TestStoreExpiredCodeIsUnavailable(t *testing.T) {
	store := New(openTestDB(t))
	ctx := context.Background()
	now := time.Now().UTC()
	if err := store.SaveCode(ctx, "user@example.com", "expired", now.Add(-time.Second), now.Add(-time.Minute)); err != nil {
		t.Fatalf("save code: %v", err)
	}
	code, _, err := store.LoadCode(ctx, "user@example.com")
	if err != nil || code != "" {
		t.Fatalf("expired code should be unavailable: code=%q err=%v", code, err)
	}
}

func TestStoreExchangeCodeIsSingleUse(t *testing.T) {
	store := New(openTestDB(t))
	ctx := context.Background()
	record := domain.ExchangeCode{
		Code:      "exchange-code",
		UserID:    "user-1",
		Provider:  domain.ProviderGoogle,
		IsNew:     true,
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save exchange: %v", err)
	}
	got, err := store.Consume(ctx, record.Code)
	if err != nil {
		t.Fatalf("consume exchange: %v", err)
	}
	if got.UserID != record.UserID || got.Provider != record.Provider || !got.IsNew {
		t.Fatalf("unexpected exchange: %#v", got)
	}
	if _, err := store.Consume(ctx, record.Code); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("second consume error=%v, want ErrInvalidExchange", err)
	}
}
