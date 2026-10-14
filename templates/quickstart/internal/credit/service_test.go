package credit

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := user.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := user.NewRepository(db)
	for _, id := range []string{"alice", "bob"} {
		if err := repo.Create(t.Context(), &user.User{ID: id, Email: id + "@example.test"}); err != nil {
			t.Fatal(err)
		}
	}
	return New(repo), db
}

func TestCreditServiceLedgerLifecycle(t *testing.T) {
	s, _ := testService(t)
	ctx := t.Context()
	for range 2 {
		if err := s.GrantCredits(ctx, "alice", "test", "grant-1", 20); err != nil {
			t.Fatal(err)
		}
	}
	purchase := Consumption{UserID: "alice", Source: "export", SourceID: "alice:job-1", Amount: 6, Reason: "Export"}
	first, err := s.ConsumeCredits(ctx, purchase)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := s.ConsumeCredits(ctx, purchase)
	if err != nil || repeated.ID != first.ID {
		t.Fatalf("repeat=%+v err=%v", repeated, err)
	}
	summary, err := s.GetCreditSummary(ctx, "alice")
	if err != nil || summary.Balance != 14 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	purchase.SourceID = "alice:too-much"
	purchase.Amount = 15
	if _, err := s.ConsumeCredits(ctx, purchase); !errors.Is(err, user.ErrInsufficientCredits) {
		t.Fatalf("insufficient: %v", err)
	}
	if _, err := s.RefundCredits(ctx, Refund{UserID: "bob", TransactionID: first.ID, ActorID: "admin", SourceID: "wrong-owner", Reason: "Return"}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("ownership: %v", err)
	}
	for range 2 {
		if _, err := s.RefundByAdmin(ctx, first.ID, "admin", "refund-1", "Return"); err != nil {
			t.Fatal(err)
		}
	}
	summary, err = s.GetCreditSummary(ctx, "alice")
	if err != nil || summary.Balance != 20 {
		t.Fatalf("refund summary=%+v err=%v", summary, err)
	}
	rows, err := s.ListCreditTransactions(ctx, "alice", 1, 20)
	if err != nil || rows.Total != 3 {
		t.Fatalf("ledger=%+v err=%v", rows, err)
	}
}

func TestCreditServiceRejectsUnscopedQueriesAndPropagatesCancellation(t *testing.T) {
	s, _ := testService(t)
	if _, err := s.ListCreditTransactions(t.Context(), " ", 1, 20); !errors.Is(err, user.ErrInvalidCreditOperation) {
		t.Fatalf("unscoped: %v", err)
	}
	if _, err := s.GetCreditSummary(t.Context(), ""); !errors.Is(err, user.ErrInvalidCreditOperation) {
		t.Fatalf("empty owner: %v", err)
	}
	if _, err := s.ListAdminTransactions(t.Context(), "", 0, 20); !errors.Is(err, user.ErrInvalidCreditOperation) {
		t.Fatalf("pagination: %v", err)
	}
	if _, err := s.ListAdminTransactions(t.Context(), "", 1, 20); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.GetCreditSummary(ctx, "alice"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestCreditServiceConsumptionAndBusinessWriteRollbackTogether(t *testing.T) {
	s, db := testService(t)
	type Result struct {
		ID    uint `gorm:"primaryKey"`
		Value string
	}
	if err := db.AutoMigrate(&Result{}); err != nil {
		t.Fatal(err)
	}
	if err := s.GrantCredits(t.Context(), "alice", "test", "grant-1", 20); err != nil {
		t.Fatal(err)
	}
	in := Consumption{UserID: "alice", Source: "job", SourceID: "alice:job-1", Amount: 5, Reason: "Generate result"}
	failure := errors.New("business write failed")
	_, err := s.ConsumeCreditsAndDo(t.Context(), in, func(tx *gorm.DB, _ *Transaction) error {
		if err := tx.Create(&Result{Value: "rolled back"}).Error; err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	summary, err := s.GetCreditSummary(t.Context(), "alice")
	if err != nil || summary.Balance != 20 {
		t.Fatalf("rollback summary=%+v err=%v", summary, err)
	}
	var count int64
	if err := db.Model(&Result{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("results=%d err=%v", count, err)
	}
	calls := 0
	for range 2 {
		if _, err := s.ConsumeCreditsAndDo(t.Context(), in, func(tx *gorm.DB, _ *Transaction) error { calls++; return tx.Create(&Result{Value: "committed"}).Error }); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("business callback calls=%d", calls)
	}
	summary, err = s.GetCreditSummary(t.Context(), "alice")
	if err != nil || summary.Balance != 15 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}
