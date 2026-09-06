package memstore

import (
	"context"
	"errors"
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
		Code:        "code-1",
		UserID:      "u1",
		Provider:    domain.ProviderGoogle,
		BindingHash: "binding-1",
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := s.Save(ctx, c); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consume(ctx, "code-1", "wrong-binding"); err == nil {
		t.Fatal("wrong binding must not consume exchange code")
	}
	got, err := s.Consume(ctx, "code-1", "binding-1")
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if got.UserID != "u1" {
		t.Errorf("user_id = %q", got.UserID)
	}
	// Second consume must fail (single-use).
	if _, err := s.Consume(ctx, "code-1", "binding-1"); err == nil {
		t.Error("expected error on second consume")
	}
}

func TestExchangeStore_RejectsExpired(t *testing.T) {
	s := NewExchangeStore()
	ctx := context.Background()
	_ = s.Save(ctx, domain.ExchangeCode{Code: "c1", UserID: "u", BindingHash: "binding", ExpiresAt: time.Now().Add(-time.Minute)})
	_, err := s.Consume(ctx, "c1", "binding")
	if err == nil {
		t.Error("expected expiry error")
	}
}

func TestExchangeStore_SaveCleansExpiredAndRejectsLiveCollision(t *testing.T) {
	store := NewExchangeStore()
	now := time.Now()
	store.codes["expired"] = domain.ExchangeCode{Code: "expired", ExpiresAt: now.Add(-time.Second)}
	live := domain.ExchangeCode{Code: "live", UserID: "original", BindingHash: "binding", ExpiresAt: now.Add(time.Minute)}
	store.codes[live.Code] = live
	if err := store.Save(t.Context(), domain.ExchangeCode{Code: "new", ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.codes["expired"]; exists {
		t.Fatal("expired exchange code was not cleaned")
	}
	if _, exists := store.codes["live"]; !exists {
		t.Fatal("live exchange code was removed")
	}
	if err := store.Save(t.Context(), domain.ExchangeCode{Code: "live", UserID: "attacker", ExpiresAt: now.Add(time.Minute)}); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("collision error=%v, want ErrInvalidExchange", err)
	}
	if got := store.codes["live"].UserID; got != "original" {
		t.Fatalf("collision rebound exchange code to %q", got)
	}
}

func TestOAuthFlowStore_BindingExpiryAndSingleUse(t *testing.T) {
	store := NewOAuthFlowStore()
	ctx := context.Background()
	flow := domain.OAuthFlow{
		StateHash: "state-hash", Provider: domain.ProviderGoogle,
		BindingHash: "cookie-hash", VerifierChallenge: "challenge",
		ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := store.SaveOAuthFlow(ctx, flow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, "wrong"); err == nil {
		t.Fatal("wrong browser binding must be rejected")
	}
	got, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, flow.BindingHash)
	if err != nil || got.VerifierChallenge != flow.VerifierChallenge {
		t.Fatalf("consume flow: got=%#v err=%v", got, err)
	}
	if _, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, flow.BindingHash); err == nil {
		t.Fatal("replayed flow must be rejected")
	}

	expired := flow
	expired.StateHash = "expired"
	expired.ExpiresAt = time.Now().Add(-time.Second)
	if err := store.SaveOAuthFlow(ctx, expired); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeOAuthFlow(ctx, expired.Provider, expired.StateHash, expired.BindingHash); err == nil {
		t.Fatal("expired flow must be rejected")
	}
}

func TestOAuthFlowStore_SaveCleansExpiredButKeepsLiveFlows(t *testing.T) {
	store := NewOAuthFlowStore()
	now := time.Now()
	store.flows["expired"] = domain.OAuthFlow{StateHash: "expired", ExpiresAt: now.Add(-time.Second)}
	store.flows["live"] = domain.OAuthFlow{StateHash: "live", ExpiresAt: now.Add(time.Minute)}
	if err := store.SaveOAuthFlow(t.Context(), domain.OAuthFlow{StateHash: "new", ExpiresAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, exists := store.flows["expired"]; exists {
		t.Fatal("expired flow was not cleaned")
	}
	if _, exists := store.flows["live"]; !exists {
		t.Fatal("live flow was removed")
	}
	if _, exists := store.flows["new"]; !exists {
		t.Fatal("new flow was not stored")
	}
}
