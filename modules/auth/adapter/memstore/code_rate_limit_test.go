package memstore

import (
	"context"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
)

func TestCodeStore_SaveAndLoad(t *testing.T) {
	s := NewCodeStore()
	ctx := context.Background()
	now := time.Now()

	if err := s.SaveCode(ctx, "a@b", "123456", now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	code, lastSent, err := s.LoadCode(ctx, "a@b")
	if err != nil {
		t.Fatal(err)
	}
	if code != "123456" || lastSent.Unix() != now.Unix() {
		t.Errorf("got code=%q lastSent=%v", code, lastSent)
	}
}

func TestCodeStore_LoadReturnsEmptyAfterTTL(t *testing.T) {
	s := NewCodeStore()
	ctx := context.Background()
	now := time.Now()
	_ = s.SaveCode(ctx, "a@b", "x", now.Add(-time.Second), now)
	code, _, _ := s.LoadCode(ctx, "a@b")
	if code != "" {
		t.Errorf("expected empty code after expiry, got %q", code)
	}
}

func TestCodeStore_DeleteCode(t *testing.T) {
	s := NewCodeStore()
	ctx := context.Background()
	_ = s.SaveCode(ctx, "a@b", "x", time.Now().Add(time.Minute), time.Now())
	_ = s.DeleteCode(ctx, "a@b")
	c, _, _ := s.LoadCode(ctx, "a@b")
	if c != "" {
		t.Error("expected code to be deleted")
	}
}

func TestCodeStore_IncrAttempts(t *testing.T) {
	s := NewCodeStore()
	ctx := context.Background()
	_ = s.SaveCode(ctx, "a@b", "x", time.Now().Add(time.Minute), time.Now())
	for i := 1; i <= 3; i++ {
		got, _ := s.IncrAttempts(ctx, "a@b")
		if got != i {
			t.Errorf("attempt %d = %d", i, got)
		}
	}
}

func TestCodeStore_IncrDailyCount(t *testing.T) {
	s := NewCodeStore()
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		got, _ := s.IncrDailyCount(ctx, "a@b", "2026-04-26")
		if got != i {
			t.Errorf("day count %d = %d", i, got)
		}
	}
	// Different bucket counts independently.
	got, _ := s.IncrDailyCount(ctx, "a@b", "2026-04-27")
	if got != 1 {
		t.Errorf("new bucket = %d, want 1", got)
	}
}

func TestExchangeStore_RoundTrip(t *testing.T) {
	s := NewExchangeStore()
	ctx := context.Background()
	c := domain.ExchangeCode{
		Code:      "code-1",
		UserID:    "u1",
		Provider:  domain.ProviderGoogle,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := s.Save(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Consume(ctx, "code-1")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.UserID != "u1" {
		t.Errorf("user_id = %q", got.UserID)
	}
	// Second consume must fail (single-use).
	if _, err := s.Consume(ctx, "code-1"); err == nil {
		t.Error("expected error on second consume")
	}
}

func TestExchangeStore_RejectsExpired(t *testing.T) {
	s := NewExchangeStore()
	ctx := context.Background()
	_ = s.Save(ctx, domain.ExchangeCode{Code: "c1", UserID: "u", ExpiresAt: time.Now().Add(-time.Minute)})
	_, err := s.Consume(ctx, "c1")
	if err == nil {
		t.Error("expected expiry error")
	}
}
