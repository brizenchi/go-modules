package user

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func creditTestRepo(t *testing.T, opening int64) (*Repository, *User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "credits.db")+"?_busy_timeout=10000&_journal_mode=WAL"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	account := &User{Email: "credits@example.com", Credits: opening}
	if err := repo.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	return repo, account
}

func requireBalance(t *testing.T, repo *Repository, account *User, want int64) {
	t.Helper()
	summary, err := repo.GetCreditSummary(context.Background(), account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Balance != want {
		t.Fatalf("balance=%d want=%d", summary.Balance, want)
	}
	var lots int64
	if err := repo.db.Model(&CreditLot{}).Select("COALESCE(SUM(remaining),0)").Where("user_id = ?", account.ID).Scan(&lots).Error; err != nil {
		t.Fatal(err)
	}
	if lots != want {
		t.Fatalf("lot balance=%d want=%d", lots, want)
	}
	var stored User
	if err := repo.db.Where("id = ?", account.ID).Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Credits != want {
		t.Fatalf("cached balance=%d want=%d", stored.Credits, want)
	}
}

func TestConcurrentConsumptionCannotOverspend(t *testing.T) {
	repo, account := creditTestRepo(t, 7)
	start := make(chan struct{})
	results := make(chan error, 20)
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			_, err := repo.ConsumeCredits(context.Background(), CreditConsumption{UserID: account.ID, Source: "test", SourceID: fmt.Sprint(i), Amount: 1, Reason: "Concurrent work"})
			results <- err
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	successes, insufficient := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrInsufficientCredits) {
			insufficient++
		} else {
			t.Errorf("unexpected concurrent error: %v", err)
		}
	}
	if successes != 7 || insufficient != 13 {
		t.Fatalf("successes=%d insufficient=%d", successes, insufficient)
	}
	requireBalance(t, repo, account, 0)
}

func TestConcurrentSameConsumptionChargesOnce(t *testing.T) {
	repo, account := creditTestRepo(t, 20)
	var workers sync.WaitGroup
	results := make(chan uint64, 12)
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			entry, err := repo.ConsumeCredits(context.Background(), CreditConsumption{UserID: account.ID, Source: "test", SourceID: "same-request", Amount: 3, Reason: "Same work"})
			if err != nil {
				t.Errorf("consume: %v", err)
				return
			}
			results <- entry.ID
		}()
	}
	workers.Wait()
	close(results)
	var first uint64
	for id := range results {
		if first == 0 {
			first = id
		}
		if id != first {
			t.Errorf("different transaction IDs %d and %d", first, id)
		}
	}
	requireBalance(t, repo, account, 17)
}

func TestExpiryUsesEarliestLotsAndUpdatesProfile(t *testing.T) {
	repo, account := creditTestRepo(t, 5)
	now := time.Now().UTC().Truncate(time.Second)
	repo.creditClock = func() time.Time { return now }
	expires := now.Add(time.Hour)
	if _, err := repo.GrantCreditsWithExpiry(context.Background(), CreditGrantInput{UserID: account.ID, Source: "promotion", SourceID: "expires-soon", Amount: 10, ExpiresAt: &expires}); err != nil {
		t.Fatal(err)
	}
	summary, err := repo.GetCreditSummary(context.Background(), account.ID)
	if err != nil || summary.Balance != 15 || summary.ExpiringCredits != 10 || summary.NextExpiryAt == nil || !summary.NextExpiryAt.Equal(expires) {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	entry, err := repo.ConsumeCredits(context.Background(), CreditConsumption{UserID: account.ID, Source: "test", SourceID: "use-expiring", Amount: 7})
	if err != nil {
		t.Fatal(err)
	}
	var allocation CreditAllocation
	if err := repo.db.Where("transaction_id = ?", entry.ID).Take(&allocation).Error; err != nil {
		t.Fatal(err)
	}
	var lot CreditLot
	if err := repo.db.Where("id = ?", allocation.LotID).Take(&lot).Error; err != nil {
		t.Fatal(err)
	}
	if lot.ExpiresAt == nil || lot.Remaining != 3 {
		t.Fatalf("wrong lot consumed: %+v", lot)
	}
	now = expires // deadline is exclusive
	profile, err := repo.FindByID(context.Background(), account.ID)
	if err != nil || profile.Credits != 5 {
		t.Fatalf("expired profile=%+v err=%v", profile, err)
	}
	requireBalance(t, repo, account, 5)
	page, err := repo.ListCreditTransactions(context.Background(), account.ID, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	var expiries int
	for _, row := range page.List {
		if row.Kind == CreditKindExpire {
			expiries++
			if row.Amount != -3 {
				t.Fatalf("expiry amount=%d", row.Amount)
			}
		}
	}
	if expiries != 1 {
		t.Fatalf("expiry entries=%d", expiries)
	}
}

func TestGrantReplayRejectsConflictingInputsAndUsers(t *testing.T) {
	repo, account := creditTestRepo(t, 0)
	ctx := context.Background()
	in := CreditGrantInput{UserID: account.ID, Source: "stripe", SourceID: "event-123", Amount: 25, Reason: "Purchase"}
	one, err := repo.GrantCreditsWithExpiry(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	two, err := repo.GrantCreditsWithExpiry(ctx, in)
	if err != nil || two.ID != one.ID {
		t.Fatalf("replay=%+v err=%v", two, err)
	}
	in.Amount++
	if _, err := repo.GrantCreditsWithExpiry(ctx, in); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("amount conflict: %v", err)
	}
	other := &User{Email: "other@example.com"}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	in.UserID = other.ID
	in.Amount = 25
	if _, err := repo.GrantCreditsWithExpiry(ctx, in); !errors.Is(err, ErrCreditConflict) {
		t.Fatalf("ownership conflict: %v", err)
	}
	requireBalance(t, repo, account, 25)
	requireBalance(t, repo, other, 0)
}

func TestFullRefundIsOwnedIdempotentAndCannotOverRefund(t *testing.T) {
	repo, account := creditTestRepo(t, 10)
	ctx := context.Background()
	spent, err := repo.ConsumeCredits(ctx, CreditConsumption{UserID: account.ID, Source: "test", SourceID: "spend", Amount: 6})
	if err != nil {
		t.Fatal(err)
	}
	other := &User{Email: "other@example.com"}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	in := CreditRefundInput{UserID: other.ID, TransactionID: spent.ID, SourceID: "refund-1", Reason: "Task failed", ActorID: "admin"}
	if _, err := repo.RefundCredits(ctx, in); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("wrong owner refund: %v", err)
	}
	in.UserID = account.ID
	one, err := repo.RefundCredits(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	two, err := repo.RefundCredits(ctx, in)
	if err != nil || two.ID != one.ID {
		t.Fatalf("replay=%+v err=%v", two, err)
	}
	in.SourceID = "refund-2"
	if _, err := repo.RefundCredits(ctx, in); !errors.Is(err, ErrCreditAlreadyRefunded) {
		t.Fatalf("second refund: %v", err)
	}
	in.TransactionID = one.ID
	if _, err := repo.RefundCredits(ctx, in); !errors.Is(err, ErrInvalidCreditOperation) {
		t.Fatalf("refund of refund: %v", err)
	}
	if one.Amount != 6 || one.ActorID != "admin" || one.ExpiresAt != nil {
		t.Fatalf("bad refund=%+v", one)
	}
	requireBalance(t, repo, account, 10)
}

func TestFailedBusinessOperationRollsBackChargeAndCanRetry(t *testing.T) {
	repo, account := creditTestRepo(t, 5)
	ctx := context.Background()
	in := CreditConsumption{UserID: account.ID, Source: "test", SourceID: "transaction", Amount: 2}
	fail := errors.New("save result failed")
	if _, err := repo.ConsumeCreditsAndDo(ctx, in, func(*gorm.DB, *CreditTransaction) error { return fail }); !errors.Is(err, fail) {
		t.Fatalf("got=%v", err)
	}
	requireBalance(t, repo, account, 5)
	called := 0
	for i := 0; i < 2; i++ {
		if _, err := repo.ConsumeCreditsAndDo(ctx, in, func(*gorm.DB, *CreditTransaction) error { called++; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if called != 1 {
		t.Fatalf("callback called %d times", called)
	}
	requireBalance(t, repo, account, 3)
}

func TestLegacyBalancesRequireExplicitMigrationAndKeepGrantDedupe(t *testing.T) {
	repo, _ := creditTestRepo(t, 0)
	ctx := context.Background()
	legacy := &User{Email: "legacy@example.com", Credits: 37}
	if err := repo.db.Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.db.Create(&CreditGrant{UserID: legacy.ID, Source: "stripe", SourceID: "legacy-event", Amount: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetCreditSummary(ctx, legacy.ID); !errors.Is(err, ErrCreditMigrationRequired) {
		t.Fatalf("unmigrated read=%v", err)
	}
	if err := repo.GrantCredits(ctx, legacy.ID, "stripe", "new-event", 10); !errors.Is(err, ErrCreditMigrationRequired) {
		t.Fatalf("unmigrated mutation=%v", err)
	}
	profile, err := repo.FindByID(ctx, legacy.ID)
	if err != nil || profile.Credits != 37 {
		t.Fatalf("legacy balance lost: %+v err=%v", profile, err)
	}
	// Exercise the explicit SQL migration's opening-balance semantics on SQLite.
	if err := repo.db.Transaction(func(tx *gorm.DB) error {
		entry := CreditTransaction{UserID: legacy.ID, Kind: CreditKindOpening, Amount: 37, BalanceAfter: 37, Source: "opening", SourceID: legacy.ID, Reason: "Migrated opening balance"}
		if err := tx.Create(&entry).Error; err != nil {
			return err
		}
		if err := tx.Create(&CreditLot{UserID: legacy.ID, TransactionID: entry.ID, Amount: 37, Remaining: 37}).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", legacy.ID).UpdateColumn("credits_version", 1).Error
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.GrantCredits(ctx, legacy.ID, "stripe", "legacy-event", 100); err != nil {
		t.Fatal(err)
	}
	requireBalance(t, repo, legacy, 37)
	if err := repo.GrantCredits(ctx, legacy.ID, "stripe", "new-event", 10); err != nil {
		t.Fatal(err)
	}
	requireBalance(t, repo, legacy, 47)
}

func TestStaleProfileSaveCannotOverwriteCredits(t *testing.T) {
	repo, account := creditTestRepo(t, 10)
	ctx := context.Background()
	stale, err := repo.FindByID(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ConsumeCredits(ctx, CreditConsumption{UserID: account.ID, Source: "test", SourceID: "charge", Amount: 4}); err != nil {
		t.Fatal(err)
	}
	stale.Username = "Updated name"
	if err := repo.Save(ctx, stale); err != nil {
		t.Fatal(err)
	}
	requireBalance(t, repo, account, 6)
}

func TestLedgerPaginationIsStableAndUserScoped(t *testing.T) {
	repo, account := creditTestRepo(t, 0)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := repo.GrantCredits(ctx, account.ID, "test", fmt.Sprint(i), 1); err != nil {
			t.Fatal(err)
		}
	}
	other := &User{Email: "other@example.com", Credits: 99}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListCreditTransactions(ctx, account.ID, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || len(page.List) != 2 || page.List[0].SourceID != "2" || page.List[1].SourceID != "1" {
		t.Fatalf("bad page=%+v", page)
	}
}

func TestRefundAfterOriginalGrantExpiryRemainsSpendable(t *testing.T) {
	repo, account := creditTestRepo(t, 0)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	repo.creditClock = func() time.Time { return now }
	expires := now.Add(time.Hour)
	grant, err := repo.GrantCreditsWithExpiry(ctx, CreditGrantInput{UserID: account.ID, Source: "test", SourceID: "refundable-promo", Amount: 5, ExpiresAt: &expires})
	if err != nil {
		t.Fatal(err)
	}
	spent, err := repo.ConsumeCredits(ctx, CreditConsumption{UserID: account.ID, Source: "task", SourceID: "failed-task", Amount: 3})
	if err != nil {
		t.Fatal(err)
	}
	now = expires.Add(time.Hour)
	refund, err := repo.RefundCredits(ctx, CreditRefundInput{UserID: account.ID, TransactionID: spent.ID, SourceID: "refund-failed-task", Reason: "Task could not complete", ActorID: "admin"})
	if err != nil || refund.ExpiresAt != nil {
		t.Fatalf("refund=%+v err=%v", refund, err)
	}
	requireBalance(t, repo, account, 3)
	// A delayed replay of the expired grant must not resurrect its unused credits.
	replayed, err := repo.GrantCreditsWithExpiry(ctx, CreditGrantInput{UserID: account.ID, Source: "test", SourceID: "refundable-promo", Amount: 5, ExpiresAt: &expires})
	if err != nil || replayed.ID != grant.ID {
		t.Fatalf("grant replay=%+v err=%v", replayed, err)
	}
	requireBalance(t, repo, account, 3)
}
