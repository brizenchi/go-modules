package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// QueryService exposes read-only billing data to the host application.
type QueryService struct {
	provider  port.Provider
	customers port.CustomerStore
}

func NewQueryService(p port.Provider, c port.CustomerStore) *QueryService {
	return &QueryService{provider: p, customers: c}
}

// SubscriptionView is a flattened view for HTTP responses.
type SubscriptionView struct {
	Plan              domain.PlanType           `json:"plan"`
	Status            domain.SubscriptionStatus `json:"status"`
	Interval          domain.BillingInterval    `json:"billing_cycle"`
	CurrentPeriodEnd  string                    `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd bool                      `json:"cancel_at_period_end"`
	CancelEffectiveAt string                    `json:"effective_cancel_at,omitempty"`
	PaymentMethod     *domain.PaymentMethodCard `json:"payment_method,omitempty"`
}

// GetSubscription returns the current snapshot for a user.
func (s *QueryService) GetSubscription(ctx context.Context, userID string) (*SubscriptionView, error) {
	cust, err := s.customers.LoadCustomer(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}

	view := &SubscriptionView{Plan: domain.PlanFree, Status: domain.SubscriptionStatus("inactive")}
	if cust.ProviderSubscriptionID != "" {
		if subscriptionStatusEnded(cust.SubscriptionStatus) {
			// Stripe may no longer return a deleted/expired subscription. The
			// canonical terminal snapshot is sufficient to render the page and let
			// the user start a new Checkout instead of turning that state into 500.
			view.Plan = domain.PlanType(strings.ToLower(strings.TrimSpace(cust.Plan)))
			view.Status = cust.SubscriptionStatus
		} else {
			snap, err := s.provider.GetSubscription(ctx, cust.ProviderSubscriptionID)
			if err != nil {
				return nil, fmt.Errorf("billing: refresh subscription from provider: %w", err)
			}
			if snap == nil {
				return nil, fmt.Errorf("billing: provider returned no snapshot for subscription %q", cust.ProviderSubscriptionID)
			}
			view.Plan = snap.Plan
			view.Status = snap.Status
			view.Interval = snap.Interval
			view.CancelAtPeriodEnd = snap.CancelAtPeriodEnd
			if snap.CancelEffectiveAt != nil {
				view.CancelEffectiveAt = snap.CancelEffectiveAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			if snap.PeriodEnd != nil {
				view.CurrentPeriodEnd = snap.PeriodEnd.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
		}
	} else if customerLifetimePlan(cust) {
		view.Plan = domain.PlanLifetime
		view.Status = domain.StatusActive
	}
	if cust.ProviderCustomerID != "" {
		if card, err := s.provider.GetDefaultPaymentMethod(ctx, cust.ProviderCustomerID); err == nil {
			view.PaymentMethod = card
		}
	}
	if view.Plan == "" {
		view.Plan = domain.PlanFree
	}
	return view, nil
}

func subscriptionStatusEnded(status domain.SubscriptionStatus) bool {
	return status == domain.StatusCanceled || status == domain.StatusIncompleteExpired
}

func customerLifetimePlan(cust port.Customer) bool {
	return strings.EqualFold(strings.TrimSpace(cust.Plan), string(domain.PlanLifetime))
}

// ListInvoices returns paginated invoices.
func (s *QueryService) ListInvoices(ctx context.Context, userID string, page, limit int) ([]domain.InvoiceItem, int, error) {
	cust, err := s.customers.LoadCustomer(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, 0, err
	}
	if cust.ProviderCustomerID == "" {
		return []domain.InvoiceItem{}, 0, nil
	}
	return s.provider.ListInvoices(ctx, cust.ProviderCustomerID, page, limit)
}

// PreviewSubscriptionChange returns a commercial preview for the requested switch.
func (s *QueryService) PreviewSubscriptionChange(ctx context.Context, userID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error) {
	cust, err := s.customers.LoadCustomer(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	return s.provider.PreviewSubscriptionChange(ctx, cust.ProviderCustomerID, cust.ProviderSubscriptionID, in)
}
