package saascore

import "time"

type Config struct {
	ServiceName string

	Auth     AuthConfig
	Email    EmailConfig
	Billing  BillingConfig
	Referral ReferralConfig
}

type AuthConfig struct {
	FrontendRedirect string
	UserJWTSecret    string
	UserJWTExpire    time.Duration
	WSTicketTTL      time.Duration
	AdminEmails      []string

	EmailCode EmailCodeConfig
	Google    GoogleOAuthConfig
}

type EmailCodeConfig struct {
	Debug                bool
	VerificationTemplate string
	TTL                  time.Duration
	MinResendGap         time.Duration
	DailyCap             int
	MaxAttempts          int
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	StateSecret  string
	Scope        string
}

type EmailConfig struct {
	Provider string

	Brevo  BrevoConfig
	Resend ResendConfig
}

type BrevoConfig struct {
	APIKey      string
	SenderEmail string
	SenderName  string
}

type ResendConfig struct {
	APIKey      string
	SenderEmail string
	SenderName  string
}

type BillingConfig struct {
	Stripe StripeConfig
}

type StripeConfig struct {
	Enabled        bool
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
	TrialDays      int64

	StarterMonthlyPriceID string
	StarterYearlyPriceID  string
	ProMonthlyPriceID     string
	ProYearlyPriceID      string

	CreditsPriceIDs   []string
	CreditsPerPackage int64
}

type ReferralConfig struct {
	Prefix           string
	BaseLink         string
	ActivationReward int
	ActivationWindow time.Duration
}
