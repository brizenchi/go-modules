package billing

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	stripeintegration "github.com/brizenchi/quickstart-template/internal/integration/stripe"
	billingservice "github.com/brizenchi/quickstart-template/internal/service/billing"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StripeTopUpHandler struct {
	service *billingservice.StripeTopUpService
}

type createTopUpPaymentIntentRequest struct {
	Amount    float64           `json:"amount"`
	AmountUSD float64           `json:"amount_usd"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type stripeWebhookProbe struct {
	Type string `json:"type"`
}

func NewStripeTopUpHandler(svc *billingservice.StripeTopUpService) *StripeTopUpHandler {
	if svc == nil {
		return nil
	}
	return &StripeTopUpHandler{service: svc}
}

func (h *StripeTopUpHandler) WebhookMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.service == nil || !h.shouldInspectWebhook(c) {
			c.Next()
			return
		}

		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Request.Body = io.NopCloser(bytes.NewReader(nil))
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))

		var probe stripeWebhookProbe
		if err := json.Unmarshal(payload, &probe); err != nil || strings.TrimSpace(probe.Type) != "payment_intent.succeeded" {
			c.Next()
			return
		}

		h.handleWebhook(c, payload)
		c.Abort()
	}
}

func (h *StripeTopUpHandler) CreatePaymentIntent(c *gin.Context) {
	if h == nil || h.service == nil || !h.service.Enabled() {
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "stripe top-up not configured", nil)
		return
	}

	id := authhttp.Authenticated(c)
	if id == nil || strings.TrimSpace(id.UserID) == "" {
		httpresp.Unauthorized(c, "unauthorized")
		return
	}

	var req createTopUpPaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.BadRequest(c, "invalid request body")
		return
	}

	customer, err := h.service.Customers().LoadCustomer(c.Request.Context(), strings.TrimSpace(id.UserID))
	if err != nil {
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "failed to load billing customer", nil)
		return
	}

	prepared, err := h.service.PrepareCreatePaymentIntent(c.Request.Context(), billingservice.CreatePaymentIntentInput{
		UserID:              id.UserID,
		Email:               id.Email,
		Amount:              req.Amount,
		AmountUSD:           req.AmountUSD,
		Metadata:            req.Metadata,
		ProviderCustomerID:  customer.ProviderCustomerID,
		StoredCustomerEmail: customer.Email,
	})
	if err != nil {
		h.respondAppError(c, err)
		return
	}

	if prepared.CustomerID != "" && prepared.CustomerID != customer.ProviderCustomerID {
		if err := h.service.PersistCustomerID(c.Request.Context(), id.UserID, prepared.CustomerID); err != nil {
			httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "failed to persist stripe customer", nil)
			return
		}
	}

	intent, err := h.service.CreatePaymentIntent(c.Request.Context(), prepared, id.UserID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "stripe top-up: create payment intent failed", "error", err, "user_id", id.UserID)
		httpresp.Custom(c, http.StatusBadGateway, http.StatusBadGateway, "failed to create stripe payment intent", nil)
		return
	}
	if strings.TrimSpace(intent.ClientSecret) == "" {
		httpresp.Custom(c, http.StatusBadGateway, http.StatusBadGateway, "stripe did not return client secret", nil)
		return
	}

	slog.InfoContext(c.Request.Context(),
		"stripe top-up: payment intent created",
		"user_id", id.UserID,
		"payment_intent_id", intent.ID,
		"amount_cents", prepared.AmountCents,
		"credits", prepared.Credits,
	)

	httpresp.OK(c, gin.H{
		"payment_intent_id": intent.ID,
		"client_secret":     intent.ClientSecret,
		"amount_cents":      prepared.AmountCents,
		"amount_usd":        float64(prepared.AmountCents) / 100,
		"currency":          "usd",
		"credits":           prepared.Credits,
	})
}

func (h *StripeTopUpHandler) handleWebhook(c *gin.Context, payload []byte) {
	signature := strings.TrimSpace(c.GetHeader("Stripe-Signature"))
	if signature == "" {
		httpresp.BadRequest(c, "missing signature header")
		return
	}

	webhookEvent, handled, err := h.service.ParseWebhook(payload, signature)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	if !handled {
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		c.Next()
		return
	}

	userID, err := h.service.ResolveUserID(c.Request.Context(), &webhookEvent.Intent)
	if err != nil {
		h.respondAppError(c, err)
		return
	}

	amountCents, credits, err := h.service.ResolveCredits(&webhookEvent.Intent)
	if err != nil {
		h.respondAppError(c, err)
		return
	}

	duplicate, err := h.service.ApplyWebhook(c.Request.Context(), webhookEvent.EventID, &webhookEvent.Intent, userID, amountCents, credits, payload)
	if err != nil {
		h.respondAppError(c, err)
		return
	}

	slog.InfoContext(c.Request.Context(),
		"stripe top-up: webhook processed",
		"user_id", userID,
		"payment_intent_id", webhookEvent.Intent.ID,
		"credits", credits,
		"duplicate", duplicate,
	)

	httpresp.OK(c, gin.H{
		"received":          true,
		"id":                webhookEvent.EventID,
		"type":              webhookEvent.EventType,
		"duplicate":         duplicate,
		"payment_intent_id": webhookEvent.Intent.ID,
		"credits":           credits,
	})
}

func (h *StripeTopUpHandler) shouldInspectWebhook(c *gin.Context) bool {
	return c.Request.Method == http.MethodPost &&
		c.Request.URL.Path == stripeintegration.WebhookPath &&
		h.service != nil &&
		h.service.Enabled() &&
		strings.TrimSpace(h.service.WebhookSecret()) != ""
}

func (h *StripeTopUpHandler) respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, billingdomain.ErrProviderDisabled):
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error(), nil)
	case errors.Is(err, billingdomain.ErrSignatureInvalid):
		httpresp.BadRequest(c, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound),
		errors.Is(err, billingdomain.ErrNoBillingCustomer):
		httpresp.NotFound(c, err.Error())
	default:
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error(), nil)
	}
}
