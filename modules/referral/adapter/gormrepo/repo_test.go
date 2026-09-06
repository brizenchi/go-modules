package gormrepo

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReferralTestRepo(t *testing.T) *ReferralRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(AutoMigrateModels()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewReferralRepo(db)
}

func TestReferralRepoActivatePersistsExpiredWithoutReward(t *testing.T) {
	repo := newReferralTestRepo(t)
	past := time.Now().UTC().Add(-time.Minute)
	created, err := repo.Create(context.Background(), domain.Referral{
		Code:       "INVTEST",
		ReferrerID: "referrer",
		RefereeID:  "referee",
		Status:     domain.StatusPending,
		ExpiresAt:  &past,
	})
	if err != nil {
		t.Fatalf("create referral: %v", err)
	}

	expired, err := repo.Activate(context.Background(), created.RefereeID, 50)
	if !errors.Is(err, domain.ErrExpired) {
		t.Fatalf("activate error=%v, want ErrExpired", err)
	}
	if expired == nil || expired.Status != domain.StatusExpired || expired.RewardCredits != 0 || expired.ActivatedAt != nil {
		t.Fatalf("expired referral=%+v", expired)
	}

	stored, err := repo.FindByReferee(context.Background(), created.RefereeID)
	if err != nil {
		t.Fatalf("find referral: %v", err)
	}
	if stored.Status != domain.StatusExpired || stored.RewardCredits != 0 || stored.ActivatedAt != nil {
		t.Fatalf("stored referral=%+v", stored)
	}
}

func TestReferralRepoActivateAllowsReferralBeforeDeadline(t *testing.T) {
	repo := newReferralTestRepo(t)
	future := time.Now().UTC().Add(time.Minute)
	created, err := repo.Create(context.Background(), domain.Referral{
		Code:       "INVFUTURE",
		ReferrerID: "referrer",
		RefereeID:  "referee",
		Status:     domain.StatusPending,
		ExpiresAt:  &future,
	})
	if err != nil {
		t.Fatalf("create referral: %v", err)
	}

	activated, err := repo.Activate(context.Background(), created.RefereeID, 50)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if activated.Status != domain.StatusActivated || activated.RewardCredits != 50 || activated.ActivatedAt == nil {
		t.Fatalf("activated referral=%+v", activated)
	}
}

func TestReferralRepoQueriesExpireOverduePendingRows(t *testing.T) {
	repo := newReferralTestRepo(t)
	past := time.Now().UTC().Add(-time.Minute)
	if _, err := repo.Create(context.Background(), domain.Referral{
		Code:       "INVOVERDUE",
		ReferrerID: "referrer",
		RefereeID:  "referee",
		Status:     domain.StatusPending,
		ExpiresAt:  &past,
	}); err != nil {
		t.Fatalf("create referral: %v", err)
	}
	if _, err := repo.Create(context.Background(), domain.Referral{
		Code:       "INVOTHER",
		ReferrerID: "other-referrer",
		RefereeID:  "other-referee",
		Status:     domain.StatusPending,
		ExpiresAt:  &past,
	}); err != nil {
		t.Fatalf("create other referral: %v", err)
	}

	stats, err := repo.StatsByReferrer(context.Background(), "referrer")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalReferred != 1 || stats.Pending != 0 || stats.Activated != 0 || stats.TotalRewardCredits != 0 {
		t.Fatalf("stats=%+v, want one expired referral", stats)
	}
	items, total, err := repo.ListByReferrer(context.Background(), "referrer", 1, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Status != domain.StatusExpired {
		t.Fatalf("items=%+v total=%d, want expired row", items, total)
	}
	other, err := repo.FindByReferee(context.Background(), "other-referee")
	if err != nil {
		t.Fatalf("find other referral: %v", err)
	}
	if other.Status != domain.StatusPending {
		t.Fatalf("other status=%s, want caller-scoped expiry update", other.Status)
	}
}

func TestReferralRepoConcurrentActivationPreservesFirstReward(t *testing.T) {
	// A file database with separate connections lets requests compete for the
	// same row. WAL and a busy timeout make SQLite wait for the winning writer.
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "referrals.db")+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(AutoMigrateModels()...); err != nil {
		t.Fatal(err)
	}
	repo := NewReferralRepo(db)
	ctx := context.Background()
	if _, err := repo.Create(ctx, domain.Referral{ReferrerID: "owner", RefereeID: "friend", Code: "INV"}); err != nil {
		t.Fatal(err)
	}
	type result struct {
		reward int
		err    error
	}
	const callbacks = 12
	// Force any read-before-write implementation to observe the same pending
	// row before it can save. Atomic updates need no initial pending read.
	var pendingReads sync.WaitGroup
	pendingReads.Add(callbacks)
	if err := db.Callback().Query().After("gorm:query").Register("test:concurrent_pending_reads", func(tx *gorm.DB) {
		if row, ok := tx.Statement.Dest.(*referralRow); ok && row.Status == string(domain.StatusPending) {
			pendingReads.Done()
			pendingReads.Wait()
		}
	}); err != nil {
		t.Fatal(err)
	}
	results := make(chan result, callbacks)
	start := make(chan struct{})
	var ready sync.WaitGroup
	ready.Add(callbacks)
	for reward := 1; reward <= callbacks; reward++ {
		go func(reward int) {
			ready.Done()
			<-start
			_, err := repo.Activate(ctx, "friend", reward)
			results <- result{reward: reward, err: err}
		}(reward)
	}
	ready.Wait()
	close(start)
	winners, winningReward := 0, 0
	for i := 0; i < callbacks; i++ {
		outcome := <-results
		if outcome.err == nil {
			winners++
			winningReward = outcome.reward
		} else if !errors.Is(outcome.err, domain.ErrAlreadyActivated) {
			t.Errorf("competing activation: %v", outcome.err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful transitions = %d, want 1", winners)
	}
	stored, err := repo.FindByReferee(ctx, "friend")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.StatusActivated || stored.RewardCredits != winningReward || stored.ActivatedAt == nil {
		t.Fatalf("winning activation was not preserved: %+v (reward %d)", stored, winningReward)
	}
}
