package stripe

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
	stripesdk "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

// VerifyAndParseWebhook validates the signature and produces domain events.
//
// Returns nil envelopes for events the adapter does not translate. The caller
// (application layer) is still responsible for persisting the raw event for
// idempotency and audit.
func (p *Provider) VerifyAndParseWebhook(payload []byte, signature string) (*port.WebhookParseResult, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}

	ev, err := webhook.ConstructEventWithOptions(payload, signature, p.cfg.WebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrSignatureInvalid, err)
	}

	var data map[string]any
	if err := json.Unmarshal(ev.Data.Raw, &data); err != nil {
		return nil, fmt.Errorf("stripe: parse event data: %w", err)
	}

	occurredAt := time.Unix(ev.Created, 0).UTC()
	hint := extractUserHint(data, string(ev.Type))

	out := &port.WebhookParseResult{
		ProviderEventID: ev.ID,
		Type:            string(ev.Type),
		UserHint:        hint,
		RawPayload:      append([]byte(nil), payload...),
	}
	if strings.HasPrefix(string(ev.Type), "checkout.session.") {
		out.CheckoutSessionID = getString(data, "id")
		out.CheckoutReservationID = getString(getMap(data, "metadata"), "checkout_reservation_id")
		out.CheckoutSubscriptionID = getString(data, "subscription")
		out.ReleaseCheckoutReservation = string(ev.Type) == "checkout.session.async_payment_failed" || string(ev.Type) == "checkout.session.expired"
	}

	envs, err := p.translateEvent(string(ev.Type), data, ev.Data.PreviousAttributes, ev.ID, occurredAt, hint)
	if err != nil {
		return nil, err
	}
	out.Envelopes = envs
	return out, nil
}

// translateEvent converts a Stripe event into zero or more domain envelopes.
func (p *Provider) translateEvent(evtType string, data map[string]any, prevAttrs map[string]any, evtID string, occurredAt time.Time, hint port.UserHint) ([]event.Envelope, error) {
	mk := func(kind event.Kind, payload any) event.Envelope {
		return event.Envelope{
			Kind:            kind,
			UserID:          hint.UserID, // app layer fills this in if empty via UserResolver
			Provider:        "stripe",
			ProviderEventID: evtID,
			OccurredAt:      occurredAt,
			Payload:         payload,
		}
	}

	switch evtType {
	case "checkout.session.completed":
		return p.onCheckoutCompleted(data, false, mk)
	case "checkout.session.async_payment_succeeded":
		return p.onCheckoutCompleted(data, true, mk)
	case "checkout.session.async_payment_failed":
		// No durable entitlement has been granted while the Session was unpaid,
		// so an asynchronous failure has nothing to revoke.
		return nil, nil
	case "checkout.session.expired":
		return nil, nil
	case "invoice.paid":
		return p.onInvoicePaid(data, mk), nil
	case "invoice.payment_failed":
		return []event.Envelope{mk(event.KindPaymentFailed, event.PaymentFailed{
			ProviderSubscriptionID: getString(data, "subscription"),
			ProviderCustomerID:     getString(data, "customer"),
		})}, nil
	case "customer.subscription.created":
		snap := p.snapshotFromMap(data)
		return []event.Envelope{mk(event.KindSubscriptionUpdated, event.SubscriptionUpdated{Snapshot: *snap})}, nil
	case "customer.subscription.updated":
		return p.onSubscriptionUpdated(data, prevAttrs, mk), nil
	case "customer.subscription.deleted":
		snap := p.snapshotFromMap(data)
		snap.Status = domain.StatusCanceled
		return []event.Envelope{mk(event.KindSubscriptionCanceled, event.SubscriptionCanceled{
			ProviderSubscriptionID: getString(data, "id"),
			ProviderCustomerID:     getString(data, "customer"),
			Snapshot:               *snap,
		})}, nil
	}
	return nil, nil
}

func (p *Provider) onCheckoutCompleted(data map[string]any, asyncPaymentSucceeded bool, mk func(event.Kind, any) event.Envelope) ([]event.Envelope, error) {
	mode := getString(data, "mode")
	metadata := getMap(data, "metadata")
	productType := getString(metadata, "product_type")
	paymentStatus := strings.ToLower(strings.TrimSpace(getString(data, "payment_status")))
	trialDays, _ := strconv.ParseInt(strings.TrimSpace(getString(metadata, "trial_days")), 10, 64)
	isUnpaidTrialStart := domain.ProductType(productType) == domain.ProductSubscription && trialDays > 0 && paymentStatus == "no_payment_required"
	if !asyncPaymentSucceeded && paymentStatus != "paid" && !isUnpaidTrialStart {
		// Checkout completion is not proof that funds settled. This applies to
		// one-time purchases and to the first invoice of non-trial subscriptions.
		// Trial Checkouts report no_payment_required and continue below as trialing.
		return nil, nil
	}

	priceID := strings.TrimSpace(getString(metadata, "price_id"))
	switch domain.ProductType(productType) {
	case domain.ProductLifetime:
		if mode != "payment" {
			return nil, fmt.Errorf("%w: lifetime checkout must use payment mode", domain.ErrInvalidInput)
		}
		if priceID == "" || priceID != strings.TrimSpace(p.cfg.LifetimePriceID) {
			return nil, fmt.Errorf("%w: unconfigured lifetime price", domain.ErrInvalidPriceID)
		}
		return []event.Envelope{mk(event.KindSubscriptionActivated, event.SubscriptionActivated{
			Snapshot: domain.SubscriptionSnapshot{
				ProviderCustomerID: getString(data, "customer"),
				ProviderPriceID:    priceID,
				ProductType:        domain.ProductLifetime,
				Plan:               domain.PlanLifetime,
				Status:             domain.StatusActive,
			},
		})}, nil
	case domain.ProductCredits:
		if mode != "payment" {
			return nil, fmt.Errorf("%w: credits checkout must use payment mode", domain.ErrInvalidInput)
		}
		if !p.cfg.IsCreditsPriceID(priceID) {
			return nil, fmt.Errorf("%w: unconfigured credits price", domain.ErrInvalidPriceID)
		}
		quantity, err := p.extractCheckoutQuantity(data, priceID)
		if err != nil {
			return nil, err
		}
		if quantity < 1 || quantity > 100 {
			return nil, fmt.Errorf("%w: credits quantity must be between 1 and 100", domain.ErrInvalidInput)
		}
		creditsPerUnit := p.cfg.CreditsPerUnit
		if creditsPerUnit <= 0 || creditsPerUnit > math.MaxInt64/quantity {
			return nil, fmt.Errorf("%w: invalid credits_per_unit", domain.ErrInvalidInput)
		}
		return []event.Envelope{mk(event.KindCreditsPurchased, event.CreditsPurchased{
			Quantity:       quantity,
			CreditsPerUnit: creditsPerUnit,
			TotalCredits:   quantity * creditsPerUnit,
			PriceID:        priceID,
		})}, nil
	case domain.ProductSubscription:
		if mode != "subscription" {
			return nil, fmt.Errorf("%w: subscription checkout must use subscription mode", domain.ErrInvalidInput)
		}
	default:
		// Other Checkout Sessions may legitimately share the Stripe account.
		// Without billing-owned product metadata they must never grant an
		// entitlement in this module.
		return nil, nil
	}

	// Subscription checkout — emit an Activated. The payload-derived snapshot
	// uses the server-owned price mapping since the session payload carries little
	// subscription info; subsequent subscription.updated events refine it.
	snap := domain.SubscriptionSnapshot{
		ProviderSubscriptionID: getString(data, "subscription"),
		ProviderCustomerID:     getString(data, "customer"),
		ProviderPriceID:        priceID,
		ProductType:            domain.ProductSubscription,
	}
	snap.Plan, snap.Interval = p.cfg.PlanForPrice(priceID)
	if priceID == "" || snap.Plan == domain.PlanFree || !snap.Interval.Valid() {
		return nil, fmt.Errorf("%w: unconfigured subscription price", domain.ErrInvalidPriceID)
	}
	snap.Status = domain.StatusActive
	if trialDays > 0 {
		snap.Status = domain.StatusTrialing
	}
	return []event.Envelope{mk(event.KindSubscriptionActivated, event.SubscriptionActivated{Snapshot: snap})}, nil
}

func (p *Provider) onInvoicePaid(data map[string]any, mk func(event.Kind, any) event.Envelope) []event.Envelope {
	subscriptionID := getString(data, "subscription")
	if subscriptionID == "" {
		return nil
	}
	billingReason := getString(data, "billing_reason")
	// Only a cycle invoice is a renewal. Checkout handles subscription_create;
	// prorations and manual subscription_update invoices must not replenish
	// renewal quotas or fire renewal notifications.
	if billingReason != "subscription_cycle" {
		return nil
	}

	snap := domain.SubscriptionSnapshot{
		ProviderSubscriptionID: subscriptionID,
		ProviderCustomerID:     getString(data, "customer"),
		Status:                 domain.StatusActive,
	}
	lines := getSlice(data, "lines", "data")
	if len(lines) == 0 {
		return nil
	}
	line, ok := lines[0].(map[string]any)
	if !ok {
		return nil
	}
	price := getMap(line, "price")
	snap.ProviderPriceID = getString(price, "id")
	snap.ProviderProductID = getString(price, "product")
	snap.Plan, snap.Interval = p.cfg.PlanForPrice(snap.ProviderPriceID)
	if snap.ProviderPriceID == "" || snap.Plan == domain.PlanFree || !snap.Interval.Valid() {
		// Invoice lines are not an authoritative subscription object. An unknown
		// or incomplete line must not publish/persist a snapshot that clears the
		// canonical paid plan and interval.
		return nil
	}
	period := getMap(line, "period")
	snap.PeriodStart = unixToTimePtr(getInt64(period, "start"))
	snap.PeriodEnd = unixToTimePtr(getInt64(period, "end"))
	return []event.Envelope{mk(event.KindSubscriptionRenewed, event.SubscriptionRenewed{
		Snapshot:   snap,
		AmountPaid: getInt64(data, "amount_paid"),
		Currency:   strings.ToLower(strings.TrimSpace(getString(data, "currency"))),
	})}
}

func (p *Provider) onSubscriptionUpdated(data map[string]any, prevAttrs map[string]any, mk func(event.Kind, any) event.Envelope) []event.Envelope {
	snap := p.snapshotFromMap(data)
	cancelAtPeriodEnd := getBool(data, "cancel_at_period_end")
	cancelAt := getInt64(data, "cancel_at")

	if (cancelAtPeriodEnd || cancelAt > 0) && snap.Status != domain.StatusCanceled && snap.Status != domain.StatusIncompleteExpired {
		var mode domain.CancelMode
		if cancelAt > 0 {
			mode = domain.CancelIn3Days
		} else {
			mode = domain.CancelAtPeriodEnd
		}
		return []event.Envelope{mk(event.KindSubscriptionCanceling, event.SubscriptionCanceling{
			Snapshot:    *snap,
			Mode:        mode,
			EffectiveAt: snap.CancelEffectiveAt,
		})}
	}

	// Reactivation requires proof that a cancellation flag was cleared. An
	// ordinary active subscription update is not a reactivation.
	wasCanceling := getBool(prevAttrs, "cancel_at_period_end") || getInt64(prevAttrs, "cancel_at") > 0
	if snap.Status == domain.StatusActive && wasCanceling {
		return []event.Envelope{mk(event.KindSubscriptionReactivated, event.SubscriptionReactivated{Snapshot: *snap})}
	}
	return []event.Envelope{mk(event.KindSubscriptionUpdated, event.SubscriptionUpdated{Snapshot: *snap})}
}

// snapshotFromMap parses a customer.subscription.* payload.
func (p *Provider) snapshotFromMap(data map[string]any) *domain.SubscriptionSnapshot {
	cancelAtPeriodEnd := getBool(data, "cancel_at_period_end")
	cancelAt := getInt64(data, "cancel_at")
	snap := &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: getString(data, "id"),
		ProviderCustomerID:     getString(data, "customer"),
		Status:                 normalizeStripeStatus(getString(data, "status"), cancelAtPeriodEnd || cancelAt > 0),
		CancelAtPeriodEnd:      cancelAtPeriodEnd,
		PeriodStart:            unixToTimePtr(getInt64(data, "current_period_start")),
		PeriodEnd:              unixToTimePtr(getInt64(data, "current_period_end")),
	}
	if cancelAtTime := unixToTimePtr(cancelAt); cancelAtTime != nil {
		snap.CancelEffectiveAt = cancelAtTime
	} else if snap.CancelAtPeriodEnd && snap.PeriodEnd != nil {
		snap.CancelEffectiveAt = snap.PeriodEnd
	}
	items := getSlice(data, "items", "data")
	if len(items) > 0 {
		if item, ok := items[0].(map[string]any); ok {
			price := getMap(item, "price")
			snap.ProviderPriceID = getString(price, "id")
			snap.ProviderProductID = getString(price, "product")
			if snap.ProviderPriceID != "" {
				snap.Plan, snap.Interval = p.cfg.PlanForPrice(snap.ProviderPriceID)
			}
		}
	}
	return snap
}

// extractCheckoutQuantity returns the authoritative quantity and price from
// Stripe's line items. Checkout metadata contains the quantity requested when
// the Session was created, but adjustable quantity can change it before
// payment, so metadata must never be used for fulfillment.
func (p *Provider) extractCheckoutQuantity(data map[string]any, expectedPriceID string) (int64, error) {
	if rawLineItems, expanded := data["line_items"]; expanded && rawLineItems != nil {
		lineItems, ok := rawLineItems.(map[string]any)
		if !ok {
			return 0, fmt.Errorf("stripe: checkout session has malformed line items")
		}
		items, ok := lineItems["data"].([]any)
		if !ok || len(items) != 1 {
			return 0, fmt.Errorf("stripe: checkout session must contain exactly one credits line item")
		}
		item, ok := items[0].(map[string]any)
		if !ok || item == nil {
			return 0, fmt.Errorf("stripe: checkout session has malformed credits line item")
		}
		return validateCheckoutLineItem(getInt64(item, "quantity"), checkoutLineItemPriceID(item), expectedPriceID)
	}

	sessionID := getString(data, "id")
	if sessionID == "" {
		return 0, fmt.Errorf("stripe: checkout session id required to resolve credits quantity")
	}
	params := &stripesdk.CheckoutSessionParams{}
	params.AddExpand("line_items")
	sess, err := session.Get(sessionID, params)
	if err != nil {
		return 0, fmt.Errorf("stripe: get checkout session line items: %w", err)
	}
	return checkoutQuantityFromProviderSession(sess, expectedPriceID)
}

func checkoutLineItemPriceID(item map[string]any) string {
	switch price := item["price"].(type) {
	case map[string]any:
		return strings.TrimSpace(getString(price, "id"))
	case string:
		return strings.TrimSpace(price)
	default:
		return ""
	}
}

func checkoutQuantityFromProviderSession(sess *stripesdk.CheckoutSession, expectedPriceID string) (int64, error) {
	if sess == nil || sess.LineItems == nil || len(sess.LineItems.Data) != 1 || sess.LineItems.Data[0] == nil {
		return 0, fmt.Errorf("stripe: checkout session missing authoritative credits line item")
	}
	item := sess.LineItems.Data[0]
	actualPriceID := ""
	if item.Price != nil {
		actualPriceID = item.Price.ID
	}
	return validateCheckoutLineItem(item.Quantity, actualPriceID, expectedPriceID)
}

func validateCheckoutLineItem(quantity int64, actualPriceID, expectedPriceID string) (int64, error) {
	actualPriceID = strings.TrimSpace(actualPriceID)
	expectedPriceID = strings.TrimSpace(expectedPriceID)
	if expectedPriceID == "" || actualPriceID == "" || actualPriceID != expectedPriceID {
		return 0, fmt.Errorf("%w: checkout line item price does not match credits offer", domain.ErrInvalidPriceID)
	}
	if quantity <= 0 {
		return 0, fmt.Errorf("%w: checkout line item quantity must be positive", domain.ErrInvalidInput)
	}
	return quantity, nil
}

// extractUserHint pulls every identifier we might use to resolve a user.
func extractUserHint(data map[string]any, evtType string) port.UserHint {
	hint := port.UserHint{}
	metadata := getMap(data, "metadata")
	hint.UserID = strings.TrimSpace(getString(metadata, "user_id"))
	if hint.UserID == "" {
		hint.UserID = strings.TrimSpace(getString(data, "client_reference_id"))
	}
	hint.Email = strings.TrimSpace(getString(metadata, "email"))
	if hint.Email == "" {
		hint.Email = strings.TrimSpace(getString(getMap(data, "customer_details"), "email"))
	}
	if hint.Email == "" {
		hint.Email = strings.TrimSpace(getString(data, "receipt_email"))
	}
	hint.ProviderCustomerID = strings.TrimSpace(getString(data, "customer"))
	hint.ProviderSubscriptionID = strings.TrimSpace(getString(data, "subscription"))
	if hint.ProviderSubscriptionID == "" && strings.HasPrefix(evtType, "customer.subscription.") {
		hint.ProviderSubscriptionID = strings.TrimSpace(getString(data, "id"))
	}
	return hint
}

// normalizeStripeStatus maps Stripe's subscription status to our enum.
func normalizeStripeStatus(stripeStatus string, cancellationPending bool) domain.SubscriptionStatus {
	normalized := strings.ToLower(strings.TrimSpace(stripeStatus))
	// Terminal provider states always win over a stale/historical cancel_at.
	switch normalized {
	case "canceled":
		return domain.StatusCanceled
	case "incomplete_expired":
		return domain.StatusIncompleteExpired
	}
	if cancellationPending {
		return domain.StatusCanceling
	}
	switch normalized {
	case "active":
		return domain.StatusActive
	case "trialing":
		return domain.StatusTrialing
	case "past_due":
		return domain.StatusPastDue
	case "incomplete":
		return domain.StatusIncomplete
	case "unpaid":
		return domain.StatusPaymentFailed
	case "":
		return domain.StatusActive
	default:
		return domain.SubscriptionStatus(stripeStatus)
	}
}
