package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	"github.com/brizenchi/go-modules/stacks/saascore"
	"github.com/gin-gonic/gin"
	stripesdk "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/gorm"
)

const (
	defaultTopUpMinAmountUSD        = 5
	defaultTopUpMaxAmountUSD        = 10000
	defaultTopUpCreditsPerUSD int64 = 100

	stripeWebhookPath   = "/api/v1/stripe/webhook"
	stripeTopUpRoute    = "/stripe/topup/payment-intent"
	stripeTopUpType     = "credits_topup"
	stripeTopUpKind     = "custom_amount"
	maxTopUpMetadataLen = 20
)

type quickstartRouteStack struct {
	*saascore.Stack
	topUp *stripeTopUpHandler
}

func newQuickstartRouteStack(stack *saascore.Stack, cfg AppConfig) routeStack {
	return &quickstartRouteStack{
		Stack: stack,
		topUp: newStripeTopUpHandler(cfg, stack),
	}
}

func (s *quickstartRouteStack) Mount(publicGroup, userGroup *gin.RouterGroup) {
	if s.topUp != nil {
		publicGroup.Use(s.topUp.webhookMiddleware())
	}
	s.Stack.Mount(publicGroup, userGroup)
	if s.topUp != nil {
		userGroup.POST(stripeTopUpRoute, s.topUp.CreatePaymentIntent)
	}
}

type stripeTopUpRuntimeConfig struct {
	WebhookSecret string
	MinAmountUSD  float64
	MaxAmountUSD  float64
	CreditsPerUSD int64
}

type stripeTopUpHandler struct {
	cfg          stripeTopUpRuntimeConfig
	db           *gorm.DB
	users        *gormrepo.Repo
	provider     billingport.Provider
	customers    billingport.CustomerStore
	userResolver billingport.UserResolver
}

type StripeTopUpEvent struct {
	ID              uint            `gorm:"primaryKey;autoIncrement"`
	ProviderEventID string          `gorm:"type:varchar(255);uniqueIndex;not null"`
	PaymentIntentID string          `gorm:"type:varchar(255);uniqueIndex;not null"`
	UserID          string          `gorm:"type:varchar(36);index"`
	CustomerID      string          `gorm:"type:varchar(255);index"`
	AmountCents     int64           `gorm:"not null"`
	Credits         int64           `gorm:"not null"`
	Payload         json.RawMessage `gorm:"type:jsonb;not null"`
	Processed       bool            `gorm:"not null;default:false;index"`
	ProcessedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (StripeTopUpEvent) TableName() string { return "stripe_topup_events" }

type createTopUpPaymentIntentRequest struct {
	Amount    float64           `json:"amount"`
	AmountUSD float64           `json:"amount_usd"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type stripeWebhookProbe struct {
	Type string `json:"type"`
}

type stripePaymentIntentEvent struct {
	ID             string            `json:"id"`
	Customer       string            `json:"customer"`
	ReceiptEmail   string            `json:"receipt_email"`
	Amount         int64             `json:"amount"`
	AmountReceived int64             `json:"amount_received"`
	Metadata       map[string]string `json:"metadata"`
	Status         string            `json:"status"`
}

type stripeTopUpProcessResult struct {
	EventID         string
	EventType       string
	PaymentIntentID string
	UserID          string
	Credits         int64
	Duplicate       bool
}

func newStripeTopUpHandler(cfg AppConfig, stack *saascore.Stack) *stripeTopUpHandler {
	if stack == nil || stack.Billing == nil {
		return nil
	}
	return &stripeTopUpHandler{
		cfg: stripeTopUpRuntimeConfig{
			WebhookSecret: strings.TrimSpace(cfg.Billing.Stripe.WebhookSecret),
			MinAmountUSD:  positiveFloatOr(cfg.Billing.Stripe.TopUp.MinAmountUSD, defaultTopUpMinAmountUSD),
			MaxAmountUSD:  positiveFloatOr(cfg.Billing.Stripe.TopUp.MaxAmountUSD, defaultTopUpMaxAmountUSD),
			CreditsPerUSD: positiveInt64Or(cfg.Billing.Stripe.TopUp.CreditsPerUSD, defaultTopUpCreditsPerUSD),
		},
		db:           stack.DB,
		users:        stack.Users,
		provider:     stack.Billing.Provider,
		customers:    stack.Billing.Customers,
		userResolver: stack.Billing.UserResolver,
	}
}

func (h *stripeTopUpHandler) webhookMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || !h.shouldInspectWebhook(c) {
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

func (h *stripeTopUpHandler) shouldInspectWebhook(c *gin.Context) bool {
	return c.Request.Method == http.MethodPost &&
		c.Request.URL.Path == stripeWebhookPath &&
		h.provider != nil &&
		h.provider.Enabled() &&
		strings.TrimSpace(h.cfg.WebhookSecret) != ""
}

func (h *stripeTopUpHandler) CreatePaymentIntent(c *gin.Context) {
	if h == nil || h.provider == nil || !h.provider.Enabled() {
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

	amountUSD := req.Amount
	if amountUSD <= 0 {
		amountUSD = req.AmountUSD
	}
	amountCents, err := h.validateAmountUSD(amountUSD)
	if err != nil {
		httpresp.BadRequest(c, err.Error())
		return
	}

	customer, err := h.customers.LoadCustomer(c.Request.Context(), strings.TrimSpace(id.UserID))
	if err != nil {
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "failed to load billing customer", nil)
		return
	}

	email := strings.TrimSpace(id.Email)
	if email == "" {
		email = strings.TrimSpace(customer.Email)
	}
	if email == "" {
		httpresp.BadRequest(c, "email required before creating top-up payment intent")
		return
	}

	customerID, err := h.provider.EnsureCustomer(c.Request.Context(), id.UserID, email, customer.ProviderCustomerID)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	if customerID != "" && customerID != customer.ProviderCustomerID {
		if err := h.customers.SaveCustomerID(c.Request.Context(), id.UserID, h.provider.Name(), customerID); err != nil {
			httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "failed to persist stripe customer", nil)
			return
		}
	}

	credits := h.creditsForAmount(amountCents)
	if credits <= 0 {
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "credits mapping is not configured", nil)
		return
	}

	params := &stripesdk.PaymentIntentParams{
		Amount:   stripesdk.Int64(amountCents),
		Currency: stripesdk.String(string(stripesdk.CurrencyUSD)),
		AutomaticPaymentMethods: &stripesdk.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripesdk.Bool(true),
		},
		Description: stripesdk.String("Custom credits top-up"),
	}
	if customerID != "" {
		params.Customer = stripesdk.String(customerID)
	} else {
		params.ReceiptEmail = stripesdk.String(email)
	}

	for k, v := range sanitizeTopUpMetadata(req.Metadata) {
		params.AddMetadata(k, v)
	}
	params.AddMetadata("user_id", strings.TrimSpace(id.UserID))
	params.AddMetadata("email", email)
	params.AddMetadata("product_type", stripeTopUpType)
	params.AddMetadata("topup_kind", stripeTopUpKind)
	params.AddMetadata("amount_cents", strconv.FormatInt(amountCents, 10))
	params.AddMetadata("credits", strconv.FormatInt(credits, 10))
	params.AddMetadata("credits_per_usd", strconv.FormatInt(h.cfg.CreditsPerUSD, 10))

	intent, err := paymentintent.New(params)
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
		"amount_cents", amountCents,
		"credits", credits,
	)

	httpresp.OK(c, gin.H{
		"payment_intent_id": intent.ID,
		"client_secret":     intent.ClientSecret,
		"amount_cents":      amountCents,
		"amount_usd":        centsToUSD(amountCents),
		"currency":          "usd",
		"credits":           credits,
	})
}

func (h *stripeTopUpHandler) handleWebhook(c *gin.Context, payload []byte) {
	signature := strings.TrimSpace(c.GetHeader("Stripe-Signature"))
	if signature == "" {
		httpresp.BadRequest(c, "missing signature header")
		return
	}

	result, handled, err := h.processWebhook(c.Request.Context(), payload, signature)
	if err != nil {
		h.respondAppError(c, err)
		return
	}
	if !handled {
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		c.Next()
		return
	}

	httpresp.OK(c, gin.H{
		"received":          true,
		"id":                result.EventID,
		"type":              result.EventType,
		"duplicate":         result.Duplicate,
		"payment_intent_id": result.PaymentIntentID,
		"credits":           result.Credits,
	})
}

func (h *stripeTopUpHandler) processWebhook(ctx context.Context, payload []byte, signature string) (*stripeTopUpProcessResult, bool, error) {
	if h == nil || h.provider == nil || !h.provider.Enabled() {
		return nil, false, billingdomain.ErrProviderDisabled
	}

	ev, err := webhook.ConstructEventWithOptions(
		payload,
		signature,
		h.cfg.WebhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", billingdomain.ErrSignatureInvalid, err)
	}
	if string(ev.Type) != "payment_intent.succeeded" {
		return nil, false, nil
	}

	var intent stripePaymentIntentEvent
	if err := json.Unmarshal(ev.Data.Raw, &intent); err != nil {
		return nil, false, fmt.Errorf("stripe top-up: parse payment intent payload: %w", err)
	}
	if !isCustomTopUpMetadata(intent.Metadata) {
		return nil, false, nil
	}

	resolvedUserID, err := h.resolveUserID(ctx, intent)
	if err != nil {
		return nil, true, err
	}

	amountCents := intent.AmountReceived
	if amountCents <= 0 {
		amountCents = intent.Amount
	}
	if amountCents <= 0 {
		return nil, true, errors.New("stripe top-up: amount_cents missing from payment intent")
	}

	credits, err := h.creditsFromMetadata(intent.Metadata, amountCents)
	if err != nil {
		return nil, true, err
	}

	duplicate, err := h.applyTopUp(ctx, ev.ID, intent, resolvedUserID, amountCents, credits, payload)
	if err != nil {
		return nil, true, err
	}

	slog.InfoContext(ctx,
		"stripe top-up: webhook processed",
		"user_id", resolvedUserID,
		"payment_intent_id", intent.ID,
		"credits", credits,
		"duplicate", duplicate,
	)

	return &stripeTopUpProcessResult{
		EventID:         ev.ID,
		EventType:       string(ev.Type),
		PaymentIntentID: intent.ID,
		UserID:          resolvedUserID,
		Credits:         credits,
		Duplicate:       duplicate,
	}, true, nil
}

func (h *stripeTopUpHandler) resolveUserID(ctx context.Context, intent stripePaymentIntentEvent) (string, error) {
	userID := strings.TrimSpace(intent.Metadata["user_id"])
	if userID != "" {
		return userID, nil
	}
	if h.userResolver == nil {
		return "", errors.New("stripe top-up: user resolver not configured")
	}
	resolved, err := h.userResolver.Resolve(ctx, billingport.UserHint{
		UserID:             userID,
		Email:              strings.TrimSpace(firstNonEmpty(intent.Metadata["email"], intent.ReceiptEmail)),
		ProviderCustomerID: strings.TrimSpace(intent.Customer),
	})
	if err != nil {
		return "", fmt.Errorf("stripe top-up: resolve user: %w", err)
	}
	return strings.TrimSpace(resolved), nil
}

func (h *stripeTopUpHandler) applyTopUp(
	ctx context.Context,
	providerEventID string,
	intent stripePaymentIntentEvent,
	userID string,
	amountCents int64,
	credits int64,
	payload []byte,
) (bool, error) {
	now := time.Now().UTC()
	duplicate := false

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		eventRow, err := findStripeTopUpEvent(tx, providerEventID, intent.ID)
		switch {
		case err == nil:
			if eventRow.Processed {
				duplicate = true
				return nil
			}
			return finishStripeTopUpTx(tx, eventRow, userID, intent.Customer, amountCents, credits, now)
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		eventRow = &StripeTopUpEvent{
			ProviderEventID: providerEventID,
			PaymentIntentID: strings.TrimSpace(intent.ID),
			UserID:          strings.TrimSpace(userID),
			CustomerID:      strings.TrimSpace(intent.Customer),
			AmountCents:     amountCents,
			Credits:         credits,
			Payload:         append(json.RawMessage(nil), payload...),
		}
		if err := tx.Create(eventRow).Error; err != nil {
			if !isUniqueViolation(err) {
				return err
			}
			eventRow, err = findStripeTopUpEvent(tx, providerEventID, intent.ID)
			if err != nil {
				return err
			}
			if eventRow.Processed {
				duplicate = true
				return nil
			}
		}
		return finishStripeTopUpTx(tx, eventRow, userID, intent.Customer, amountCents, credits, now)
	})
	return duplicate, err
}

func findStripeTopUpEvent(tx *gorm.DB, providerEventID, paymentIntentID string) (*StripeTopUpEvent, error) {
	var eventRow StripeTopUpEvent
	err := tx.
		Where("provider_event_id = ? OR payment_intent_id = ?", strings.TrimSpace(providerEventID), strings.TrimSpace(paymentIntentID)).
		First(&eventRow).
		Error
	if err != nil {
		return nil, err
	}
	return &eventRow, nil
}

func finishStripeTopUpTx(
	tx *gorm.DB,
	eventRow *StripeTopUpEvent,
	userID, customerID string,
	amountCents, credits int64,
	now time.Time,
) error {
	if eventRow == nil {
		return errors.New("stripe top-up: event row required")
	}
	if strings.TrimSpace(userID) == "" {
		return errors.New("stripe top-up: user_id required")
	}

	updates := map[string]any{}
	if strings.TrimSpace(customerID) != "" {
		updates["stripe_customer_id"] = strings.TrimSpace(customerID)
	}
	if len(updates) > 0 {
		if err := tx.Model(&gormrepo.UserRow{}).
			Where("id = ?", strings.TrimSpace(userID)).
			Updates(updates).Error; err != nil {
			return err
		}
	}

	res := tx.Model(&gormrepo.UserRow{}).
		Where("id = ?", strings.TrimSpace(userID)).
		UpdateColumn("credits", gorm.Expr("credits + ?", int(credits)))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return tx.Model(eventRow).
		Updates(map[string]any{
			"user_id":      strings.TrimSpace(userID),
			"customer_id":  strings.TrimSpace(customerID),
			"amount_cents": amountCents,
			"credits":      credits,
			"processed":    true,
			"processed_at": &now,
		}).
		Error
}

func sanitizeTopUpMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" || isReservedTopUpMetadataKey(k) {
			continue
		}
		out[k] = v
		if len(out) >= maxTopUpMetadataLen {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isReservedTopUpMetadataKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "user_id", "email", "product_type", "topup_kind", "amount_cents", "credits", "credits_per_usd":
		return true
	default:
		return false
	}
}

func isCustomTopUpMetadata(metadata map[string]string) bool {
	if len(metadata) == 0 {
		return false
	}
	return strings.TrimSpace(metadata["product_type"]) == stripeTopUpType &&
		strings.TrimSpace(metadata["topup_kind"]) == stripeTopUpKind
}

func (h *stripeTopUpHandler) validateAmountUSD(amount float64) (int64, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, errors.New("amount must be a real number")
	}
	amount = math.Round(amount*100) / 100
	if amount <= 0 {
		return 0, errors.New("amount must be greater than 0")
	}
	if amount < h.cfg.MinAmountUSD {
		return 0, fmt.Errorf("amount must be at least %.2f USD", h.cfg.MinAmountUSD)
	}
	if amount > h.cfg.MaxAmountUSD {
		return 0, fmt.Errorf("amount must be at most %.2f USD", h.cfg.MaxAmountUSD)
	}
	return usdToCents(amount), nil
}

func (h *stripeTopUpHandler) creditsForAmount(amountCents int64) int64 {
	if amountCents <= 0 || h.cfg.CreditsPerUSD <= 0 {
		return 0
	}
	return int64(math.Round(float64(amountCents) * float64(h.cfg.CreditsPerUSD) / 100))
}

func (h *stripeTopUpHandler) creditsFromMetadata(metadata map[string]string, amountCents int64) (int64, error) {
	if raw := strings.TrimSpace(metadata["credits"]); raw != "" {
		credits, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || credits <= 0 {
			return 0, errors.New("stripe top-up: invalid credits metadata")
		}
		return credits, nil
	}
	credits := h.creditsForAmount(amountCents)
	if credits <= 0 {
		return 0, errors.New("stripe top-up: credits mapping is not configured")
	}
	return credits, nil
}

func (h *stripeTopUpHandler) respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, billingdomain.ErrProviderDisabled):
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error(), nil)
	case errors.Is(err, billingdomain.ErrSignatureInvalid):
		httpresp.BadRequest(c, "invalid signature")
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpresp.NotFound(c, "user not found")
	default:
		slog.ErrorContext(c.Request.Context(), "stripe top-up: request failed", "error", err)
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error(), nil)
	}
}

func usdToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func centsToUSD(amountCents int64) float64 {
	return float64(amountCents) / 100
}

func positiveFloatOr(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func positiveInt64Or(value, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "unique") ||
		strings.Contains(text, "duplicate") ||
		strings.Contains(text, "23505")
}
