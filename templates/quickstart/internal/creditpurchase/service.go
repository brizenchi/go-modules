// Package creditpurchase fulfills paid credit checkouts from both payment returns
// and webhooks. The checkout session, rather than each notification, identifies a purchase.
package creditpurchase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = gorm.ErrRecordNotFound
	ErrConflict    = credit.ErrConflict
	ErrUnavailable = errors.New("credit purchase confirmation unavailable")
	ErrInvalid     = errors.New("invalid credit purchase")
)

// Result distinguishes confirmed credit grants from provider payment confirmation.
type Result struct {
	State          string `json:"state"`
	CreditsGranted int64  `json:"credits_granted"`
}

// Service unifies return-page and webhook credit fulfillment.
type Service struct {
	repo    repository
	credits *credit.Service
}

// New uses the existing checkout history and shared transactional credit service.
func New(db *gorm.DB, credits *credit.Service) *Service {
	return &Service{repo: repository{db}, credits: credits}
}

// Status reads only the authenticated owner's checkout and immutable grant.
func (s *Service) Status(ctx context.Context, owner, id string) (Result, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(id) == "" {
		return Result{}, ErrInvalid
	}
	if err := s.repo.owned(ctx, owner, id); err != nil {
		return Result{}, err
	}
	aliases, state, err := s.repo.related(ctx, owner, "stripe", id)
	if err != nil {
		return Result{}, err
	}
	grant, err := s.credits.FindGrant(ctx, owner, "stripe", append([]string{id}, aliases...))
	if err == nil {
		return Result{State: "fulfilled", CreditsGranted: grant.Amount}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Result{}, err
	}
	return Result{State: state}, nil
}

// Confirm verifies provider payment before fulfilling; GET Status stays read-only.
func (s *Service) Confirm(ctx context.Context, owner, id string, reader port.CreditsCheckoutReader) (Result, error) {
	status, err := s.Status(ctx, owner, id)
	if err != nil || status.State == "fulfilled" {
		return status, err
	}
	if reader == nil {
		return Result{}, ErrUnavailable
	}
	checkout, err := reader.GetCreditsCheckout(ctx, id)
	if err != nil {
		return Result{}, fmt.Errorf("verify credit purchase: %w", err)
	}
	if checkout == nil || checkout.SessionID != id || checkout.UserID != owner {
		return Result{}, ErrNotFound
	}
	if checkout.State == "expired" {
		return Result{State: "expired"}, nil
	}
	if checkout.PaymentStatus != "paid" {
		return status, nil
	}
	if checkout.State != "complete" || checkout.Purchase == nil || checkout.Purchase.CheckoutSessionID != id {
		return Result{}, ErrInvalid
	}
	if err := s.Grant(ctx, event.Envelope{UserID: owner, Provider: "stripe"}, *checkout.Purchase); err != nil {
		return Result{}, err
	}
	return s.Status(ctx, owner, id)
}

// Grant is shared by confirmed returns and verified webhooks. Aliases preserve
// event-key grants from earlier releases; the balance and ledger change atomically.
func (s *Service) Grant(ctx context.Context, env event.Envelope, purchase event.CreditsPurchased) error {
	if purchase.TotalCredits <= 0 {
		return ErrInvalid
	}
	provider := env.Provider
	if provider == "" {
		provider = "stripe"
	}
	if purchase.CheckoutSessionID == "" {
		// Compatibility for existing hosts publishing the older event shape.
		return s.credits.GrantCredits(ctx, env.UserID, provider, env.ProviderEventID, purchase.TotalCredits)
	}
	if purchase.Quantity < 1 || purchase.Quantity > 100 || purchase.CreditsPerUnit < 1 || purchase.CreditsPerUnit > math.MaxInt64/purchase.Quantity || purchase.TotalCredits != purchase.Quantity*purchase.CreditsPerUnit {
		return ErrInvalid
	}
	aliases, _, err := s.repo.related(ctx, env.UserID, provider, purchase.CheckoutSessionID)
	if err != nil {
		return err
	}
	knownEvent := false
	for _, id := range aliases {
		if id == env.ProviderEventID {
			knownEvent = true
			break
		}
	}
	if env.ProviderEventID != "" && !knownEvent {
		aliases = append(aliases, env.ProviderEventID)
	}
	_, err = s.credits.GrantCreditsWithExpiry(ctx, credit.Grant{UserID: env.UserID, Source: provider, SourceID: purchase.CheckoutSessionID, IdempotencyAliases: aliases, Amount: purchase.TotalCredits, Reason: provider + " credit grant"})
	return err
}
