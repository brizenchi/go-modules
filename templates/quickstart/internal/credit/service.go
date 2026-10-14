// Package credit is the host's reusable credit application service.
// HTTP handlers, payment/referral hooks and product services use this package;
// user.Repository owns the transactional ledger and database locks.
package credit

import (
	"context"
	"strings"

	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/gorm"
)

// Keep one set of ledger types; callers need not import the user repository.
type Summary = user.CreditSummary
type Transaction = user.CreditTransaction
type Page = user.CreditPage
type Grant = user.CreditGrantInput
type Consumption = user.CreditConsumption
type Refund = user.CreditRefundInput

// Errors preserve errors.Is compatibility with the existing ledger.
var (
	ErrInvalidOperation    = user.ErrInvalidCreditOperation
	ErrInsufficientCredits = user.ErrInsufficientCredits
	ErrConflict            = user.ErrCreditConflict
	ErrAlreadyRefunded     = user.ErrCreditAlreadyRefunded
	ErrMigrationRequired   = user.ErrCreditMigrationRequired
)

// Store isolates database access. Every mutation must atomically check its key,
// update the balance and write the ledger; Service never prechecks the balance.
type Store interface {
	GetCreditSummary(context.Context, string) (Summary, error)
	ListCreditTransactions(context.Context, string, int, int) (Page, error)
	FindCreditTransaction(context.Context, uint64) (*Transaction, error)
	GrantCreditsWithExpiry(context.Context, Grant) (*Transaction, error)
	ConsumeCreditsAndDo(context.Context, Consumption, func(*gorm.DB, *Transaction) error) (*Transaction, error)
	RefundCredits(context.Context, Refund) (*Transaction, error)
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }

func (s *Service) GetCreditSummary(ctx context.Context, userID string) (Summary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Summary{}, user.ErrInvalidCreditOperation
	}
	return s.store.GetCreditSummary(ctx, userID)
}

// ListCreditTransactions is always owner scoped; an empty ID must not mean all users.
func (s *Service) ListCreditTransactions(ctx context.Context, userID string, page, limit int) (Page, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Page{}, user.ErrInvalidCreditOperation
	}
	return s.ListAdminTransactions(ctx, userID, page, limit)
}

// ListAdminTransactions permits an empty user filter. The caller must authorize an administrator.
func (s *Service) ListAdminTransactions(ctx context.Context, userID string, page, limit int) (Page, error) {
	if page < 1 || page > 1_000_000 || limit < 1 || limit > 100 {
		return Page{}, user.ErrInvalidCreditOperation
	}
	return s.store.ListCreditTransactions(ctx, strings.TrimSpace(userID), page, limit)
}

func (s *Service) GrantCredits(ctx context.Context, userID, source, sourceID string, amount int64) error {
	if amount == 0 {
		return nil
	} // Compatibility with optional signup/purchase rewards.
	_, err := s.GrantCreditsWithExpiry(ctx, Grant{UserID: userID, Source: source, SourceID: sourceID, Amount: amount, Reason: source + " credit grant"})
	return err
}
func (s *Service) GrantCreditsWithExpiry(ctx context.Context, in Grant) (*Transaction, error) {
	return s.store.GrantCreditsWithExpiry(ctx, in)
}
func (s *Service) ConsumeCredits(ctx context.Context, in Consumption) (*Transaction, error) {
	return s.ConsumeCreditsAndDo(ctx, in, nil)
}

// ConsumeCreditsAndDo is the advanced local-DB operation. perform runs only on
// the first successful charge, in the same transaction. Never call an external
// API here: database rollback cannot undo an external side effect.
func (s *Service) ConsumeCreditsAndDo(ctx context.Context, in Consumption, perform func(*gorm.DB, *Transaction) error) (*Transaction, error) {
	return s.store.ConsumeCreditsAndDo(ctx, in, perform)
}
func (s *Service) RefundCredits(ctx context.Context, in Refund) (*Transaction, error) {
	return s.store.RefundCredits(ctx, in)
}

// RefundByAdmin resolves ownership from the original transaction, not user input.
// HTTP callers must authenticate the administrator before invoking this method.
func (s *Service) RefundByAdmin(ctx context.Context, transactionID uint64, actorID, key, reason string) (*Transaction, error) {
	if transactionID == 0 || strings.TrimSpace(actorID) == "" || strings.TrimSpace(key) == "" || strings.TrimSpace(reason) == "" {
		return nil, user.ErrInvalidCreditOperation
	}
	original, err := s.store.FindCreditTransaction(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	return s.RefundCredits(ctx, Refund{UserID: original.UserID, TransactionID: transactionID, ActorID: actorID, SourceID: actorID + ":" + key, Reason: reason})
}

var _ Store = (*user.Repository)(nil)
