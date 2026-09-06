package user

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) creditNow() time.Time {
	if r.creditClock != nil {
		return r.creditClock().UTC()
	}
	return time.Now().UTC()
}

// withCreditAccount obtains the user row's write lock before any balance reads.
// The initial UPDATE also serializes SQLite transactions (SELECT FOR UPDATE is
// ignored by SQLite), avoiding stale read-then-write balances on either database.
func (r *Repository) withCreditAccount(ctx context.Context, userID string, fn func(*gorm.DB, *User) error) error {
	if strings.TrimSpace(userID) == "" {
		return ErrInvalidCreditOperation
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).Where("id = ?", strings.TrimSpace(userID)).UpdateColumn("credits", gorm.Expr("credits"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		var account User
		if err := tx.Where("id = ?", strings.TrimSpace(userID)).Take(&account).Error; err != nil {
			return err
		}
		if account.CreditsVersion != 1 {
			return ErrCreditMigrationRequired
		}
		if err := expireCreditLots(tx, &account, r.creditNow()); err != nil {
			return err
		}
		return fn(tx, &account)
	})
}

func expireCreditLots(tx *gorm.DB, account *User, now time.Time) error {
	var lots []CreditLot
	if err := tx.Where("user_id = ? AND remaining > 0 AND expires_at IS NOT NULL AND expires_at <= ?", account.ID, now).Order("expires_at ASC, id ASC").Find(&lots).Error; err != nil {
		return err
	}
	for _, lot := range lots {
		if account.Credits < lot.Remaining {
			return fmt.Errorf("credit ledger balance mismatch")
		}
		account.Credits -= lot.Remaining
		entry := CreditTransaction{UserID: account.ID, Kind: CreditKindExpire, Amount: -lot.Remaining, BalanceAfter: account.Credits, Source: "expiry", SourceID: strconv.FormatUint(lot.ID, 10), Reason: "Unused credits expired", ExpiresAt: lot.ExpiresAt}
		if err := insertCreditTransaction(tx, &entry); err != nil {
			return err
		}
		if err := tx.Model(&CreditLot{}).Where("id = ?", lot.ID).UpdateColumn("remaining", 0).Error; err != nil {
			return err
		}
	}
	if len(lots) > 0 {
		return saveCreditBalance(tx, account)
	}
	return nil
}

func saveCreditBalance(tx *gorm.DB, account *User) error {
	return tx.Model(&User{}).Where("id = ?", account.ID).UpdateColumn("credits", account.Credits).Error
}

func insertCreditTransaction(tx *gorm.DB, entry *CreditTransaction) error {
	// A globally shared external key can race across different user row locks.
	// ON CONFLICT keeps PostgreSQL's transaction usable and reports a stable
	// conflict instead of surfacing a database uniqueness error.
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCreditConflict
	}
	return nil
}

func (r *Repository) GetCreditSummary(ctx context.Context, userID string) (CreditSummary, error) {
	var summary CreditSummary
	err := r.withCreditAccount(ctx, userID, func(tx *gorm.DB, account *User) error {
		summary.Balance = account.Credits
		var lots []CreditLot
		if err := tx.Where("user_id = ? AND remaining > 0 AND expires_at IS NOT NULL", account.ID).Order("expires_at ASC, id ASC").Find(&lots).Error; err != nil {
			return err
		}
		deadline := r.creditNow().Add(30 * 24 * time.Hour)
		for _, lot := range lots {
			if summary.NextExpiryAt == nil {
				summary.NextExpiryAt = lot.ExpiresAt
			}
			if !lot.ExpiresAt.After(deadline) {
				summary.ExpiringCredits += lot.Remaining
			}
		}
		return nil
	})
	return summary, err
}

func (r *Repository) ListCreditTransactions(ctx context.Context, userID string, page, limit int) (CreditPage, error) {
	if page < 1 || page > 1_000_000 || limit < 1 || limit > 100 {
		return CreditPage{}, ErrInvalidCreditOperation
	}
	result := CreditPage{List: []CreditTransaction{}, Page: page, Limit: limit}
	list := func(tx *gorm.DB) error {
		query := tx.Model(&CreditTransaction{})
		if userID != "" {
			query = query.Where("user_id = ?", strings.TrimSpace(userID))
		}
		if err := query.Count(&result.Total).Error; err != nil {
			return err
		}
		return query.Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&result.List).Error
	}
	var err error
	if userID != "" {
		err = r.withCreditAccount(ctx, userID, func(tx *gorm.DB, _ *User) error { return list(tx) })
	} else {
		err = list(r.db.WithContext(ctx))
	}
	return result, err
}

func (r *Repository) FindCreditTransaction(ctx context.Context, id uint64) (*CreditTransaction, error) {
	var entry CreditTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&entry).Error
	return &entry, err
}

func validCreditKey(userID, source, sourceID, reason string) bool {
	return userID != "" && len(userID) <= 36 && source != "" && len(source) <= 32 && sourceID != "" && len(sourceID) <= 255 && len(reason) <= 500
}
func sameExpiry(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}

// GrantCredits preserves external webhook idempotency, including records in the
// pre-ledger user_credit_grants table retained by the explicit migration.
func (r *Repository) GrantCredits(ctx context.Context, userID, source, sourceID string, delta int64) error {
	if delta == 0 {
		return nil
	}
	_, err := r.GrantCreditsWithExpiry(ctx, CreditGrantInput{UserID: userID, Source: source, SourceID: sourceID, Amount: delta, Reason: source + " credit grant"})
	return err
}

// AddCredits remains for existing hosts. New callers should pass a stable key to
// GrantCredits or ConsumeCredits to make their own retries idempotent.
func (r *Repository) AddCredits(ctx context.Context, userID string, delta int64) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return r.GrantCredits(ctx, userID, "adjustment", uuid.NewString(), delta)
	}
	if delta == math.MinInt64 {
		return ErrInvalidCreditOperation
	}
	_, err := r.ConsumeCredits(ctx, CreditConsumption{UserID: userID, Source: "adjustment", SourceID: uuid.NewString(), Reason: "Credit adjustment", Amount: -delta})
	return err
}

func (r *Repository) GrantCreditsWithExpiry(ctx context.Context, in CreditGrantInput) (*CreditTransaction, error) {
	in.UserID, in.Source, in.SourceID, in.Reason, in.ActorID = strings.TrimSpace(in.UserID), strings.ToLower(strings.TrimSpace(in.Source)), strings.TrimSpace(in.SourceID), strings.TrimSpace(in.Reason), strings.TrimSpace(in.ActorID)
	if !validCreditKey(in.UserID, in.Source, in.SourceID, in.Reason) || len(in.ActorID) > 36 || in.Amount <= 0 || in.Amount > MaxCreditAmount {
		return nil, ErrInvalidCreditOperation
	}
	if in.ExpiresAt != nil {
		at := in.ExpiresAt.UTC().Truncate(time.Microsecond)
		in.ExpiresAt = &at
	}
	var entry CreditTransaction
	err := r.withCreditAccount(ctx, in.UserID, func(tx *gorm.DB, account *User) error {
		err := tx.Where("source = ? AND source_id = ?", in.Source, in.SourceID).Take(&entry).Error
		if err == nil {
			if entry.UserID != in.UserID || entry.Kind != CreditKindGrant || entry.Amount != in.Amount || entry.Reason != in.Reason || entry.ActorID != in.ActorID || !sameExpiry(entry.ExpiresAt, in.ExpiresAt) {
				return ErrCreditConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var archived CreditGrant
		err = tx.Where("source = ? AND source_id = ?", in.Source, in.SourceID).Take(&archived).Error
		if err == nil {
			if archived.UserID != in.UserID || archived.Amount != in.Amount || in.ExpiresAt != nil || in.ActorID != "" {
				return ErrCreditConflict
			}
			// The old event is already represented in the migrated opening balance.
			entry = CreditTransaction{UserID: in.UserID, Kind: CreditKindGrant, Amount: archived.Amount, Source: archived.Source, SourceID: archived.SourceID, Reason: "Archived grant included in migrated balance", CreatedAt: archived.CreatedAt, BalanceAfter: account.Credits}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if in.ExpiresAt != nil && !in.ExpiresAt.After(r.creditNow()) {
			return ErrInvalidCreditOperation
		}
		if account.Credits > math.MaxInt64-in.Amount {
			return ErrInvalidCreditOperation
		}
		account.Credits += in.Amount
		entry = CreditTransaction{UserID: in.UserID, Kind: CreditKindGrant, Amount: in.Amount, BalanceAfter: account.Credits, Source: in.Source, SourceID: in.SourceID, Reason: in.Reason, ActorID: in.ActorID, ExpiresAt: in.ExpiresAt}
		if err := insertCreditTransaction(tx, &entry); err != nil {
			return err
		}
		if err := tx.Create(&CreditGrant{UserID: in.UserID, Source: in.Source, SourceID: in.SourceID, Amount: in.Amount}).Error; err != nil {
			return err
		}
		if err := tx.Create(&CreditLot{UserID: in.UserID, TransactionID: entry.ID, Amount: in.Amount, Remaining: in.Amount, ExpiresAt: in.ExpiresAt}).Error; err != nil {
			return err
		}
		return saveCreditBalance(tx, account)
	})
	return &entry, err
}

func (r *Repository) ConsumeCredits(ctx context.Context, in CreditConsumption) (*CreditTransaction, error) {
	return r.ConsumeCreditsAndDo(ctx, in, nil)
}

// ConsumeCreditsAndDo charges once and persists the feature result in the same
// transaction. perform runs only for a new charge; its error rolls everything
// back. It must do database work only, never network calls or external effects.
func (r *Repository) ConsumeCreditsAndDo(ctx context.Context, in CreditConsumption, perform func(*gorm.DB, *CreditTransaction) error) (*CreditTransaction, error) {
	in.UserID, in.Source, in.SourceID, in.Reason = strings.TrimSpace(in.UserID), strings.ToLower(strings.TrimSpace(in.Source)), strings.TrimSpace(in.SourceID), strings.TrimSpace(in.Reason)
	if !validCreditKey(in.UserID, in.Source, in.SourceID, in.Reason) || in.Amount <= 0 || in.Amount > MaxCreditAmount {
		return nil, ErrInvalidCreditOperation
	}
	var entry CreditTransaction
	err := r.withCreditAccount(ctx, in.UserID, func(tx *gorm.DB, account *User) error {
		err := tx.Where("source = ? AND source_id = ?", in.Source, in.SourceID).Take(&entry).Error
		if err == nil {
			if entry.UserID != in.UserID || entry.Kind != CreditKindConsume || entry.Amount != -in.Amount || entry.Reason != in.Reason {
				return ErrCreditConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if account.Credits < in.Amount {
			return ErrInsufficientCredits
		}
		account.Credits -= in.Amount
		entry = CreditTransaction{UserID: in.UserID, Kind: CreditKindConsume, Amount: -in.Amount, BalanceAfter: account.Credits, Source: in.Source, SourceID: in.SourceID, Reason: in.Reason}
		if err := insertCreditTransaction(tx, &entry); err != nil {
			return err
		}
		var lots []CreditLot
		if err := tx.Where("user_id = ? AND remaining > 0", in.UserID).Order("CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END, expires_at ASC, id ASC").Find(&lots).Error; err != nil {
			return err
		}
		left := in.Amount
		for _, lot := range lots {
			used := min(left, lot.Remaining)
			if err := tx.Model(&CreditLot{}).Where("id = ?", lot.ID).UpdateColumn("remaining", lot.Remaining-used).Error; err != nil {
				return err
			}
			if err := tx.Create(&CreditAllocation{TransactionID: entry.ID, LotID: lot.ID, Amount: used}).Error; err != nil {
				return err
			}
			left -= used
			if left == 0 {
				break
			}
		}
		if left != 0 {
			return fmt.Errorf("credit ledger balance mismatch")
		}
		if err := saveCreditBalance(tx, account); err != nil {
			return err
		}
		if perform != nil {
			return perform(tx, &entry)
		}
		return nil
	})
	return &entry, err
}

// RefundCredits restores the complete original consumption once. Refund lots do
// not expire, so an operator can return credits even after the original grant's
// expiry. Partial refunds and refunding a grant/refund are deliberately rejected.
func (r *Repository) RefundCredits(ctx context.Context, in CreditRefundInput) (*CreditTransaction, error) {
	in.UserID, in.SourceID, in.Reason, in.ActorID = strings.TrimSpace(in.UserID), strings.TrimSpace(in.SourceID), strings.TrimSpace(in.Reason), strings.TrimSpace(in.ActorID)
	if !validCreditKey(in.UserID, "refund", in.SourceID, in.Reason) || in.TransactionID == 0 || in.Reason == "" || in.ActorID == "" || len(in.ActorID) > 36 {
		return nil, ErrInvalidCreditOperation
	}
	var entry CreditTransaction
	err := r.withCreditAccount(ctx, in.UserID, func(tx *gorm.DB, account *User) error {
		var original CreditTransaction
		if err := tx.Where("id = ? AND user_id = ?", in.TransactionID, in.UserID).Take(&original).Error; err != nil {
			return err
		}
		if original.Kind != CreditKindConsume || original.Amount >= 0 {
			return ErrInvalidCreditOperation
		}
		err := tx.Where("source = ? AND source_id = ?", "refund", in.SourceID).Take(&entry).Error
		if err == nil {
			if entry.UserID != in.UserID || entry.RelatedTransactionID == nil || *entry.RelatedTransactionID != in.TransactionID || entry.Reason != in.Reason || entry.ActorID != in.ActorID {
				return ErrCreditConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var previous int64
		if err := tx.Model(&CreditTransaction{}).Where("related_transaction_id = ?", in.TransactionID).Count(&previous).Error; err != nil {
			return err
		}
		if previous > 0 {
			return ErrCreditAlreadyRefunded
		}
		amount := -original.Amount
		if account.Credits > math.MaxInt64-amount {
			return ErrInvalidCreditOperation
		}
		account.Credits += amount
		entry = CreditTransaction{UserID: in.UserID, Kind: CreditKindRefund, Amount: amount, BalanceAfter: account.Credits, Source: "refund", SourceID: in.SourceID, Reason: in.Reason, ActorID: in.ActorID, RelatedTransactionID: &in.TransactionID}
		if err := insertCreditTransaction(tx, &entry); err != nil {
			return err
		}
		if err := tx.Create(&CreditLot{UserID: in.UserID, TransactionID: entry.ID, Amount: amount, Remaining: amount}).Error; err != nil {
			return err
		}
		return saveCreditBalance(tx, account)
	})
	return &entry, err
}
