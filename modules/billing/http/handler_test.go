package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/app"
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/gin-gonic/gin"
)

func TestSanitizeMetadata_DropsReservedAndEmpty(t *testing.T) {
	in := map[string]string{
		"referral":   "rwf_abc",
		"campaign":   "spring",
		"user_id":    "spoof",   // reserved → dropped
		"plan":       "premium", // reserved → dropped
		"":           "v",       // empty key → dropped
		"k":          "",        // empty value → dropped
		"  spaced  ": "  v  ",   // trimmed
	}
	out := sanitizeMetadata(in)

	if got := out["referral"]; got != "rwf_abc" {
		t.Errorf("referral = %q", got)
	}
	if got := out["campaign"]; got != "spring" {
		t.Errorf("campaign = %q", got)
	}
	if _, ok := out["user_id"]; ok {
		t.Errorf("user_id should be dropped (reserved)")
	}
	if _, ok := out["plan"]; ok {
		t.Errorf("plan should be dropped (reserved)")
	}
	if got := out["spaced"]; got != "v" {
		t.Errorf("trim failed, got %q", got)
	}
	if len(out) != 3 {
		t.Errorf("expected 3 entries, got %d: %+v", len(out), out)
	}
}

func TestSanitizeMetadata_NilOnEmpty(t *testing.T) {
	if out := sanitizeMetadata(nil); out != nil {
		t.Errorf("nil input should yield nil, got %+v", out)
	}
	if out := sanitizeMetadata(map[string]string{"user_id": "x"}); out != nil {
		t.Errorf("only-reserved input should yield nil, got %+v", out)
	}
}

func TestSanitizeMetadata_EnforcesCap(t *testing.T) {
	in := make(map[string]string, 50)
	for i := range 50 {
		in["k"+strconv.Itoa(i)] = "v"
	}
	out := sanitizeMetadata(in)
	if len(out) != maxMetadataEntries {
		t.Errorf("len = %d, want %d", len(out), maxMetadataEntries)
	}
}

type handlerTestProvider struct{}

func (handlerTestProvider) Name() string  { return "stripe" }
func (handlerTestProvider) Enabled() bool { return true }
func (handlerTestProvider) EnsureCustomer(ctx context.Context, userID, email, existingCustomerID string) (string, error) {
	return "cus_x", nil
}
func (handlerTestProvider) CreateCheckout(ctx context.Context, in domain.CheckoutInput) (*domain.CheckoutResult, error) {
	return &domain.CheckoutResult{SessionID: "cs_test", CheckoutURL: "https://checkout.stripe.test/session"}, nil
}
func (handlerTestProvider) CancelSubscription(ctx context.Context, providerSubscriptionID string, mode domain.CancelMode) error {
	return nil
}
func (handlerTestProvider) ChangeSubscription(ctx context.Context, providerSubscriptionID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	return &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: providerSubscriptionID,
		Plan:                   in.Plan,
		Interval:               in.Interval,
		Status:                 domain.StatusActive,
	}, nil
}
func (handlerTestProvider) ScheduleSubscriptionChange(ctx context.Context, providerSubscriptionID string, in domain.SubscriptionChangeInput) (*domain.SubscriptionSnapshot, error) {
	return &domain.SubscriptionSnapshot{
		ProviderSubscriptionID: providerSubscriptionID,
		Plan:                   in.Plan,
		Interval:               in.Interval,
		Status:                 domain.StatusActive,
	}, nil
}
func (handlerTestProvider) ReactivateSubscription(ctx context.Context, providerSubscriptionID string) error {
	return nil
}
func (handlerTestProvider) GetSubscription(ctx context.Context, providerSubscriptionID string) (*domain.SubscriptionSnapshot, error) {
	return &domain.SubscriptionSnapshot{ProviderSubscriptionID: providerSubscriptionID, Status: domain.StatusActive}, nil
}
func (handlerTestProvider) GetDefaultPaymentMethod(ctx context.Context, providerCustomerID string) (*domain.PaymentMethodCard, error) {
	return nil, nil
}
func (handlerTestProvider) ListInvoices(ctx context.Context, providerCustomerID string, page, limit int) ([]domain.InvoiceItem, int, error) {
	return []domain.InvoiceItem{}, 0, nil
}
func (handlerTestProvider) CreateBillingPortalSession(ctx context.Context, providerCustomerID, returnURL string) (*domain.PortalSessionResult, error) {
	return &domain.PortalSessionResult{URL: "https://billing.stripe.test/session_123"}, nil
}
func (handlerTestProvider) PreviewSubscriptionChange(ctx context.Context, providerCustomerID, providerSubscriptionID string, in domain.SubscriptionPreviewInput) (*domain.SubscriptionPreview, error) {
	return &domain.SubscriptionPreview{
		Currency:             "usd",
		AmountDueNow:         30,
		TargetPlan:           in.Plan,
		TargetInterval:       in.Interval,
		Mode:                 domain.ChangeModeImmediateProrated,
		ImmediateCharge:      true,
		EffectiveAtPeriodEnd: false,
		Message:              "preview ready",
	}, nil
}
func (handlerTestProvider) VerifyAndParseWebhook(payload []byte, signature string) (*port.WebhookParseResult, error) {
	return nil, nil
}
func (handlerTestProvider) MapPriceToPlan(priceID string) (domain.PlanType, domain.BillingInterval) {
	return domain.PlanFree, ""
}
func (handlerTestProvider) CreditsPerUnit() int64 { return 40 }
func (handlerTestProvider) IsCreditsPriceID(priceID string) bool {
	return false
}

type handlerTestCustomerStore struct {
	customer port.Customer
}

func (s handlerTestCustomerStore) LoadCustomer(ctx context.Context, userID string) (port.Customer, error) {
	return s.customer, nil
}
func (s handlerTestCustomerStore) SaveCustomerID(ctx context.Context, userID, provider, customerID string) error {
	return nil
}

type handlerTestBus struct{}

func (handlerTestBus) Subscribe(kind event.Kind, listener port.Listener) {}
func (handlerTestBus) Publish(ctx context.Context, env event.Envelope)    {}

func TestChangeSubscriptionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	subscription := app.NewSubscriptionService(
		handlerTestProvider{},
		handlerTestCustomerStore{customer: port.Customer{
			UserID:                 "u1",
			ProviderCustomerID:     "cus_x",
			ProviderSubscriptionID: "sub_x",
		}},
		handlerTestBus{},
	)
	handler := NewHandler(Deps{
		Subscription: subscription,
		GetUserID: func(c *gin.Context) (string, bool) {
			return "u1", true
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/stripe/subscription/change", bytes.NewBufferString(`{"plan":"pro","interval":"monthly"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.ChangeSubscription(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateBillingPortalSessionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	subscription := app.NewSubscriptionService(
		handlerTestProvider{},
		handlerTestCustomerStore{customer: port.Customer{
			UserID:             "u1",
			ProviderCustomerID: "cus_x",
		}},
		handlerTestBus{},
	)
	handler := NewHandler(Deps{
		Subscription: subscription,
		GetUserID: func(c *gin.Context) (string, bool) {
			return "u1", true
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/stripe/portal/session", bytes.NewBufferString(`{"return_url":"https://app.test/billing"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.CreateBillingPortalSession(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestPreviewSubscriptionChangeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	subscription := app.NewSubscriptionService(
		handlerTestProvider{},
		handlerTestCustomerStore{customer: port.Customer{
			UserID:                 "u1",
			ProviderCustomerID:     "cus_x",
			ProviderSubscriptionID: "sub_x",
		}},
		handlerTestBus{},
	)
	handler := NewHandler(Deps{
		Subscription: subscription,
		GetUserID: func(c *gin.Context) (string, bool) {
			return "u1", true
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/stripe/subscription/preview", bytes.NewBufferString(`{"plan":"premium","interval":"monthly"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.PreviewSubscriptionChange(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}
