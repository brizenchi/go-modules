package stripe

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	stripesdk "github.com/stripe/stripe-go/v76"
	billingportalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/invoice"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/subscriptionschedule"
)

// Provider implements port.Provider for Stripe.
type Provider struct {
	cfg Config
}

// buildCheckoutMetadata merges caller-supplied metadata with billing's
// system fields. System fields are written last and therefore win on
// collision — this prevents request bodies that pass-through Metadata
// (e.g. for Rewardful) from spoofing user_id / email / plan.
func buildCheckoutMetadata(in domain.CheckoutInput, priceID string, quantity int64) map[string]string {
	m := make(map[string]string, len(in.Metadata)+8)
	for k, v := range in.Metadata {
		m[k] = v
	}
	m["user_id"] = in.UserID
	m["email"] = in.Email
	m["plan"] = string(in.Plan)
	m["interval"] = string(in.Interval)
	m["product_type"] = string(in.ProductType)
	m["price_id"] = priceID
	m["quantity"] = strconv.FormatInt(quantity, 10)
	return m
}

// NewProvider builds a Stripe provider. If cfg.Enabled is true and
// cfg.SecretKey is set, the global stripesdk.Key is initialized.
func NewProvider(cfg Config) *Provider {
	if cfg.Enabled && cfg.SecretKey != "" {
		stripesdk.Key = cfg.SecretKey
	}
	return &Provider{cfg: cfg}
}

func (p *Provider) Name() string            { return "stripe" }
func (p *Provider) Enabled() bool           { return p.cfg.Enabled }
func (p *Provider) LifetimePriceID() string { return p.cfg.LifetimePriceID }

func (p *Provider) MapPriceToPlan(priceID string) (domain.PlanType, domain.BillingInterval) {
	return p.cfg.PlanForPrice(priceID)
}

func (p *Provider) CreditsPerUnit() int64           { return p.cfg.CreditsPerUnit }
func (p *Provider) IsCreditsPriceID(id string) bool { return p.cfg.IsCreditsPriceID(id) }

// EnsureCustomer returns a Stripe customer ID, creating one if needed.
func (p *Provider) EnsureCustomer(ctx context.Context, userID, email, existingID string) (string, error) {
	if !p.cfg.Enabled {
		return "", domain.ErrProviderDisabled
	}
	if existingID != "" {
		if cust, err := customer.Get(existingID, nil); err == nil && cust != nil {
			return cust.ID, nil
		}
		slog.WarnContext(ctx, "stripe: existing customer not found, creating new", "customer_id", existingID, "user_id", userID)
	}
	params := &stripesdk.CustomerParams{Email: stripesdk.String(email)}
	params.AddMetadata("user_id", userID)
	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe: create customer: %w", err)
	}
	slog.InfoContext(ctx, "stripe: customer created", "customer_id", cust.ID, "user_id", userID)
	return cust.ID, nil
}

// CreateCheckout creates a Stripe Checkout session.
func (p *Provider) CreateCheckout(ctx context.Context, in domain.CheckoutInput) (*domain.CheckoutResult, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}

	var (
		priceID string
		mode    stripesdk.CheckoutSessionMode
	)
	quantity := in.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	switch in.ProductType {
	case domain.ProductCredits:
		priceID = in.PriceID
		if priceID == "" && len(p.cfg.CreditsPriceIDs) > 0 {
			priceID = p.cfg.CreditsPriceIDs[0]
		}
		if priceID == "" {
			return nil, fmt.Errorf("%w: price_id required for credits", domain.ErrInvalidInput)
		}
		if !p.cfg.IsCreditsPriceID(priceID) {
			return nil, domain.ErrInvalidPriceID
		}
		mode = stripesdk.CheckoutSessionModePayment
	case domain.ProductLifetime:
		priceID = strings.TrimSpace(p.cfg.LifetimePriceID)
		if priceID == "" {
			return nil, fmt.Errorf("%w: lifetime price not configured", domain.ErrPriceNotFound)
		}
		in.Plan = domain.PlanLifetime
		in.Interval = ""
		quantity = 1
		mode = stripesdk.CheckoutSessionModePayment
	case domain.ProductSubscription:
		priceID = p.cfg.PriceFor(in.Plan, in.Interval)
		if priceID == "" {
			return nil, fmt.Errorf("%w: plan=%s interval=%s", domain.ErrPriceNotFound, in.Plan, in.Interval)
		}
		mode = stripesdk.CheckoutSessionModeSubscription
	default:
		return nil, fmt.Errorf("%w: unknown product_type", domain.ErrInvalidInput)
	}

	params := &stripesdk.CheckoutSessionParams{
		Mode: stripesdk.String(string(mode)),
		LineItems: []*stripesdk.CheckoutSessionLineItemParams{
			{Price: stripesdk.String(priceID), Quantity: stripesdk.Int64(quantity)},
		},
		SuccessURL:          stripesdk.String(in.SuccessURL),
		CancelURL:           stripesdk.String(in.CancelURL),
		ClientReferenceID:   stripesdk.String(in.UserID),
		AllowPromotionCodes: stripesdk.Bool(true),
	}

	if in.ProductType == domain.ProductSubscription {
		// TrialDays < 0 means "explicitly skip trial" (e.g. returning user).
		// TrialDays == 0 means "use config default".
		// TrialDays > 0 means "override with this value".
		trial := in.TrialDays
		if trial == 0 {
			trial = p.cfg.TrialDays
		}
		if trial > 0 {
			params.SubscriptionData = &stripesdk.CheckoutSessionSubscriptionDataParams{
				TrialPeriodDays: stripesdk.Int64(trial),
			}
			params.PaymentMethodCollection = stripesdk.String("always")
		}
		if params.PaymentMethodCollection == nil {
			params.PaymentMethodCollection = stripesdk.String("always")
		}
	}

	if in.ProductType == domain.ProductCredits {
		params.LineItems[0].AdjustableQuantity = &stripesdk.CheckoutSessionLineItemAdjustableQuantityParams{
			Enabled: stripesdk.Bool(true),
			Minimum: stripesdk.Int64(1),
			Maximum: stripesdk.Int64(100),
		}
	}

	if in.CustomerID != "" {
		params.Customer = stripesdk.String(in.CustomerID)
	} else if in.Email != "" {
		params.CustomerEmail = stripesdk.String(in.Email)
	}

	for k, v := range buildCheckoutMetadata(in, priceID, quantity) {
		params.AddMetadata(k, v)
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create checkout: %w", err)
	}
	slog.InfoContext(ctx, "stripe: checkout created", "session_id", sess.ID, "user_id", in.UserID, "plan", in.Plan)
	return &domain.CheckoutResult{SessionID: sess.ID, CheckoutURL: sess.URL}, nil
}

// CancelSubscription schedules cancellation according to mode.
func (p *Provider) CancelSubscription(ctx context.Context, subID string, mode domain.CancelMode) error {
	if !p.cfg.Enabled {
		return domain.ErrProviderDisabled
	}
	if subID == "" {
		return fmt.Errorf("%w: subscription_id required", domain.ErrInvalidInput)
	}
	params := &stripesdk.SubscriptionParams{}
	switch mode {
	case domain.CancelIn3Days:
		cancelAt := time.Now().Add(3 * 24 * time.Hour).Unix()
		params.CancelAt = stripesdk.Int64(cancelAt)
	case domain.CancelAtPeriodEnd:
		params.CancelAtPeriodEnd = stripesdk.Bool(true)
	default:
		return domain.ErrInvalidCancelMode
	}
	if _, err := subscription.Update(subID, params); err != nil {
		return fmt.Errorf("stripe: cancel subscription: %w", err)
	}
	slog.InfoContext(ctx, "stripe: subscription cancellation scheduled", "subscription_id", subID, "mode", mode)
	return nil
}

func (p *Provider) ChangeSubscription(ctx context.Context, subID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if subID == "" {
		return nil, fmt.Errorf("%w: subscription_id required", domain.ErrInvalidInput)
	}
	if !in.Plan.Valid() || in.Plan == domain.PlanFree {
		return nil, fmt.Errorf("%w: paid plan required", domain.ErrInvalidInput)
	}
	if !in.Interval.Valid() {
		return nil, fmt.Errorf("%w: billing interval required", domain.ErrInvalidInput)
	}

	priceID := p.cfg.PriceFor(in.Plan, in.Interval)
	if priceID == "" {
		return nil, fmt.Errorf("%w: plan=%s interval=%s", domain.ErrPriceNotFound, in.Plan, in.Interval)
	}

	current, err := subscription.Get(subID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get current subscription: %w", err)
	}
	if current == nil || len(current.Items.Data) == 0 || current.Items.Data[0] == nil {
		return nil, fmt.Errorf("%w: missing subscription item", domain.ErrInvalidInput)
	}

	item := current.Items.Data[0]
	params := &stripesdk.SubscriptionParams{
		Items: []*stripesdk.SubscriptionItemsParams{
			{
				ID:    stripesdk.String(item.ID),
				Price: stripesdk.String(priceID),
			},
		},
		ProrationBehavior: stripesdk.String("always_invoice"),
		PaymentBehavior:   stripesdk.String("pending_if_incomplete"),
	}
	if in.Mode == domain.ChangeModeImmediateResetCycle {
		params.BillingCycleAnchorNow = stripesdk.Bool(true)
	} else {
		params.BillingCycleAnchorUnchanged = stripesdk.Bool(true)
	}

	updated, err := subscription.Update(subID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: change subscription: %w", err)
	}
	slog.InfoContext(
		ctx,
		"stripe: subscription changed",
		"subscription_id", subID,
		"plan", in.Plan,
		"interval", in.Interval,
		"change_mode", in.Mode,
	)
	return p.snapshotFromSubscription(updated), nil
}

func (p *Provider) ScheduleSubscriptionChange(ctx context.Context, subID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if subID == "" {
		return nil, fmt.Errorf("%w: subscription_id required", domain.ErrInvalidInput)
	}

	priceID := p.cfg.PriceFor(in.Plan, in.Interval)
	if priceID == "" {
		return nil, fmt.Errorf("%w: plan=%s interval=%s", domain.ErrPriceNotFound, in.Plan, in.Interval)
	}

	current, err := subscription.Get(subID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get current subscription: %w", err)
	}
	if current == nil || current.Schedule == nil || current.Schedule.ID == "" {
		schedule, err := subscriptionschedule.New(&stripesdk.SubscriptionScheduleParams{
			FromSubscription: stripesdk.String(subID),
			EndBehavior:      stripesdk.String(string(stripesdk.SubscriptionScheduleEndBehaviorRelease)),
		})
		if err != nil {
			return nil, fmt.Errorf("stripe: create subscription schedule: %w", err)
		}
		current.Schedule = schedule
	}

	scheduleID := current.Schedule.ID
	schedule, err := subscriptionschedule.Get(scheduleID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get subscription schedule: %w", err)
	}
	if schedule == nil || schedule.CurrentPhase == nil || schedule.Subscription == nil {
		return nil, fmt.Errorf("%w: active subscription schedule required", domain.ErrInvalidInput)
	}

	start := schedule.CurrentPhase.StartDate
	end := schedule.CurrentPhase.EndDate
	if end <= 0 {
		return nil, fmt.Errorf("%w: current phase end required", domain.ErrInvalidInput)
	}

	params := &stripesdk.SubscriptionScheduleParams{
		EndBehavior:       stripesdk.String(string(stripesdk.SubscriptionScheduleEndBehaviorRelease)),
		ProrationBehavior: stripesdk.String("none"),
		Phases: []*stripesdk.SubscriptionSchedulePhaseParams{
			{
				StartDate: stripesdk.Int64(start),
				EndDate:   stripesdk.Int64(end),
				Items: []*stripesdk.SubscriptionSchedulePhaseItemParams{
					{
						Price:    stripesdk.String(current.Items.Data[0].Price.ID),
						Quantity: stripesdk.Int64(current.Items.Data[0].Quantity),
					},
				},
			},
			{
				StartDate: stripesdk.Int64(end),
				Items: []*stripesdk.SubscriptionSchedulePhaseItemParams{
					{
						Price:    stripesdk.String(priceID),
						Quantity: stripesdk.Int64(current.Items.Data[0].Quantity),
					},
				},
				ProrationBehavior: stripesdk.String("none"),
			},
		},
	}

	if _, err := subscriptionschedule.Update(scheduleID, params); err != nil {
		return nil, fmt.Errorf("stripe: schedule subscription change: %w", err)
	}

	snap := p.snapshotFromSubscription(current)
	if snap != nil {
		snap.CancelAtPeriodEnd = false
	}
	return snap, nil
}

func (p *Provider) ReactivateSubscription(ctx context.Context, subID string) error {
	if !p.cfg.Enabled {
		return domain.ErrProviderDisabled
	}
	if subID == "" {
		return fmt.Errorf("%w: subscription_id required", domain.ErrInvalidInput)
	}
	params := &stripesdk.SubscriptionParams{
		CancelAtPeriodEnd: stripesdk.Bool(false),
		CancelAt:          stripesdk.Int64(0),
	}
	if _, err := subscription.Update(subID, params); err != nil {
		return fmt.Errorf("stripe: reactivate subscription: %w", err)
	}
	slog.InfoContext(ctx, "stripe: subscription reactivated", "subscription_id", subID)
	return nil
}

func (p *Provider) GetSubscription(ctx context.Context, subID string) (*domain.SubscriptionSnapshot, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if subID == "" {
		return nil, fmt.Errorf("%w: subscription_id required", domain.ErrInvalidInput)
	}
	sub, err := subscription.Get(subID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get subscription: %w", err)
	}
	return p.snapshotFromSubscription(sub), nil
}

func (p *Provider) GetDefaultPaymentMethod(ctx context.Context, customerID string) (*domain.PaymentMethodCard, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if customerID == "" {
		return nil, nil
	}
	params := &stripesdk.CustomerParams{}
	params.AddExpand("invoice_settings.default_payment_method")
	cust, err := customer.Get(customerID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe: get customer: %w", err)
	}
	if cust == nil || cust.InvoiceSettings == nil || cust.InvoiceSettings.DefaultPaymentMethod == nil || cust.InvoiceSettings.DefaultPaymentMethod.Card == nil {
		return nil, nil
	}
	card := cust.InvoiceSettings.DefaultPaymentMethod.Card
	return &domain.PaymentMethodCard{
		Brand:    string(card.Brand),
		Last4:    card.Last4,
		ExpMonth: card.ExpMonth,
		ExpYear:  card.ExpYear,
	}, nil
}

func (p *Provider) ListInvoices(ctx context.Context, customerID string, page, limit int) ([]domain.InvoiceItem, int, error) {
	if !p.cfg.Enabled {
		return nil, 0, domain.ErrProviderDisabled
	}
	if customerID == "" {
		return []domain.InvoiceItem{}, 0, nil
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	total := 0
	items := make([]domain.InvoiceItem, 0, limit)

	params := &stripesdk.InvoiceListParams{Customer: stripesdk.String(customerID)}
	iter := invoice.List(params)
	for iter.Next() {
		inv := iter.Invoice()
		if inv == nil {
			continue
		}
		if total >= offset && len(items) < limit {
			created := time.Unix(inv.Created, 0).UTC()
			items = append(items, domain.InvoiceItem{
				ID:        inv.ID,
				AmountUSD: float64(inv.AmountPaid) / 100.0,
				Status:    string(inv.Status),
				Period:    created.Format("2006-01"),
				PDFURL:    inv.InvoicePDF,
				CreatedAt: created,
			})
		}
		total++
	}
	if err := iter.Err(); err != nil {
		return nil, 0, fmt.Errorf("stripe: list invoices: %w", err)
	}
	return items, total, nil
}

func (p *Provider) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*domain.PortalSessionResult, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if customerID == "" {
		return nil, domain.ErrNoBillingCustomer
	}
	if returnURL == "" {
		return nil, fmt.Errorf("%w: return_url required", domain.ErrInvalidInput)
	}

	params := &stripesdk.BillingPortalSessionParams{
		Customer:  stripesdk.String(customerID),
		ReturnURL: stripesdk.String(returnURL),
	}
	sess, err := billingportalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: create billing portal session: %w", err)
	}
	return &domain.PortalSessionResult{URL: sess.URL}, nil
}

func (p *Provider) PreviewSubscriptionChange(ctx context.Context, customerID, subID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error) {
	if !p.cfg.Enabled {
		return nil, domain.ErrProviderDisabled
	}
	if subID == "" {
		return &domain.SubscriptionPreview{
			Currency:        "usd",
			AmountDueNow:    0,
			TargetPlan:      in.Plan,
			TargetInterval:  in.Interval,
			Mode:            in.Mode,
			ImmediateCharge: true,
			Message:         "new subscription will be created through checkout",
		}, nil
	}

	current, err := subscription.Get(subID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: get current subscription: %w", err)
	}
	if current == nil || len(current.Items.Data) == 0 || current.Items.Data[0] == nil {
		return nil, fmt.Errorf("%w: missing subscription item", domain.ErrInvalidInput)
	}

	targetPriceID := p.cfg.PriceFor(in.Plan, in.Interval)
	if targetPriceID == "" {
		return nil, fmt.Errorf("%w: plan=%s interval=%s", domain.ErrPriceNotFound, in.Plan, in.Interval)
	}

	preview := &domain.SubscriptionPreview{
		Currency:       string(current.Currency),
		TargetPlan:     in.Plan,
		TargetInterval: in.Interval,
		Mode:           in.Mode,
	}
	if end := unixToTimePtr(current.CurrentPeriodEnd); end != nil {
		preview.CurrentPeriodEnd = end
	}

	if in.Mode == domain.ChangeModePeriodEnd {
		preview.ImmediateCharge = false
		preview.EffectiveAtPeriodEnd = true
		preview.NextBillingAt = unixToTimePtr(current.CurrentPeriodEnd)
		preview.Message = "change will take effect at the next renewal"
		return preview, nil
	}

	params := &stripesdk.InvoiceUpcomingParams{
		Customer:     stripesdk.String(customerID),
		Subscription: stripesdk.String(subID),
		SubscriptionItems: []*stripesdk.SubscriptionItemsParams{
			{
				ID:    stripesdk.String(current.Items.Data[0].ID),
				Price: stripesdk.String(targetPriceID),
			},
		},
		SubscriptionProrationBehavior: stripesdk.String("always_invoice"),
	}
	if in.Mode == domain.ChangeModeImmediateResetCycle {
		params.SubscriptionBillingCycleAnchorNow = stripesdk.Bool(true)
	} else {
		params.SubscriptionBillingCycleAnchorUnchanged = stripesdk.Bool(true)
	}

	upcoming, err := invoice.Upcoming(params)
	if err != nil {
		return nil, fmt.Errorf("stripe: preview subscription change: %w", err)
	}

	preview.AmountDueNow = float64(upcoming.AmountDue) / 100.0
	preview.ImmediateCharge = true
	if upcoming.NextPaymentAttempt > 0 {
		preview.NextBillingAt = unixToTimePtr(upcoming.NextPaymentAttempt)
	}
	if preview.NextBillingAt == nil {
		preview.NextBillingAt = unixToTimePtr(current.CurrentPeriodEnd)
	}
	if in.Mode == domain.ChangeModeImmediateResetCycle {
		preview.Message = "subscription will switch now and restart the billing cycle"
	} else {
		preview.Message = "subscription will switch now with prorated billing"
	}
	return preview, nil
}

// snapshotFromSubscription maps a stripe.Subscription to a domain snapshot.
func (p *Provider) snapshotFromSubscription(sub *stripesdk.Subscription) *domain.SubscriptionSnapshot {
	snap := &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: sub.ID,
		Status:                 normalizeStripeStatus(string(sub.Status), sub.CancelAtPeriodEnd),
		CancelAtPeriodEnd:      sub.CancelAtPeriodEnd,
	}
	if sub.Customer != nil {
		snap.ProviderCustomerID = sub.Customer.ID
	}
	if start := unixToTimePtr(sub.CurrentPeriodStart); start != nil {
		snap.PeriodStart = start
	}
	if end := unixToTimePtr(sub.CurrentPeriodEnd); end != nil {
		snap.PeriodEnd = end
	}
	if cancelAt := unixToTimePtr(sub.CancelAt); cancelAt != nil {
		snap.CancelEffectiveAt = cancelAt
	} else if sub.CancelAtPeriodEnd && snap.PeriodEnd != nil {
		snap.CancelEffectiveAt = snap.PeriodEnd
	}
	if len(sub.Items.Data) > 0 && sub.Items.Data[0] != nil && sub.Items.Data[0].Price != nil {
		price := sub.Items.Data[0].Price
		snap.ProviderPriceID = price.ID
		if price.Product != nil {
			snap.ProviderProductID = price.Product.ID
		}
		if price.Recurring != nil {
			switch price.Recurring.Interval {
			case stripesdk.PriceRecurringIntervalMonth:
				snap.Interval = domain.IntervalMonthly
			case stripesdk.PriceRecurringIntervalYear:
				snap.Interval = domain.IntervalYearly
			}
		}
		plan, _ := p.cfg.PlanForPrice(price.ID)
		snap.Plan = plan
	}
	return snap
}

// Compile-time check that Provider satisfies port.Provider.
var _ port.Provider = (*Provider)(nil)

func unixToTimePtr(ts int64) *time.Time {
	if ts <= 0 {
		return nil
	}
	t := time.Unix(ts, 0).UTC()
	return &t
}
