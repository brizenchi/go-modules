package stripe

import (
	"context"
	"fmt"
	"strings"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	stripesdk "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/subscriptionschedule"
)

// RetireSubscriptionForLifetime stops the superseded recurring contract after a
// verified Lifetime payment, without refunding or invoicing unused time.
func (p *Provider) RetireSubscriptionForLifetime(ctx context.Context, subscriptionID, customerID string) error {
	if !p.Enabled() {
		return domain.ErrProviderDisabled
	}
	if strings.TrimSpace(subscriptionID) == "" || strings.TrimSpace(customerID) == "" {
		return domain.ErrInvalidInput
	}
	current, err := subscription.Get(subscriptionID, &stripesdk.SubscriptionParams{Params: stripesdk.Params{Context: ctx}})
	if err != nil {
		return fmt.Errorf("stripe: read subscription for lifetime replacement: %w", err)
	}
	if current == nil || current.Customer == nil || current.Customer.ID != customerID {
		return fmt.Errorf("%w: lifetime subscription customer mismatch", domain.ErrInvalidInput)
	}
	if current.Status == stripesdk.SubscriptionStatusCanceled || current.Status == stripesdk.SubscriptionStatusIncompleteExpired {
		return nil
	}
	if current.PendingUpdate != nil {
		if current.LatestInvoice == nil || current.LatestInvoice.ID == "" {
			return fmt.Errorf("stripe: pending subscription update has no invoice")
		}
		if _, err := invoice.VoidInvoice(current.LatestInvoice.ID, &stripesdk.InvoiceVoidInvoiceParams{Params: stripesdk.Params{Context: ctx}}); err != nil {
			return fmt.Errorf("stripe: void superseded subscription change: %w", err)
		}
	}
	if current.Schedule != nil && current.Schedule.ID != "" {
		if _, err := subscriptionschedule.Release(current.Schedule.ID, &stripesdk.SubscriptionScheduleReleaseParams{Params: stripesdk.Params{Context: ctx}}); err != nil {
			return fmt.Errorf("stripe: release superseded subscription schedule: %w", err)
		}
	}
	retired, err := subscription.Cancel(subscriptionID, &stripesdk.SubscriptionCancelParams{
		Params: stripesdk.Params{Context: ctx}, InvoiceNow: stripesdk.Bool(false), Prorate: stripesdk.Bool(false),
	})
	if err != nil {
		return fmt.Errorf("stripe: retire subscription for lifetime: %w", err)
	}
	if retired == nil || retired.Status != stripesdk.SubscriptionStatusCanceled {
		return fmt.Errorf("stripe: lifetime subscription retirement unconfirmed")
	}
	return nil
}

var _ port.LifetimeSubscriptionRetirer = (*Provider)(nil)
