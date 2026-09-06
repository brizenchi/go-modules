package user

import (
	"errors"
	"time"
)

var (
	ErrInvalidCreditOperation  = errors.New("invalid_credit_operation")
	ErrInsufficientCredits     = errors.New("insufficient_credits")
	ErrCreditConflict          = errors.New("idempotency_conflict")
	ErrCreditAlreadyRefunded   = errors.New("already_refunded")
	ErrCreditMigrationRequired = errors.New("credit_ledger_migration_required")
)

const (
	CreditKindGrant         = "grant"
	CreditKindConsume       = "consume"
	CreditKindRefund        = "refund"
	CreditKindExpire        = "expire"
	CreditKindOpening       = "opening"
	MaxCreditAmount   int64 = 1_000_000_000_000
)

// CreditTransaction is immutable. Amount is signed; BalanceAfter is historical,
// while GetCreditSummary returns the current spendable balance after expiry.
type CreditTransaction struct {
	ID           uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       string     `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Kind         string     `json:"kind" gorm:"type:varchar(16);not null;index"`
	Amount       int64      `json:"amount" gorm:"not null"`
	BalanceAfter int64      `json:"balance_after" gorm:"not null"`
	Source       string     `json:"source" gorm:"type:varchar(32);not null;uniqueIndex:uniq_credit_transaction_source"`
	SourceID     string     `json:"source_id" gorm:"type:varchar(255);not null;uniqueIndex:uniq_credit_transaction_source"`
	Reason       string     `json:"reason" gorm:"type:varchar(500);not null"`
	ActorID      string     `json:"actor_id" gorm:"type:varchar(36);not null;default:''"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	// Only refunds set this value, so uniqueness prevents a second full refund.
	RelatedTransactionID *uint64   `json:"related_transaction_id,omitempty" gorm:"uniqueIndex:uniq_credit_refund_transaction"`
	CreatedAt            time.Time `json:"created_at" gorm:"not null"`
}

func (CreditTransaction) TableName() string { return "user_credit_transactions" }

type CreditLot struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement"`
	UserID        string     `gorm:"type:varchar(36);not null;index:idx_credit_lot_user_expiry"`
	TransactionID uint64     `gorm:"not null;uniqueIndex"`
	Amount        int64      `gorm:"not null"`
	Remaining     int64      `gorm:"not null"`
	ExpiresAt     *time.Time `gorm:"index:idx_credit_lot_user_expiry"`
	CreatedAt     time.Time  `gorm:"not null"`
}

func (CreditLot) TableName() string { return "user_credit_lots" }

// CreditAllocation records exactly which grant lots funded each consumption.
type CreditAllocation struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	TransactionID uint64 `gorm:"not null;index"`
	LotID         uint64 `gorm:"not null;index"`
	Amount        int64  `gorm:"not null"`
}

func (CreditAllocation) TableName() string { return "user_credit_allocations" }

type CreditGrantInput struct {
	UserID, Source, SourceID, Reason, ActorID string
	Amount                                    int64
	ExpiresAt                                 *time.Time
}
type CreditConsumption struct {
	UserID, Source, SourceID, Reason string
	Amount                           int64
}
type CreditRefundInput struct {
	UserID                    string // must own the original consumption; callers cannot refund another user's transaction
	TransactionID             uint64
	SourceID, Reason, ActorID string
}
type CreditSummary struct {
	Balance         int64      `json:"balance"`
	ExpiringCredits int64      `json:"expiring_credits"` // remaining credits expiring within the next 30 days
	NextExpiryAt    *time.Time `json:"next_expiry_at,omitempty"`
}
type CreditPage struct {
	List  []CreditTransaction `json:"list"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}
