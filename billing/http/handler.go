// Package http exposes Gin handlers for the billing module.
//
// Handlers depend only on app use cases. Auth (extracting userID from
// the request) is delegated to a UserIDFunc supplied by the host so
// the module stays compatible with any auth scheme.
package http

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/brizenchi/go-modules/billing/app"
	"github.com/brizenchi/go-modules/billing/domain"
	"github.com/gin-gonic/gin"
)

// UserIDFunc resolves the authenticated user ID from a Gin context.
// Hosts wire this to their auth middleware.
type UserIDFunc func(c *gin.Context) (userID string, ok bool)

// Handler bundles billing HTTP endpoints.
type Handler struct {
	checkout     *app.CheckoutService
	subscription *app.SubscriptionService
	webhook      *app.WebhookService
	query        *app.QueryService
	getUserID    UserIDFunc
}

// Deps gathers the constructor dependencies.
type Deps struct {
	Checkout     *app.CheckoutService
	Subscription *app.SubscriptionService
	Webhook      *app.WebhookService
	Query        *app.QueryService
	GetUserID    UserIDFunc
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		checkout:     deps.Checkout,
		subscription: deps.Subscription,
		webhook:      deps.Webhook,
		query:        deps.Query,
		getUserID:    deps.GetUserID,
	}
}

// --- Webhook (public) ----------------------------------------------------

// HandleWebhook is the provider webhook endpoint. Mount on a public route.
func (h *Handler) HandleWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "failed to read webhook payload")
		return
	}
	if len(payload) == 0 {
		respondError(c, http.StatusBadRequest, "webhook body required")
		return
	}
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		respondError(c, http.StatusBadRequest, "missing signature header")
		return
	}

	res, err := h.webhook.Process(c.Request.Context(), payload, signature)
	if err != nil {
		if app.IsSignatureError(err) {
			respondError(c, http.StatusBadRequest, "invalid signature")
			return
		}
		slog.Error("billing: webhook processing failed", "error", err)
		respondError(c, http.StatusInternalServerError, "event handling failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"received":  true,
		"id":        res.ProviderEventID,
		"type":      res.Type,
		"duplicate": res.Duplicate,
	})
}

// --- Checkout (authenticated) -------------------------------------------

type createCheckoutRequest struct {
	Plan        string            `json:"plan"`
	Interval    string            `json:"interval"`
	ProductType string            `json:"product_type"`
	PriceID     string            `json:"price_id"`
	Quantity    int64             `json:"quantity"`
	SuccessURL  string            `json:"success_url"`
	CancelURL   string            `json:"cancel_url"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateCheckoutSession opens a hosted checkout session.
func (h *Handler) CreateCheckoutSession(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req createCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	in, err := buildCheckoutInput(userID, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.checkout.Create(c.Request.Context(), in)
	if err != nil {
		respondAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":   res.SessionID,
		"checkout_url": res.CheckoutURL,
	})
}

// maxMetadataEntries caps client-supplied metadata size to keep request
// payloads bounded. Stripe itself allows 50 keys per object.
const maxMetadataEntries = 20

// sanitizeMetadata trims keys/values, drops empty pairs and reserved keys,
// and caps the size. Reserved keys are populated by the billing layer
// itself — silently ignoring them prevents request bodies from spoofing
// system fields like user_id.
func sanitizeMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		if domain.IsReservedMetadataKey(k) {
			continue
		}
		out[k] = v
		if len(out) >= maxMetadataEntries {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildCheckoutInput(userID string, req createCheckoutRequest) (app.CheckoutInput, error) {
	in := app.CheckoutInput{
		UserID:     userID,
		PriceID:    strings.TrimSpace(req.PriceID),
		Quantity:   req.Quantity,
		SuccessURL: strings.TrimSpace(req.SuccessURL),
		CancelURL:  strings.TrimSpace(req.CancelURL),
		Metadata:   sanitizeMetadata(req.Metadata),
	}

	productType := strings.ToLower(strings.TrimSpace(req.ProductType))
	if productType == "" {
		productType = string(domain.ProductSubscription)
	}
	switch productType {
	case string(domain.ProductSubscription):
		in.ProductType = domain.ProductSubscription
	case string(domain.ProductCredits):
		in.ProductType = domain.ProductCredits
	default:
		return in, errors.New("product_type must be subscription or credits")
	}

	if in.ProductType == domain.ProductSubscription {
		plan := domain.PlanType(strings.ToLower(strings.TrimSpace(req.Plan)))
		if !plan.Valid() || plan == domain.PlanFree {
			return in, errors.New("plan must be starter, pro, or premium")
		}
		in.Plan = plan
		switch strings.ToLower(strings.TrimSpace(req.Interval)) {
		case "monthly", "month":
			in.Interval = domain.IntervalMonthly
		case "yearly", "year":
			in.Interval = domain.IntervalYearly
		default:
			return in, errors.New("interval must be monthly or yearly")
		}
	} else {
		if in.Quantity < 0 {
			return in, errors.New("quantity must be non-negative")
		}
		if in.Quantity == 0 {
			in.Quantity = 1
		}
		if in.Quantity > 100 {
			return in, errors.New("quantity too large")
		}
	}
	return in, nil
}

// --- Subscription mutations (authenticated) -----------------------------

type cancelSubscriptionRequest struct {
	CancelType string `json:"cancel_type"`
}

// CancelSubscription schedules cancellation.
func (h *Handler) CancelSubscription(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req cancelSubscriptionRequest
	_ = c.ShouldBindJSON(&req) // body is optional
	mode := domain.CancelMode(strings.TrimSpace(req.CancelType))
	if mode == "" {
		mode = domain.CancelAtPeriodEnd
	}

	res, err := h.subscription.Cancel(c.Request.Context(), userID, mode)
	if err != nil {
		respondAppError(c, err)
		return
	}
	msg := "subscription will be cancelled at the end of the billing period"
	if mode == domain.CancelIn3Days {
		msg = "subscription will be cancelled in 3 days"
	}
	resp := gin.H{
		"status":      "cancelling",
		"cancel_type": string(mode),
		"message":     msg,
	}
	if res.EffectiveAt != nil {
		resp["effective_cancel_at"] = res.EffectiveAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	c.JSON(http.StatusOK, resp)
}

// ReactivateSubscription clears a pending cancellation.
func (h *Handler) ReactivateSubscription(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	if _, err := h.subscription.Reactivate(c.Request.Context(), userID); err != nil {
		respondAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "active",
		"message": "subscription has been reactivated",
	})
}

// --- Queries (authenticated) --------------------------------------------

// GetSubscription returns the current subscription view.
func (h *Handler) GetSubscription(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	view, err := h.query.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		respondAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

// ListInvoices returns paginated invoices.
func (h *Handler) ListInvoices(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	items, total, err := h.query.ListInvoices(c.Request.Context(), userID, page, limit)
	if err != nil {
		respondAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// --- helpers ------------------------------------------------------------

func (h *Handler) userID(c *gin.Context) (string, bool) {
	if h.getUserID == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	id, ok := h.getUserID(c)
	if !ok || id == "" {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return id, true
}

func respondError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

func respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidPriceID),
		errors.Is(err, domain.ErrInvalidCancelMode),
		errors.Is(err, domain.ErrPriceNotFound),
		errors.Is(err, domain.ErrNoActiveSubscription),
		errors.Is(err, domain.ErrNoSubscriptionToReactive):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrProviderDisabled):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		slog.Error("billing: internal error", "error", err)
		respondError(c, http.StatusInternalServerError, err.Error())
	}
}
