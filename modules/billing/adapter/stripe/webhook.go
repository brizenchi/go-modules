package stripe

import (
	"encoding/json"
	"fmt"
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

	envs := p.translateEvent(string(ev.Type), data, ev.ID, occurredAt, hint)
	out.Envelopes = envs
	return out, nil
}

// translateEvent converts a Stripe event into zero or more domain envelopes.
func (p *Provider) translateEvent(evtType string, data map[string]any, evtID string, occurredAt time.Time, hint port.UserHint) []event.Envelope {
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
		return p.onCheckoutCompleted(data, mk)
	case "invoice.payment_succeeded", "invoice.paid":
		return p.onInvoicePaid(data, mk)
	case "invoice.payment_failed":
		return []event.Envelope{mk(event.KindPaymentFailed, event.PaymentFailed{
			ProviderSubscriptionID: getString(data, "subscription"),
			ProviderCustomerID:     getString(data, "customer"),
		})}
	case "customer.subscription.created":
		snap := p.snapshotFromMap(data)
		return []event.Envelope{mk(event.KindSubscriptionUpdated, event.SubscriptionUpdated{Snapshot: *snap})}
	case "customer.subscription.updated":
		return p.onSubscriptionUpdated(data, mk)
	case "customer.subscription.deleted":
		return []event.Envelope{mk(event.KindSubscriptionCanceled, event.SubscriptionCanceled{
			ProviderSubscriptionID: getString(data, "id"),
			ProviderCustomerID:     getString(data, "customer"),
		})}
	}
	return nil
}

func (p *Provider) onCheckoutCompleted(data map[string]any, mk func(event.Kind, any) event.Envelope) []event.Envelope {
	mode := getString(data, "mode")
	metadata := getMap(data, "metadata")
	productType := getString(metadata, "product_type")

	if mode == "payment" || productType == string(domain.ProductCredits) {
		quantity := p.extractCheckoutQuantity(data)
		creditsPerUnit := p.cfg.CreditsPerUnit
		return []event.Envelope{mk(event.KindCreditsPurchased, event.CreditsPurchased{
			Quantity:       quantity,
			CreditsPerUnit: creditsPerUnit,
			TotalCredits:   quantity * creditsPerUnit,
			PriceID:        getString(metadata, "price_id"),
		})}
	}

	// Subscription checkout — emit an Activated. The payload-derived snapshot
	// uses metadata since the session payload carries little subscription
	// info; subsequent subscription.updated events refine it.
	snap := domain.SubscriptionSnapshot{
		ProviderSubscriptionID: getString(data, "subscription"),
		ProviderCustomerID:     getString(data, "customer"),
	}
	if priceID := getString(metadata, "price_id"); priceID != "" {
		snap.ProviderPriceID = priceID
		snap.Plan, snap.Interval = p.cfg.PlanForPrice(priceID)
	}
	if snap.Plan == "" {
		snap.Plan = domain.PlanType(strings.ToLower(strings.TrimSpace(getString(metadata, "plan"))))
	}
	if snap.Interval == "" {
		snap.Interval = domain.BillingInterval(strings.ToLower(strings.TrimSpace(getString(metadata, "interval"))))
	}
	snap.Status = domain.StatusActive
	return []event.Envelope{mk(event.KindSubscriptionActivated, event.SubscriptionActivated{Snapshot: snap})}
}

func (p *Provider) onInvoicePaid(data map[string]any, mk func(event.Kind, any) event.Envelope) []event.Envelope {
	subscriptionID := getString(data, "subscription")
	if subscriptionID == "" {
		return nil
	}
	billingReason := getString(data, "billing_reason")
	// Skip first invoice — checkout.session.completed handles activation.
	if billingReason == "subscription_create" {
		return nil
	}

	snap := domain.SubscriptionSnapshot{
		ProviderSubscriptionID: subscriptionID,
		ProviderCustomerID:     getString(data, "customer"),
		Status:                 domain.StatusActive,
	}
	lines := getSlice(data, "lines", "data")
	if len(lines) > 0 {
		if line, ok := lines[0].(map[string]any); ok {
			price := getMap(line, "price")
			snap.ProviderPriceID = getString(price, "id")
			snap.ProviderProductID = getString(price, "product")
			period := getMap(line, "period")
			snap.PeriodStart = unixToTimePtr(getInt64(period, "start"))
			snap.PeriodEnd = unixToTimePtr(getInt64(period, "end"))
			if snap.ProviderPriceID != "" {
				snap.Plan, snap.Interval = p.cfg.PlanForPrice(snap.ProviderPriceID)
			}
		}
	}
	return []event.Envelope{mk(event.KindSubscriptionRenewed, event.SubscriptionRenewed{Snapshot: snap})}
}

func (p *Provider) onSubscriptionUpdated(data map[string]any, mk func(event.Kind, any) event.Envelope) []event.Envelope {
	snap := p.snapshotFromMap(data)
	cancelAtPeriodEnd := getBool(data, "cancel_at_period_end")
	cancelAt := getInt64(data, "cancel_at")

	if cancelAtPeriodEnd || cancelAt > 0 {
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

	// Reactivation: was canceling, now active again. We can't see "was",
	// so emit a Reactivated event whenever cancel flags are clear; the
	// listener should be idempotent.
	if snap.Status == domain.StatusActive {
		return []event.Envelope{mk(event.KindSubscriptionReactivated, event.SubscriptionReactivated{Snapshot: *snap})}
	}
	return []event.Envelope{mk(event.KindSubscriptionUpdated, event.SubscriptionUpdated{Snapshot: *snap})}
}

// snapshotFromMap parses a customer.subscription.* payload.
func (p *Provider) snapshotFromMap(data map[string]any) *domain.SubscriptionSnapshot {
	snap := &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: getString(data, "id"),
		ProviderCustomerID:     getString(data, "customer"),
		Status:                 normalizeStripeStatus(getString(data, "status"), getBool(data, "cancel_at_period_end")),
		CancelAtPeriodEnd:      getBool(data, "cancel_at_period_end"),
		PeriodStart:            unixToTimePtr(getInt64(data, "current_period_start")),
		PeriodEnd:              unixToTimePtr(getInt64(data, "current_period_end")),
	}
	if cancelAt := unixToTimePtr(getInt64(data, "cancel_at")); cancelAt != nil {
		snap.CancelEffectiveAt = cancelAt
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

// extractCheckoutQuantity returns the quantity from a checkout.session.completed payload.
// Falls back to the API if line_items isn't expanded in the webhook (Stripe default).
func (p *Provider) extractCheckoutQuantity(data map[string]any) int64 {
	lineItems := getMap(data, "line_items")
	if items, ok := lineItems["data"].([]any); ok && len(items) > 0 {
		if item, ok := items[0].(map[string]any); ok {
			if qty := getInt64(item, "quantity"); qty > 0 {
				return qty
			}
		}
	}
	sessionID := getString(data, "id")
	if sessionID == "" {
		return 1
	}
	params := &stripesdk.CheckoutSessionParams{}
	params.AddExpand("line_items")
	sess, err := session.Get(sessionID, params)
	if err != nil {
		return 1
	}
	if sess.LineItems != nil && len(sess.LineItems.Data) > 0 {
		return sess.LineItems.Data[0].Quantity
	}
	if md := sess.Metadata; md != nil {
		if raw, ok := md["quantity"]; ok {
			if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
				return n
			}
		}
	}
	return 1
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
func normalizeStripeStatus(stripeStatus string, cancelAtPeriodEnd bool) domain.SubscriptionStatus {
	if cancelAtPeriodEnd {
		return domain.StatusCanceling
	}
	switch strings.ToLower(strings.TrimSpace(stripeStatus)) {
	case "active":
		return domain.StatusActive
	case "trialing":
		return domain.StatusTrialing
	case "past_due":
		return domain.StatusPastDue
	case "canceled":
		return domain.StatusCanceled
	case "incomplete", "incomplete_expired":
		return domain.StatusIncomplete
	case "unpaid":
		return domain.StatusPaymentFailed
	case "":
		return domain.StatusActive
	default:
		return domain.SubscriptionStatus(stripeStatus)
	}
}
