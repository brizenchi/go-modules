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
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/modules/billing/app"
	"github.com/brizenchi/go-modules/modules/billing/domain"
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
		slog.ErrorContext(c.Request.Context(), "billing: webhook processing failed", "error", err)
		respondError(c, http.StatusInternalServerError, "event handling failed")
		return
	}
	httpresp.OK(c, gin.H{
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
	httpresp.OK(c, gin.H{
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
	case string(domain.ProductLifetime):
		in.ProductType = domain.ProductLifetime
	default:
		return in, errors.New("product_type must be subscription, credits, or lifetime")
	}

	if in.ProductType == domain.ProductSubscription {
		plan := domain.PlanType(strings.ToLower(strings.TrimSpace(req.Plan)))
		if !plan.Valid() || plan == domain.PlanFree || plan == domain.PlanLifetime {
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
	} else if in.ProductType == domain.ProductCredits {
		if in.Quantity < 0 {
			return in, errors.New("quantity must be non-negative")
		}
		if in.Quantity == 0 {
			in.Quantity = 1
		}
		if in.Quantity > 100 {
			return in, errors.New("quantity too large")
		}
	} else {
		in.Plan = domain.PlanLifetime
		in.Quantity = 1
	}
	return in, nil
}

// --- Subscription mutations (authenticated) -----------------------------

type cancelSubscriptionRequest struct {
	CancelType string `json:"cancel_type"`
}

type changeSubscriptionRequest struct {
	Plan       string `json:"plan"`
	Interval   string `json:"interval"`
	ChangeMode string `json:"change_mode"`
}

type previewSubscriptionChangeRequest struct {
	Plan       string `json:"plan"`
	Interval   string `json:"interval"`
	ChangeMode string `json:"change_mode"`
}

type portalSessionRequest struct {
	ReturnURL string `json:"return_url"`
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
	httpresp.OK(c, resp)
}

// ChangeSubscription changes the active paid plan in-place.
func (h *Handler) ChangeSubscription(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req changeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := domain.PlanType(strings.ToLower(strings.TrimSpace(req.Plan)))
	if !plan.Valid() || plan == domain.PlanFree {
		respondError(c, http.StatusBadRequest, "plan must be starter, pro, or premium")
		return
	}

	var interval domain.BillingInterval
	switch strings.ToLower(strings.TrimSpace(req.Interval)) {
	case "monthly", "month":
		interval = domain.IntervalMonthly
	case "yearly", "year":
		interval = domain.IntervalYearly
	default:
		respondError(c, http.StatusBadRequest, "interval must be monthly or yearly")
		return
	}

	mode := domain.SubscriptionChangeMode(strings.TrimSpace(req.ChangeMode))
	if req.ChangeMode != "" && !mode.Valid() {
		respondError(c, http.StatusBadRequest, "change_mode must be immediate_prorated, immediate_reset_cycle, or period_end")
		return
	}

	res, err := h.subscription.Change(c.Request.Context(), userID, domain.SubscriptionChangeInput{
		Plan:     plan,
		Interval: interval,
		Mode:     mode,
	})
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{
		"status":                   res.Snapshot.Status,
		"plan":                     res.Snapshot.Plan,
		"billing_cycle":            res.Snapshot.Interval,
		"change_mode":              res.Mode,
		"provider_subscription_id": res.ProviderSubscriptionID,
		"message":                  "subscription changed",
	})
}

// PreviewSubscriptionChange previews how a plan change will be billed.
func (h *Handler) PreviewSubscriptionChange(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req previewSubscriptionChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	plan := domain.PlanType(strings.ToLower(strings.TrimSpace(req.Plan)))
	if !plan.Valid() || plan == domain.PlanFree {
		respondError(c, http.StatusBadRequest, "plan must be starter, pro, or premium")
		return
	}

	var interval domain.BillingInterval
	switch strings.ToLower(strings.TrimSpace(req.Interval)) {
	case "monthly", "month":
		interval = domain.IntervalMonthly
	case "yearly", "year":
		interval = domain.IntervalYearly
	default:
		respondError(c, http.StatusBadRequest, "interval must be monthly or yearly")
		return
	}

	mode := domain.SubscriptionChangeMode(strings.TrimSpace(req.ChangeMode))
	if req.ChangeMode != "" && !mode.Valid() {
		respondError(c, http.StatusBadRequest, "change_mode must be immediate_prorated, immediate_reset_cycle, or period_end")
		return
	}

	res, err := h.subscription.PreviewChange(c.Request.Context(), userID, domain.SubscriptionPreviewInput{
		Plan:     plan,
		Interval: interval,
		Mode:     mode,
	})
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{
		"currency":                res.Currency,
		"amount_due_now":          res.AmountDueNow,
		"current_period_end":      formatTimePtr(res.CurrentPeriodEnd),
		"next_billing_at":         formatTimePtr(res.NextBillingAt),
		"target_plan":             res.TargetPlan,
		"target_interval":         res.TargetInterval,
		"change_mode":             res.Mode,
		"immediate_charge":        res.ImmediateCharge,
		"effective_at_period_end": res.EffectiveAtPeriodEnd,
		"message":                 res.Message,
	})
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
	httpresp.OK(c, gin.H{
		"status":  "active",
		"message": "subscription has been reactivated",
	})
}

// CreateBillingPortalSession opens Stripe's hosted customer portal.
func (h *Handler) CreateBillingPortalSession(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	var req portalSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.subscription.OpenBillingPortal(c.Request.Context(), userID, req.ReturnURL)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{
		"url": res.URL,
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
	httpresp.OK(c, view)
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
	httpresp.OK(c, gin.H{
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
	switch status {
	case http.StatusBadRequest:
		httpresp.BadRequest(c, msg)
	case http.StatusUnauthorized:
		httpresp.Unauthorized(c, msg)
	case http.StatusServiceUnavailable:
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, msg, nil)
	default:
		httpresp.Custom(c, status, status, msg, nil)
	}
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02T15:04:05Z07:00")
}

func respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidPriceID),
		errors.Is(err, domain.ErrInvalidCancelMode),
		errors.Is(err, domain.ErrPriceNotFound),
		errors.Is(err, domain.ErrNoBillingCustomer),
		errors.Is(err, domain.ErrNoActiveSubscription),
		errors.Is(err, domain.ErrNoSubscriptionToReactive):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrProviderDisabled):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "billing: internal error", "error", err)
		respondError(c, http.StatusInternalServerError, err.Error())
	}
}
