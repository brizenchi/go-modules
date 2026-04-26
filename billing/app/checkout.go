// Package app contains the billing module's use cases.
//
// Use cases coordinate the Provider (port.Provider), persistence
// (port.BillingEventRepository, port.CustomerStore), and the event bus
// (port.EventBus). They never touch HTTP.
package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/billing/domain"
	"github.com/brizenchi/go-modules/billing/port"
)

// CheckoutService opens hosted checkout sessions.
type CheckoutService struct {
	provider  port.Provider
	customers port.CustomerStore
}

func NewCheckoutService(p port.Provider, c port.CustomerStore) *CheckoutService {
	return &CheckoutService{provider: p, customers: c}
}

// CheckoutInput mirrors domain.CheckoutInput for API stability.
type CheckoutInput = domain.CheckoutInput

// CheckoutResult mirrors domain.CheckoutResult.
type CheckoutResult = domain.CheckoutResult

// Create opens a checkout session, ensuring the user has a provider customer ID.
func (s *CheckoutService) Create(ctx context.Context, in CheckoutInput) (*CheckoutResult, error) {
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID == "" {
		return nil, fmt.Errorf("%w: user_id required", domain.ErrInvalidInput)
	}
	if in.SuccessURL == "" || in.CancelURL == "" {
		return nil, fmt.Errorf("%w: success_url and cancel_url required", domain.ErrInvalidInput)
	}

	cust, err := s.customers.LoadCustomer(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if in.Email == "" {
		in.Email = cust.Email
	}
	if in.Email == "" {
		return nil, fmt.Errorf("%w: email required", domain.ErrInvalidInput)
	}

	customerID, err := s.provider.EnsureCustomer(ctx, in.UserID, in.Email, cust.ProviderCustomerID)
	if err != nil {
		return nil, err
	}
	if customerID != cust.ProviderCustomerID {
		if err := s.customers.SaveCustomerID(ctx, in.UserID, s.provider.Name(), customerID); err != nil {
			return nil, err
		}
	}
	in.CustomerID = customerID
	return s.provider.CreateCheckout(ctx, in)
}
