package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingport "github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	stripeintegration "github.com/brizenchi/quickstart-template/internal/integration/stripe"
	billingrepo "github.com/brizenchi/quickstart-template/internal/repository/billing"
	"gorm.io/gorm"
)

type StripeTopUpService struct {
	cfg          stripeintegration.TopUpRuntimeConfig
	db           *gorm.DB
	users        *gormrepo.Repo
	provider     billingport.Provider
	customers    billingport.CustomerStore
	userResolver billingport.UserResolver
	events       *billingrepo.StripeTopUpEventRepository
	stripe       *stripeintegration.TopUpClient
}

type CreatePaymentIntentInput struct {
	UserID              string
	Email               string
	Amount              float64
	AmountUSD           float64
	Metadata            map[string]string
	ProviderCustomerID  string
	StoredCustomerEmail string
}

type CreatePaymentIntentResult struct {
	CustomerID    string
	AmountCents   int64
	Credits       int64
	ReceiptEmail  string
	SanitizedMeta map[string]string
}

type PaymentIntentWebhook struct {
	ID             string            `json:"id"`
	Customer       string            `json:"customer"`
	ReceiptEmail   string            `json:"receipt_email"`
	Amount         int64             `json:"amount"`
	AmountReceived int64             `json:"amount_received"`
	Metadata       map[string]string `json:"metadata"`
	Status         string            `json:"status"`
}

type ProcessWebhookResult struct {
	EventID         string
	EventType       string
	PaymentIntentID string
	UserID          string
	Credits         int64
	Duplicate       bool
}

type BillingCustomer struct {
	ProviderCustomerID string
	Email              string
}

func NewStripeTopUpService(
	cfg stripeintegration.TopUpRuntimeConfig,
	db *gorm.DB,
	users *gormrepo.Repo,
	p billingport.Provider,
	customers billingport.CustomerStore,
	resolver billingport.UserResolver,
) *StripeTopUpService {
	return &StripeTopUpService{
		cfg:          cfg,
		db:           db,
		users:        users,
		provider:     p,
		customers:    customers,
		userResolver: resolver,
		events:       billingrepo.NewStripeTopUpEventRepository(),
		stripe:       stripeintegration.NewTopUpClient(cfg.WebhookSecret),
	}
}

func (s *StripeTopUpService) Enabled() bool {
	return s != nil && s.provider != nil && s.provider.Enabled()
}

func (s *StripeTopUpService) Customers() billingport.CustomerStore {
	if s == nil {
		return nil
	}
	return s.customers
}

func (s *StripeTopUpService) WebhookSecret() string {
	if s == nil {
		return ""
	}
	return s.cfg.WebhookSecret
}

func (s *StripeTopUpService) CreditsPerUSD() int64 {
	if s == nil {
		return 0
	}
	return s.cfg.CreditsPerUSD
}

func (s *StripeTopUpService) PrepareCreatePaymentIntent(ctx context.Context, in CreatePaymentIntentInput) (*CreatePaymentIntentResult, error) {
	if !s.Enabled() {
		return nil, billingdomain.ErrProviderDisabled
	}

	amountUSD := in.Amount
	if amountUSD <= 0 {
		amountUSD = in.AmountUSD
	}
	amountCents, err := s.validateAmountUSD(amountUSD)
	if err != nil {
		return nil, err
	}

	email := strings.TrimSpace(in.Email)
	if email == "" {
		email = strings.TrimSpace(in.StoredCustomerEmail)
	}
	if email == "" {
		return nil, errors.New("email required before creating top-up payment intent")
	}

	customerID, err := s.provider.EnsureCustomer(ctx, strings.TrimSpace(in.UserID), email, strings.TrimSpace(in.ProviderCustomerID))
	if err != nil {
		return nil, err
	}

	credits := s.creditsForAmount(amountCents)
	if credits <= 0 {
		return nil, errors.New("credits mapping is not configured")
	}

	return &CreatePaymentIntentResult{
		CustomerID:    customerID,
		AmountCents:   amountCents,
		Credits:       credits,
		ReceiptEmail:  email,
		SanitizedMeta: sanitizeTopUpMetadata(in.Metadata),
	}, nil
}

func (s *StripeTopUpService) PersistCustomerID(ctx context.Context, userID, customerID string) error {
	if strings.TrimSpace(customerID) == "" {
		return nil
	}
	return s.customers.SaveCustomerID(ctx, strings.TrimSpace(userID), s.provider.Name(), strings.TrimSpace(customerID))
}

func (s *StripeTopUpService) CreatePaymentIntent(ctx context.Context, in *CreatePaymentIntentResult, userID string) (*stripeintegration.PaymentIntentCreateResult, error) {
	if in == nil {
		return nil, errors.New("stripe top-up: create payment intent input required")
	}
	return s.stripe.CreatePaymentIntent(stripeintegration.PaymentIntentCreateInput{
		CustomerID:    in.CustomerID,
		ReceiptEmail:  in.ReceiptEmail,
		UserID:        userID,
		AmountCents:   in.AmountCents,
		Credits:       in.Credits,
		CreditsPerUSD: s.cfg.CreditsPerUSD,
		Metadata:      in.SanitizedMeta,
	})
}

func (s *StripeTopUpService) ParseWebhook(payload []byte, signature string) (*stripeintegration.WebhookPaymentIntent, bool, error) {
	if !s.Enabled() {
		return nil, false, billingdomain.ErrProviderDisabled
	}
	webhookEvent, handled, err := s.stripe.ParseSucceededPaymentIntentWebhook(payload, signature)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", billingdomain.ErrSignatureInvalid, err)
	}
	return webhookEvent, handled, err
}

func (s *StripeTopUpService) ResolveUserID(ctx context.Context, intent *stripeintegration.PaymentIntentEvent) (string, error) {
	if intent == nil {
		return "", errors.New("stripe top-up: payment intent required")
	}
	userID := strings.TrimSpace(intent.Metadata["user_id"])
	if userID != "" {
		return userID, nil
	}
	if s.userResolver == nil {
		return "", errors.New("stripe top-up: user resolver not configured")
	}
	resolved, err := s.userResolver.Resolve(ctx, billingport.UserHint{
		UserID:             userID,
		Email:              strings.TrimSpace(stripeintegration.FirstNonEmpty(intent.Metadata["email"], intent.ReceiptEmail)),
		ProviderCustomerID: strings.TrimSpace(intent.Customer),
	})
	if err != nil {
		return "", fmt.Errorf("stripe top-up: resolve user: %w", err)
	}
	return strings.TrimSpace(resolved), nil
}

func (s *StripeTopUpService) ResolveCredits(intent *stripeintegration.PaymentIntentEvent) (int64, int64, error) {
	if intent == nil {
		return 0, 0, errors.New("stripe top-up: payment intent required")
	}
	amountCents := intent.AmountReceived
	if amountCents <= 0 {
		amountCents = intent.Amount
	}
	if amountCents <= 0 {
		return 0, 0, errors.New("stripe top-up: amount_cents missing from payment intent")
	}
	credits, err := s.creditsFromMetadata(intent.Metadata, amountCents)
	if err != nil {
		return 0, 0, err
	}
	return amountCents, credits, nil
}

func (s *StripeTopUpService) ApplyWebhook(
	ctx context.Context,
	providerEventID string,
	intent *stripeintegration.PaymentIntentEvent,
	userID string,
	amountCents, credits int64,
	payload []byte,
) (bool, error) {
	now := time.Now().UTC()
	duplicate := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		eventRow, err := s.events.Find(tx, providerEventID, intent.ID)
		switch {
		case err == nil:
			if eventRow.Processed {
				duplicate = true
				return nil
			}
			return s.events.Finish(tx, eventRow, userID, intent.Customer, amountCents, credits, now)
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		eventRow, err = s.events.SavePending(tx, providerEventID, intent.ID, userID, intent.Customer, amountCents, credits, payload)
		if err != nil {
			if !billingrepo.IsUniqueViolation(err) {
				return err
			}
			eventRow, err = s.events.Find(tx, providerEventID, intent.ID)
			if err != nil {
				return err
			}
			if eventRow.Processed {
				duplicate = true
				return nil
			}
		}
		return s.events.Finish(tx, eventRow, userID, intent.Customer, amountCents, credits, now)
	})

	return duplicate, err
}

func sanitizeTopUpMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		key := strings.TrimSpace(k)
		value := strings.TrimSpace(v)
		if key == "" || value == "" {
			continue
		}
		if len(out) >= stripeintegration.TopUpMaxMetadataFields {
			break
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *StripeTopUpService) creditsFromMetadata(metadata map[string]string, amountCents int64) (int64, error) {
	if raw := strings.TrimSpace(metadata["credits"]); raw != "" {
		credits, err := strconv.ParseInt(raw, 10, 64)
		if err == nil && credits > 0 {
			return credits, nil
		}
	}
	credits := s.creditsForAmount(amountCents)
	if credits <= 0 {
		return 0, errors.New("stripe top-up: invalid credits mapping")
	}
	return credits, nil
}

func (s *StripeTopUpService) creditsForAmount(amountCents int64) int64 {
	if amountCents <= 0 || s.cfg.CreditsPerUSD <= 0 {
		return 0
	}
	return int64(math.Round(float64(amountCents) / 100 * float64(s.cfg.CreditsPerUSD)))
}

func (s *StripeTopUpService) validateAmountUSD(amountUSD float64) (int64, error) {
	if amountUSD < s.cfg.MinAmountUSD {
		return 0, fmt.Errorf("minimum top-up is %.2f USD", s.cfg.MinAmountUSD)
	}
	if amountUSD > s.cfg.MaxAmountUSD {
		return 0, fmt.Errorf("maximum top-up is %.2f USD", s.cfg.MaxAmountUSD)
	}
	amountCents := int64(math.Round(amountUSD * 100))
	if amountCents <= 0 {
		return 0, errors.New("invalid top-up amount")
	}
	return amountCents, nil
}
