package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// mockProvider is a minimal port.Provider for use-case tests.
type mockProvider struct {
	name                   string
	enabled                bool
	parseResult            *port.WebhookParseResult
	parseErr               error
	checkoutResult         *domain.CheckoutResult
	checkoutErr            error
	checkoutSession        *domain.CheckoutSessionSnapshot
	checkoutSessionErr     error
	findCheckoutSession    *domain.CheckoutSessionSnapshot
	findCheckoutSessionErr error
	cancelErr              error
	changeErr              error
	scheduleErr            error
	reactivateErr          error
	ensureCustomerID       string
	ensureCustomerErr      error
	subSnapshot            *domain.SubscriptionSnapshot
	subErr                 error
	portalResult           *domain.PortalSessionResult
	portalErr              error
	previewResult          *domain.SubscriptionPreview
	previewErr             error

	cancelCalls              int
	changeCalls              int
	scheduleCalls            int
	reactivateCalls          int
	checkoutCalls            int
	checkoutInputs           []domain.CheckoutInput
	checkoutSessionCalls     int
	findCheckoutSessionCalls int
	checkoutStarted          chan struct{}
	checkoutContinue         chan struct{}
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
	m.checkoutInputs = append(m.checkoutInputs, in)
	if m.checkoutStarted != nil {
		close(m.checkoutStarted)
		m.checkoutStarted = nil
	}
	if m.checkoutContinue != nil {
		<-m.checkoutContinue
	}
	if m.checkoutErr != nil {
		return nil, m.checkoutErr
	}
	if m.checkoutResult != nil {
		return m.checkoutResult, nil
	}
	return &domain.CheckoutResult{SessionID: "cs_test", CheckoutURL: "https://example.com/c/cs_test"}, nil
}

func (m *mockProvider) GetCheckoutSession(ctx context.Context, providerSessionID string) (*domain.CheckoutSessionSnapshot, error) {
	m.checkoutSessionCalls++
	if m.checkoutSessionErr != nil {
		return nil, m.checkoutSessionErr
	}
	if m.checkoutSession != nil {
		copy := *m.checkoutSession
		return &copy, nil
	}
	return &domain.CheckoutSessionSnapshot{SessionID: providerSessionID, State: domain.CheckoutSessionOpen}, nil
}

func (m *mockProvider) FindCheckoutSession(ctx context.Context, providerCustomerID, reservationID string) (*domain.CheckoutSessionSnapshot, error) {
	m.findCheckoutSessionCalls++
	if m.findCheckoutSessionErr != nil {
		return nil, m.findCheckoutSessionErr
	}
	if m.findCheckoutSession != nil {
		copy := *m.findCheckoutSession
		return &copy, nil
	}
	return nil, nil
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
	if m.reactivateErr != nil {
		return m.reactivateErr
	}
	if m.subSnapshot != nil {
		updated := *m.subSnapshot
		updated.Status = domain.StatusActive
		updated.CancelAtPeriodEnd = false
		updated.CancelEffectiveAt = nil
		m.subSnapshot = &updated
	}
	return nil
}

func (m *mockProvider) GetSubscription(ctx context.Context, subID string) (*domain.SubscriptionSnapshot, error) {
	if m.subErr != nil {
		return nil, m.subErr
	}
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
	mu         sync.Mutex
	published  []event.Envelope
	publishErr error
}

func newMockBus() *mockBus { return &mockBus{} }

func (b *mockBus) Subscribe(kind event.Kind, fn port.Listener) {}

func (b *mockBus) Publish(ctx context.Context, env event.Envelope) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, env)
	return b.publishErr
}

func (b *mockBus) Published() []event.Envelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]event.Envelope, len(b.published))
	copy(out, b.published)
	return out
}

type snapshotWrite struct {
	userID          string
	provider        string
	snapshot        domain.SubscriptionSnapshot
	occurredAt      time.Time
	providerEventID string
}

type mockSubscriptionRepo struct {
	writes []snapshotWrite
	err    error
	skip   bool
}

func (r *mockSubscriptionRepo) UpsertSnapshot(ctx context.Context, userID, provider string, snapshot domain.SubscriptionSnapshot, occurredAt time.Time, providerEventID string) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	if r.skip {
		return false, nil
	}
	r.writes = append(r.writes, snapshotWrite{userID: userID, provider: provider, snapshot: snapshot, occurredAt: occurredAt, providerEventID: providerEventID})
	return true, nil
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
	customer                   port.Customer
	loadErr                    error
	saved                      map[string]string
	hasUsedTrial               bool
	reservationMu              sync.Mutex
	reservation                *domain.BillingCheckoutReservation
	completeReservationErrOnce error
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

func (s *mockCustomerStore) HasUsedTrial(ctx context.Context, userID string) (bool, error) {
	return s.hasUsedTrial, nil
}

func (s *mockCustomerStore) ReserveCheckout(ctx context.Context, userID, provider, providerCustomerID, reservationID, intentKey string, now, expiresAt time.Time) (*domain.BillingCheckoutReservation, bool, error) {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation == nil {
		s.reservation = &domain.BillingCheckoutReservation{
			UserID: userID, Provider: provider, ProviderCustomerID: providerCustomerID, ReservationID: reservationID,
			IntentKey: intentKey, ExpiresAt: expiresAt,
		}
		copy := *s.reservation
		return &copy, true, nil
	}
	copy := *s.reservation
	return &copy, false, nil
}

func (s *mockCustomerStore) CompleteCheckoutReservation(ctx context.Context, userID, provider, reservationID string, result domain.CheckoutResult) error {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.completeReservationErrOnce != nil {
		err := s.completeReservationErrOnce
		s.completeReservationErrOnce = nil
		return err
	}
	if s.reservation == nil || s.reservation.ReservationID != reservationID {
		return errors.New("reservation lost")
	}
	s.reservation.SessionID = result.SessionID
	s.reservation.CheckoutURL = result.CheckoutURL
	return nil
}

func (s *mockCustomerStore) ReplaceCheckoutReservation(ctx context.Context, userID, provider, providerCustomerID, expectedReservationID, reservationID, intentKey string, expiresAt time.Time) (bool, error) {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation == nil || s.reservation.ReservationID != expectedReservationID {
		return false, nil
	}
	s.reservation = &domain.BillingCheckoutReservation{
		UserID: userID, Provider: provider, ProviderCustomerID: providerCustomerID, ReservationID: reservationID,
		IntentKey: intentKey, ExpiresAt: expiresAt,
	}
	return true, nil
}

func (s *mockCustomerStore) LinkCheckoutSubscription(ctx context.Context, provider, providerSessionID, providerSubscriptionID string) error {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation != nil && s.reservation.Provider == provider && s.reservation.SessionID == providerSessionID {
		s.reservation.ProviderSubscriptionID = providerSubscriptionID
	}
	return nil
}

func (s *mockCustomerStore) ReleaseCheckoutReservation(ctx context.Context, provider, providerSessionID string) (bool, error) {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation != nil && s.reservation.Provider == provider && s.reservation.SessionID == providerSessionID {
		s.reservation = nil
		return true, nil
	}
	return false, nil
}

func (s *mockCustomerStore) ReleaseCheckoutReservationByReservationID(ctx context.Context, provider, reservationID string) (bool, error) {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation != nil && s.reservation.Provider == provider && s.reservation.ReservationID == reservationID {
		s.reservation = nil
		return true, nil
	}
	return false, nil
}

func (s *mockCustomerStore) ReleaseCheckoutReservationBySubscription(ctx context.Context, provider, providerSubscriptionID string) (bool, error) {
	s.reservationMu.Lock()
	defer s.reservationMu.Unlock()
	if s.reservation != nil && s.reservation.Provider == provider && s.reservation.ProviderSubscriptionID == providerSubscriptionID {
		s.reservation = nil
		return true, nil
	}
	return false, nil
}

// helper to satisfy unused-import warnings if any test removes uses
var _ = errors.New
