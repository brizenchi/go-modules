// Package app contains the billing module's use cases.
//
// Use cases coordinate the Provider (port.Provider), persistence
// (port.BillingEventRepository, port.CustomerStore), and the event bus
// (port.EventBus). They never touch HTTP.
package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// CheckoutService opens hosted checkout sessions.
type CheckoutService struct {
	provider     port.Provider
	customers    port.CustomerStore
	reservations port.CheckoutReservationRepository
	returnURLs   port.ReturnURLValidator
}

func NewCheckoutService(p port.Provider, c port.CustomerStore, validators ...port.ReturnURLValidator) *CheckoutService {
	reservations, _ := c.(port.CheckoutReservationRepository)
	var returnURLs port.ReturnURLValidator
	if len(validators) > 0 {
		returnURLs = validators[0]
	}
	return &CheckoutService{provider: p, customers: c, reservations: reservations, returnURLs: returnURLs}
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
	if err := validateReturnURL(s.returnURLs, in.SuccessURL); err != nil {
		return nil, err
	}
	if err := validateReturnURL(s.returnURLs, in.CancelURL); err != nil {
		return nil, err
	}
	if in.ProductType == domain.ProductCredits && (in.Quantity < 1 || in.Quantity > 100) {
		return nil, fmt.Errorf("%w: credits quantity must be between 1 and 100", domain.ErrInvalidInput)
	}
	if in.ProductType == domain.ProductCredits {
		// Credits currently have one global credits-per-unit policy. Never let a
		// caller choose among configured Stripe SKUs; the provider selects the
		// canonical server-side offer.
		in.PriceID = ""
	}
	if in.ProductType != domain.ProductCredits {
		in.Quantity = 1
	}

	cust, err := s.customers.LoadCustomer(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	// The authenticated account is the source of truth. Allowing the request
	// body to override this address can bind a newly-created provider customer
	// to an attacker-controlled email.
	in.Email = strings.TrimSpace(cust.Email)
	if in.Email == "" {
		return nil, fmt.Errorf("%w: email required", domain.ErrInvalidInput)
	}
	if (in.ProductType == domain.ProductSubscription || in.ProductType == domain.ProductLifetime) && hasConflictingPaidEntitlement(cust) {
		return nil, domain.ErrSubscriptionCheckoutConflict
	}

	// Prevent duplicate free trials: if the user has ever had a subscription,
	// strip trial days so they go straight to paid.
	if in.ProductType == domain.ProductSubscription && in.TrialDays >= 0 {
		used, err := s.customers.HasUsedTrial(ctx, in.UserID)
		if err != nil {
			return nil, err
		}
		if used {
			in.TrialDays = -1 // signal provider to skip trial
		}
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

	if in.ProductType != domain.ProductSubscription && in.ProductType != domain.ProductLifetime {
		return s.provider.CreateCheckout(ctx, in)
	}
	if s.reservations == nil {
		return nil, fmt.Errorf("billing: checkout reservation repository required")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Hour)
	intentKey := checkoutIntentKey(in)
	reservationID, err := newCheckoutReservationID()
	if err != nil {
		return nil, err
	}
	reservation, acquired, err := s.reservations.ReserveCheckout(
		ctx, in.UserID, s.provider.Name(), customerID, reservationID, intentKey, now, expiresAt,
	)
	if err != nil {
		return nil, err
	}
	if !acquired {
		if reservation == nil {
			return nil, domain.ErrSubscriptionCheckoutConflict
		}
		if strings.TrimSpace(reservation.SessionID) == "" {
			if reservation.ExpiresAt.After(now) {
				if reservation.IntentKey != intentKey {
					return nil, domain.ErrSubscriptionCheckoutConflict
				}
				// The provider response or the DB completion write may have been lost.
				// Retrying the same intent with the original reservation ID reuses the
				// exact Stripe idempotency key and cannot create a second Session.
				reservationID = reservation.ReservationID
				expiresAt = reservation.ExpiresAt
			} else {
				recoveryCustomerID := strings.TrimSpace(reservation.ProviderCustomerID)
				if recoveryCustomerID == "" {
					// Rows created before provider_customer_id was introduced can be
					// recovered through the currently verified customer binding.
					recoveryCustomerID = customerID
				}
				recovered, err := s.provider.FindCheckoutSession(ctx, recoveryCustomerID, reservation.ReservationID)
				if err != nil {
					return nil, fmt.Errorf("billing: recover reserved checkout session: %w", err)
				}
				if recovered != nil {
					reservation.SessionID = recovered.SessionID
					reservation.CheckoutURL = recovered.CheckoutURL
					reservation.ProviderSubscriptionID = recovered.ProviderSubscriptionID
					if err := s.reservations.CompleteCheckoutReservation(ctx, in.UserID, s.provider.Name(), reservation.ReservationID, domain.CheckoutResult{
						SessionID: recovered.SessionID, CheckoutURL: recovered.CheckoutURL,
					}); err != nil {
						return nil, err
					}
					if err := s.reservations.LinkCheckoutSubscription(ctx, s.provider.Name(), recovered.SessionID, recovered.ProviderSubscriptionID); err != nil {
						return nil, err
					}
					if recovered.State == domain.CheckoutSessionOpen && reservation.IntentKey == intentKey {
						return reusableCheckoutResult(reservation, recovered)
					}
				}
				canReplace := recovered == nil || recovered.State == domain.CheckoutSessionExpired || recovered.State == domain.CheckoutSessionFailed
				if recovered != nil && recovered.State == domain.CheckoutSessionComplete && terminalEntitlementMatchesReservation(cust, reservation) {
					canReplace = true
				}
				if !canReplace {
					return nil, domain.ErrSubscriptionCheckoutConflict
				}
				replaced, err := s.reservations.ReplaceCheckoutReservation(
					ctx, in.UserID, s.provider.Name(), customerID, reservation.ReservationID,
					reservationID, intentKey, expiresAt,
				)
				if err != nil {
					return nil, err
				}
				if !replaced {
					return nil, domain.ErrSubscriptionCheckoutConflict
				}
			}
		} else {
			providerSession, err := s.provider.GetCheckoutSession(ctx, reservation.SessionID)
			if err != nil {
				// Fail closed: an elapsed local expires_at is not proof that Stripe did
				// not complete the Session just before webhook delivery was delayed.
				return nil, fmt.Errorf("billing: verify reserved checkout session: %w", err)
			}
			if providerSession != nil && strings.TrimSpace(providerSession.ProviderSubscriptionID) != "" {
				reservation.ProviderSubscriptionID = providerSession.ProviderSubscriptionID
				if err := s.reservations.LinkCheckoutSubscription(ctx, s.provider.Name(), reservation.SessionID, providerSession.ProviderSubscriptionID); err != nil {
					return nil, err
				}
			}
			if providerSession != nil && providerSession.State == domain.CheckoutSessionOpen && reservation.IntentKey == intentKey {
				return reusableCheckoutResult(reservation, providerSession)
			}
			canReplace := providerSession != nil && (providerSession.State == domain.CheckoutSessionExpired || providerSession.State == domain.CheckoutSessionFailed)
			if providerSession != nil && providerSession.State == domain.CheckoutSessionComplete && terminalEntitlementMatchesReservation(cust, reservation) {
				canReplace = true
			}
			if !canReplace {
				// Both open and completed Sessions remain conflicting unless the
				// completed Session's resulting subscription is provably terminal.
				return nil, domain.ErrSubscriptionCheckoutConflict
			}
			replaced, err := s.reservations.ReplaceCheckoutReservation(
				ctx, in.UserID, s.provider.Name(), customerID, reservation.ReservationID,
				reservationID, intentKey, expiresAt,
			)
			if err != nil {
				return nil, err
			}
			if !replaced {
				// Another request or a webhook changed/released the exact reservation
				// that we verified. Never act on stale proof.
				return nil, domain.ErrSubscriptionCheckoutConflict
			}
		}
	}
	in.CheckoutReservationID = reservationID
	in.CheckoutExpiresAt = expiresAt
	result, err := s.provider.CreateCheckout(ctx, in)
	if err != nil {
		// Keep the reservation: a network error can happen after Stripe created
		// the Session. An exact retry is safe; a different intent stays blocked
		// until that provider-side Session expires.
		return nil, err
	}
	if err := s.reservations.CompleteCheckoutReservation(ctx, in.UserID, s.provider.Name(), reservationID, *result); err != nil {
		return nil, err
	}
	return result, nil
}

func reusableCheckoutResult(reservation *domain.BillingCheckoutReservation, providerSession *domain.CheckoutSessionSnapshot) (*CheckoutResult, error) {
	sessionID := strings.TrimSpace(providerSession.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(reservation.SessionID)
	}
	checkoutURL := strings.TrimSpace(providerSession.CheckoutURL)
	if checkoutURL == "" {
		checkoutURL = strings.TrimSpace(reservation.CheckoutURL)
	}
	if sessionID == "" || checkoutURL == "" {
		return nil, fmt.Errorf("billing: reusable checkout session is missing its id or URL")
	}
	return &CheckoutResult{SessionID: sessionID, CheckoutURL: checkoutURL}, nil
}

func terminalEntitlementMatchesReservation(customer port.Customer, reservation *domain.BillingCheckoutReservation) bool {
	if reservation == nil || strings.TrimSpace(customer.ProviderSubscriptionID) == "" ||
		strings.TrimSpace(reservation.ProviderSubscriptionID) == "" ||
		customer.ProviderSubscriptionID != reservation.ProviderSubscriptionID {
		return false
	}
	switch customer.SubscriptionStatus {
	case domain.StatusCanceled, domain.StatusIncompleteExpired:
		return true
	default:
		return false
	}
}

func checkoutIntentKey(in CheckoutInput) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(in.UserID),
		strings.TrimSpace(in.Email),
		string(in.ProductType),
		string(in.Plan),
		string(in.Interval),
		fmt.Sprintf("%d", in.TrialDays),
		strings.TrimSpace(in.SuccessURL),
		strings.TrimSpace(in.CancelURL),
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func newCheckoutReservationID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("billing: generate checkout reservation: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func hasConflictingPaidEntitlement(customer port.Customer) bool {
	if strings.EqualFold(strings.TrimSpace(customer.Plan), string(domain.PlanLifetime)) {
		return true
	}
	if strings.TrimSpace(customer.ProviderSubscriptionID) == "" {
		return false
	}
	switch customer.SubscriptionStatus {
	case domain.StatusCanceled, domain.StatusIncompleteExpired:
		return false
	default:
		// Unknown and legacy blank statuses are treated conservatively: a known
		// provider subscription ID may still produce recurring charges.
		return true
	}
}
