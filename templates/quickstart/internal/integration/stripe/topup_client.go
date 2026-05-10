package stripe

import (
	"encoding/json"
	"fmt"
	"strings"

	stripesdk "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/webhook"
)

const (
	WebhookPath               = "/api/v1/stripe/webhook"
	TopUpPaymentIntentPath    = "/stripe/topup/payment-intent"
	DefaultTopUpMinAmountUSD  = 5
	DefaultTopUpMaxAmountUSD  = 10000
	DefaultTopUpCreditsPerUSD = int64(100)
	TopUpProductType          = "credits_topup"
	TopUpKind                 = "custom_amount"
	TopUpMaxMetadataFields    = 20
)

type TopUpRuntimeConfig struct {
	WebhookSecret string
	MinAmountUSD  float64
	MaxAmountUSD  float64
	CreditsPerUSD int64
}

type PaymentIntentCreateInput struct {
	CustomerID    string
	ReceiptEmail  string
	UserID        string
	AmountCents   int64
	Credits       int64
	CreditsPerUSD int64
	Metadata      map[string]string
}

type PaymentIntentCreateResult struct {
	ID           string
	ClientSecret string
}

type PaymentIntentEvent struct {
	ID             string            `json:"id"`
	Customer       string            `json:"customer"`
	ReceiptEmail   string            `json:"receipt_email"`
	Amount         int64             `json:"amount"`
	AmountReceived int64             `json:"amount_received"`
	Metadata       map[string]string `json:"metadata"`
	Status         string            `json:"status"`
}

type WebhookPaymentIntent struct {
	EventID   string
	EventType string
	Intent    PaymentIntentEvent
}

type TopUpClient struct {
	webhookSecret string
}

func NewTopUpClient(webhookSecret string) *TopUpClient {
	return &TopUpClient{webhookSecret: strings.TrimSpace(webhookSecret)}
}

func (p *TopUpClient) CreatePaymentIntent(in PaymentIntentCreateInput) (*PaymentIntentCreateResult, error) {
	params := &stripesdk.PaymentIntentParams{
		Amount:   stripesdk.Int64(in.AmountCents),
		Currency: stripesdk.String(string(stripesdk.CurrencyUSD)),
		AutomaticPaymentMethods: &stripesdk.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripesdk.Bool(true),
		},
		Description: stripesdk.String("Custom credits top-up"),
	}
	if strings.TrimSpace(in.CustomerID) != "" {
		params.Customer = stripesdk.String(strings.TrimSpace(in.CustomerID))
	} else {
		params.ReceiptEmail = stripesdk.String(strings.TrimSpace(in.ReceiptEmail))
	}

	for k, v := range in.Metadata {
		params.AddMetadata(k, v)
	}
	params.AddMetadata("user_id", strings.TrimSpace(in.UserID))
	params.AddMetadata("email", strings.TrimSpace(in.ReceiptEmail))
	params.AddMetadata("product_type", TopUpProductType)
	params.AddMetadata("topup_kind", TopUpKind)
	params.AddMetadata("amount_cents", fmt.Sprintf("%d", in.AmountCents))
	params.AddMetadata("credits", fmt.Sprintf("%d", in.Credits))
	params.AddMetadata("credits_per_usd", fmt.Sprintf("%d", in.CreditsPerUSD))

	intent, err := paymentintent.New(params)
	if err != nil {
		return nil, err
	}
	return &PaymentIntentCreateResult{
		ID:           intent.ID,
		ClientSecret: intent.ClientSecret,
	}, nil
}

func (p *TopUpClient) ParseSucceededPaymentIntentWebhook(payload []byte, signature string) (*WebhookPaymentIntent, bool, error) {
	ev, err := webhook.ConstructEventWithOptions(
		payload,
		strings.TrimSpace(signature),
		p.webhookSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		return nil, false, err
	}
	if string(ev.Type) != "payment_intent.succeeded" {
		return nil, false, nil
	}

	var intent PaymentIntentEvent
	if err := json.Unmarshal(ev.Data.Raw, &intent); err != nil {
		return nil, false, err
	}
	if !isCustomTopUpMetadata(intent.Metadata) {
		return nil, false, nil
	}

	return &WebhookPaymentIntent{
		EventID:   ev.ID,
		EventType: string(ev.Type),
		Intent:    intent,
	}, true, nil
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func PositiveFloatOr(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}

func PositiveInt64Or(value, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func isCustomTopUpMetadata(metadata map[string]string) bool {
	return strings.TrimSpace(metadata["product_type"]) == TopUpProductType &&
		strings.TrimSpace(metadata["topup_kind"]) == TopUpKind
}
