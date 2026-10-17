package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
	stripesdk "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
)

// GetCreditsCheckout shares webhook validation with payment-return confirmation.
func (p *Provider) GetCreditsCheckout(ctx context.Context, id string) (*port.CreditsCheckout, error) {
	if !p.Enabled() {
		return nil, domain.ErrProviderDisabled
	}
	if strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	params := &stripesdk.CheckoutSessionParams{Params: stripesdk.Params{Context: ctx}}
	params.AddExpand("line_items")
	sess, err := session.Get(id, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: read credits checkout: %w", err)
	}
	if sess == nil || sess.ID != id || sess.Mode != stripesdk.CheckoutSessionModePayment || sess.Metadata["product_type"] != string(domain.ProductCredits) {
		return nil, domain.ErrInvalidInput
	}
	owner := strings.TrimSpace(sess.Metadata["user_id"])
	if owner == "" {
		owner = strings.TrimSpace(sess.ClientReferenceID)
	}
	if owner == "" || (sess.ClientReferenceID != "" && sess.ClientReferenceID != owner) {
		return nil, domain.ErrInvalidInput
	}
	result := &port.CreditsCheckout{SessionID: sess.ID, UserID: owner, State: string(sess.Status), PaymentStatus: string(sess.PaymentStatus)}
	if sess.PaymentStatus != stripesdk.CheckoutSessionPaymentStatusPaid {
		return result, nil
	}
	if sess.Status != stripesdk.CheckoutSessionStatusComplete || sess.LineItems == nil {
		return nil, domain.ErrInvalidInput
	}
	raw, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("stripe: encode credits checkout: %w", err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, fmt.Errorf("stripe: decode credits checkout: %w", err)
	}
	envelopes, err := p.onCheckoutCompletedContext(ctx, object, false, func(kind event.Kind, payload any) event.Envelope { return event.Envelope{Kind: kind, Payload: payload} })
	if err != nil {
		return nil, err
	}
	if len(envelopes) != 1 {
		return nil, domain.ErrInvalidInput
	}
	purchase, ok := envelopes[0].Payload.(event.CreditsPurchased)
	if !ok {
		return nil, domain.ErrInvalidInput
	}
	result.Purchase = &purchase
	return result, nil
}

var _ port.CreditsCheckoutReader = (*Provider)(nil)
