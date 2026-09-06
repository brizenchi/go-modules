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
		Code:        "exchange-code",
		UserID:      "user-1",
		Provider:    domain.ProviderGoogle,
		IsNew:       true,
		BindingHash: "binding-hash",
		ExpiresAt:   time.Now().UTC().Add(time.Minute),
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save exchange: %v", err)
	}
	if _, err := store.Consume(ctx, record.Code, "wrong-binding"); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("wrong binding error=%v, want ErrInvalidExchange", err)
	}
	got, err := store.Consume(ctx, record.Code, record.BindingHash)
	if err != nil {
		t.Fatalf("consume exchange: %v", err)
	}
	if got.UserID != record.UserID || got.Provider != record.Provider || !got.IsNew {
		t.Fatalf("unexpected exchange: %#v", got)
	}
	if _, err := store.Consume(ctx, record.Code, record.BindingHash); !errors.Is(err, domain.ErrInvalidExchange) {
		t.Fatalf("second consume error=%v, want ErrInvalidExchange", err)
	}
}

func TestSaveExchangeCodeCleansExpiredAndRejectsLiveCollision(t *testing.T) {
	db := openTestDB(t)
	store := New(db)
	now := time.Now().UTC()
	rows := []exchangeCodeRow{
		{Code: "expired", UserID: "old", Provider: "google", BindingHash: "old", ExpiresAt: now.Add(-time.Minute)},
		{Code: "live", UserID: "original", Provider: "github", BindingHash: "binding", ExpiresAt: now.Add(time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Save(t.Context(), domain.ExchangeCode{
		Code: "new", UserID: "new-user", Provider: domain.ProviderGoogle,
		BindingHash: "new-binding", ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&exchangeCodeRow{}).Where("code = ?", "expired").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("expired count=%d err=%v", count, err)
	}
	if err := store.Save(t.Context(), domain.ExchangeCode{
		Code: "live", UserID: "attacker", Provider: domain.ProviderGoogle,
		BindingHash: "attacker-binding", ExpiresAt: now.Add(time.Minute),
	}); err == nil {
		t.Fatal("live exchange-code collision must fail")
	}
	var live exchangeCodeRow
	if err := db.First(&live, "code = ?", "live").Error; err != nil {
		t.Fatal(err)
	}
	if live.UserID != "original" || live.BindingHash != "binding" {
		t.Fatalf("collision rebound live row: %#v", live)
	}
}

func TestStoreOAuthFlowIsBrowserBoundAndSingleUse(t *testing.T) {
	store := New(openTestDB(t))
	ctx := context.Background()
	flow := domain.OAuthFlow{
		StateHash: "state-hash", Provider: domain.ProviderGitHub,
		BindingHash: "cookie-binding", VerifierChallenge: "verifier-challenge",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	if err := store.SaveOAuthFlow(ctx, flow); err != nil {
		t.Fatalf("save flow: %v", err)
	}
	if _, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, "wrong-browser"); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("wrong binding error=%v, want ErrInvalidState", err)
	}
	got, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, flow.BindingHash)
	if err != nil {
		t.Fatalf("consume flow: %v", err)
	}
	if got.VerifierChallenge != flow.VerifierChallenge {
		t.Fatalf("challenge=%q, want %q", got.VerifierChallenge, flow.VerifierChallenge)
	}
	if _, err := store.ConsumeOAuthFlow(ctx, flow.Provider, flow.StateHash, flow.BindingHash); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("replay error=%v, want ErrInvalidState", err)
	}
}

func TestStoreExpiredOAuthFlowIsUnavailable(t *testing.T) {
	store := New(openTestDB(t))
	flow := domain.OAuthFlow{
		StateHash: "expired-state", Provider: domain.ProviderGoogle,
		BindingHash: "cookie-binding", VerifierChallenge: "challenge",
		ExpiresAt: time.Now().UTC().Add(-time.Second),
	}
	if err := store.SaveOAuthFlow(t.Context(), flow); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeOAuthFlow(t.Context(), flow.Provider, flow.StateHash, flow.BindingHash); !errors.Is(err, domain.ErrInvalidState) {
		t.Fatalf("expired flow error=%v, want ErrInvalidState", err)
	}
}

func TestSaveOAuthFlowCleansOnlyExpiredRows(t *testing.T) {
	db := openTestDB(t)
	store := New(db)
	now := time.Now().UTC()
	rows := []oauthFlowRow{
		{StateHash: "old", Provider: "google", BindingHash: "old-binding", VerifierChallenge: "old-challenge", ExpiresAt: now.Add(-time.Minute)},
		{StateHash: "live", Provider: "github", BindingHash: "live-binding", VerifierChallenge: "live-challenge", ExpiresAt: now.Add(time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.SaveOAuthFlow(t.Context(), domain.OAuthFlow{
		StateHash: "new", Provider: domain.ProviderGoogle, BindingHash: "new-binding",
		VerifierChallenge: "new-challenge", ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var remaining []oauthFlowRow
	if err := db.Order("state_hash").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 || remaining[0].StateHash != "live" || remaining[1].StateHash != "new" {
		t.Fatalf("remaining rows=%#v, want live and new", remaining)
	}
}
