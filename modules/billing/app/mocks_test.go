package app

import (
	"context"
	"errors"
	"sync"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// mockProvider is a minimal port.Provider for use-case tests.
type mockProvider struct {
	name              string
	enabled           bool
	parseResult       *port.WebhookParseResult
	parseErr          error
	checkoutResult    *domain.CheckoutResult
	checkoutErr       error
	cancelErr         error
	changeErr         error
	scheduleErr       error
	reactivateErr     error
	ensureCustomerID  string
	ensureCustomerErr error
	subSnapshot       *domain.SubscriptionSnapshot
	portalResult      *domain.PortalSessionResult
	portalErr         error
	previewResult     *domain.SubscriptionPreview
	previewErr        error

	cancelCalls     int
	changeCalls     int
	scheduleCalls   int
	reactivateCalls int
	checkoutCalls   int
}

func newMockProvider() *mockProvider {
	return &mockProvider{name: "mock", enabled: true}
}

func (m *mockProvider) Name() string  { return m.name }
func (m *mockProvider) Enabled() bool { return m.enabled }

func (m *mockProvider) EnsureCustomer(ctx context.Context, userID, email, existing string) (string, error) {
	if m.ensureCustomerErr != nil {
		return "", m.ensureCustomerErr
	}
	if m.ensureCustomerID != "" {
		return m.ensureCustomerID, nil
	}
	if existing != "" {
		return existing, nil
	}
	return "cus_test_" + userID, nil
}

func (m *mockProvider) CreateCheckout(ctx context.Context, in domain.CheckoutInput) (*domain.CheckoutResult, error) {
	m.checkoutCalls++
	if m.checkoutErr != nil {
		return nil, m.checkoutErr
	}
	if m.checkoutResult != nil {
		return m.checkoutResult, nil
	}
	return &domain.CheckoutResult{SessionID: "cs_test", CheckoutURL: "https://example.com/c/cs_test"}, nil
}

func (m *mockProvider) CancelSubscription(ctx context.Context, subID string, mode domain.CancelMode) error {
	m.cancelCalls++
	return m.cancelErr
}

func (m *mockProvider) ChangeSubscription(ctx context.Context, subID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	m.changeCalls++
	if m.changeErr != nil {
		return nil, m.changeErr
	}
	if m.subSnapshot != nil {
		return m.subSnapshot, nil
	}
	return &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: subID,
		Plan:                   in.Plan,
		Interval:               in.Interval,
		Status:                 domain.StatusActive,
	}, nil
}

func (m *mockProvider) ScheduleSubscriptionChange(ctx context.Context, subID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	m.scheduleCalls++
	if m.scheduleErr != nil {
		return nil, m.scheduleErr
	}
	if m.subSnapshot != nil {
		return m.subSnapshot, nil
	}
	return &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: subID,
		Plan:                   in.Plan,
		Interval:               in.Interval,
		Status:                 domain.StatusActive,
	}, nil
}

func (m *mockProvider) ReactivateSubscription(ctx context.Context, subID string) error {
	m.reactivateCalls++
	return m.reactivateErr
}

func (m *mockProvider) GetSubscription(ctx context.Context, subID string) (*domain.SubscriptionSnapshot, error) {
	if m.subSnapshot != nil {
		return m.subSnapshot, nil
	}
	return &domain.SubscriptionSnapshot{ProviderSubscriptionID: subID, Status: domain.StatusActive}, nil
}

func (m *mockProvider) GetDefaultPaymentMethod(ctx context.Context, customerID string) (*domain.PaymentMethodCard, error) {
	return nil, nil
}

func (m *mockProvider) ListInvoices(ctx context.Context, customerID string, page, limit int) ([]domain.InvoiceItem, int, error) {
	return nil, 0, nil
}

func (m *mockProvider) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*domain.PortalSessionResult, error) {
	if m.portalErr != nil {
		return nil, m.portalErr
	}
	if m.portalResult != nil {
		return m.portalResult, nil
	}
	return &domain.PortalSessionResult{URL: "https://billing.stripe.test/session_123"}, nil
}

func (m *mockProvider) PreviewSubscriptionChange(ctx context.Context, customerID, subscriptionID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error) {
	if m.previewErr != nil {
		return nil, m.previewErr
	}
	if m.previewResult != nil {
		return m.previewResult, nil
	}
	return &domain.SubscriptionPreview{
		Currency:             "usd",
		AmountDueNow:         30,
		TargetPlan:           in.Plan,
		TargetInterval:       in.Interval,
		Mode:                 in.Mode,
		ImmediateCharge:      in.Mode != domain.ChangeModePeriodEnd,
		EffectiveAtPeriodEnd: in.Mode == domain.ChangeModePeriodEnd,
		Message:              "preview ready",
	}, nil
}

func (m *mockProvider) VerifyAndParseWebhook(payload []byte, signature string) (*port.WebhookParseResult, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.parseResult, nil
}

func (m *mockProvider) MapPriceToPlan(priceID string) (domain.PlanType, domain.BillingInterval) {
	return domain.PlanFree, ""
}

func (m *mockProvider) CreditsPerUnit() int64        { return 40 }
func (m *mockProvider) IsCreditsPriceID(string) bool { return false }
func (m *mockProvider) LifetimePriceID() string      { return "price_lifetime" }

// --- mockRepo ----------------------------------------------------------

type mockRepo struct {
	mu        sync.Mutex
	rows      map[string]*domain.BillingEvent
	createErr error
	markErr   error
}

func newMockRepo() *mockRepo {
	return &mockRepo{rows: map[string]*domain.BillingEvent{}}
}

func (r *mockRepo) CreateIfAbsent(ctx context.Context, e *domain.BillingEvent) (*domain.BillingEvent, bool, error) {
	if r.createErr != nil {
		return nil, false, r.createErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.rows[e.ProviderEventID]; ok {
		return existing, false, nil
	}
	r.rows[e.ProviderEventID] = e
	return e, true, nil
}

func (r *mockRepo) MarkProcessed(ctx context.Context, provider, id string) error {
	if r.markErr != nil {
		return r.markErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.rows[id]; ok {
		e.Processed = true
	}
	return nil
}

// --- mockBus ------------------------------------------------------------

type mockBus struct {
	mu        sync.Mutex
	published []event.Envelope
}

func newMockBus() *mockBus { return &mockBus{} }

func (b *mockBus) Subscribe(kind event.Kind, fn port.Listener) {}

func (b *mockBus) Publish(ctx context.Context, env event.Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, env)
}

func (b *mockBus) Published() []event.Envelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]event.Envelope, len(b.published))
	copy(out, b.published)
	return out
}

// --- mockResolver -------------------------------------------------------

type mockResolver struct {
	resolveTo  string
	resolveErr error
	calls      int
}

func (r *mockResolver) Resolve(ctx context.Context, h port.UserHint) (string, error) {
	r.calls++
	if r.resolveErr != nil {
		return "", r.resolveErr
	}
	if r.resolveTo != "" {
		return r.resolveTo, nil
	}
	return h.UserID, nil
}

// --- mockCustomerStore --------------------------------------------------

type mockCustomerStore struct {
	customer port.Customer
	loadErr  error
	saved    map[string]string
}

func newMockCustomerStore(c port.Customer) *mockCustomerStore {
	return &mockCustomerStore{customer: c, saved: map[string]string{}}
}

func (s *mockCustomerStore) LoadCustomer(ctx context.Context, userID string) (port.Customer, error) {
	if s.loadErr != nil {
		return port.Customer{}, s.loadErr
	}
	return s.customer, nil
}

func (s *mockCustomerStore) SaveCustomerID(ctx context.Context, userID, provider, customerID string) error {
	s.saved[userID] = customerID
	return nil
}

// helper to satisfy unused-import warnings if any test removes uses
var _ = errors.New
