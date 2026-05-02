package domain

import "errors"

var (
	ErrProviderDisabled         = errors.New("billing: payment provider is not enabled")
	ErrInvalidInput             = errors.New("billing: invalid input")
	ErrPlanNotFound             = errors.New("billing: plan not found")
	ErrPriceNotFound            = errors.New("billing: price not configured")
	ErrInvalidPriceID           = errors.New("billing: invalid price id")
	ErrInvalidCancelMode        = errors.New("billing: invalid cancel mode")
	ErrNoBillingCustomer        = errors.New("billing: no billing customer")
	ErrNoActiveSubscription     = errors.New("billing: no active subscription")
	ErrNoSubscriptionToReactive = errors.New("billing: no subscription to reactivate")
	ErrSignatureInvalid         = errors.New("billing: webhook signature invalid")
	ErrEventAlreadyProcessed    = errors.New("billing: event already processed")
)
