package stripe

import (
	"context"
	"fmt"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	stripesdk "github.com/stripe/stripe-go/v76"
	providerevent "github.com/stripe/stripe-go/v76/event"
)

// ListRecoveryEvents returns one bounded page, newest first. The caller must
// persist the cursor and fixed time window until all pages are consumed.
func (p *Provider) ListRecoveryEvents(ctx context.Context, from, to time.Time, after string) ([]string, bool, error) {
	if !p.Enabled() {
		return nil, false, domain.ErrProviderDisabled
	}
	params := &stripesdk.EventListParams{CreatedRange: &stripesdk.RangeQueryParams{GreaterThanOrEqual: from.Unix(), LesserThanOrEqual: to.Unix()}}
	params.Types = stripesdk.StringSlice([]string{"checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed", "checkout.session.expired", "invoice.paid", "invoice.payment_failed", "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted", "refund.created", "refund.updated", "refund.failed", "charge.refund.updated", "charge.refunded", "charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed", "charge.dispute.funds_withdrawn", "charge.dispute.funds_reinstated"})
	params.Context = ctx
	params.Limit = stripesdk.Int64(100)
	params.Single = true
	if after != "" {
		params.StartingAfter = stripesdk.String(after)
	}
	it := providerevent.List(params)
	ids := []string{}
	for it.Next() {
		ids = append(ids, it.Event().ID)
	}
	if err := it.Err(); err != nil {
		return nil, false, fmt.Errorf("stripe: list recovery events: %w", err)
	}
	return ids, it.Meta().HasMore, nil
}
