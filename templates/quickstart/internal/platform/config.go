// Package platform is the copied application's composition layer.
// It chooses adapters and module combinations; shared modules never import it.
package platform

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	Auth        AuthConfig
	Email       EmailConfig
	Billing     BillingConfig
	Referral    ReferralConfig
}

type AuthConfig struct {
	Enabled            *bool    `mapstructure:"enabled"`
	UserJWTSecret      string   `mapstructure:"user_jwt_secret"`
	UserJWTExpireHours int      `mapstructure:"user_jwt_expire_hours"`
	WSTicketTTLSeconds int      `mapstructure:"ws_ticket_ttl_seconds"`
	AdminEmails        []string `mapstructure:"admin_emails"`
	FrontendRedirect   string   `mapstructure:"frontend_redirect"`
	// OAuthCookieSecure may be explicitly set for unusual proxy setups. When
	// nil, it is derived from the enabled providers' callback URL schemes.
	OAuthCookieSecure *bool           `mapstructure:"oauth_cookie_secure"`
	Email             AuthEmailConfig `mapstructure:"email"`
	Google            GoogleConfig    `mapstructure:"google"`
	GitHub            GitHubConfig    `mapstructure:"github"`
}

type AuthEmailConfig struct {
	Enabled *bool               `mapstructure:"enabled"`
	Debug   bool                `mapstructure:"debug"`
	Code    AuthEmailCodeConfig `mapstructure:"code"`
}

type AuthEmailCodeConfig struct {
	TTLMinutes          int `mapstructure:"ttl_minutes"`
	MinResendGapSeconds int `mapstructure:"min_resend_gap_seconds"`
	DailyCap            int `mapstructure:"daily_cap"`
	MaxAttempts         int `mapstructure:"max_attempts"`
}

type GoogleConfig struct {
	Enabled      *bool  `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	StateSecret  string `mapstructure:"state_secret"`
	StateTTLMin  int    `mapstructure:"state_ttl_minutes"`
	Scope        string `mapstructure:"scope"`
}

type GitHubConfig struct {
	Enabled      *bool  `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	StateSecret  string `mapstructure:"state_secret"`
	StateTTLMin  int    `mapstructure:"state_ttl_minutes"`
	Scope        string `mapstructure:"scope"`
}

type EmailConfig struct {
	// Provider: none | log | resend | brevo.
	Provider string       `mapstructure:"provider"`
	Brevo    BrevoConfig  `mapstructure:"brevo"`
	Resend   ResendConfig `mapstructure:"resend"`
}

type BrevoConfig struct {
	APIKey      string `mapstructure:"api_key"`
	SenderEmail string `mapstructure:"sender_email"`
	SenderName  string `mapstructure:"sender_name"`
}

type ResendConfig struct {
	APIKey      string `mapstructure:"api_key"`
	SenderEmail string `mapstructure:"sender_email"`
	SenderName  string `mapstructure:"sender_name"`
}

type BillingConfig struct {
	Enabled  *bool        `mapstructure:"enabled"`
	Provider string       `mapstructure:"provider"`
	Stripe   StripeConfig `mapstructure:"stripe"`
}

type StripeConfig struct {
	SecretKey      string              `mapstructure:"secret_key"`
	PublishableKey string              `mapstructure:"publishable_key"`
	WebhookSecret  string              `mapstructure:"webhook_secret"`
	TrialDays      int64               `mapstructure:"trial_days"`
	Prices         StripePricesConfig  `mapstructure:"prices"`
	Credits        StripeCreditsConfig `mapstructure:"credits"`
}

type StripePricesConfig struct {
	StarterMonthly string   `mapstructure:"starter_monthly"`
	StarterYearly  string   `mapstructure:"starter_yearly"`
	ProMonthly     string   `mapstructure:"pro_monthly"`
	ProYearly      string   `mapstructure:"pro_yearly"`
	PremiumMonthly string   `mapstructure:"premium_monthly"`
	PremiumYearly  string   `mapstructure:"premium_yearly"`
	Lifetime       string   `mapstructure:"lifetime"`
	Credits        []string `mapstructure:"credits"`
}

type StripeCreditsConfig struct {
	PerPackage int64 `mapstructure:"per_package"`
}

type ReferralConfig struct {
	Enabled              *bool  `mapstructure:"enabled"`
	Prefix               string `mapstructure:"prefix"`
	BaseLink             string `mapstructure:"base_link"`
	ActivationReward     int    `mapstructure:"activation_reward"`
	ActivationWindowDays int    `mapstructure:"activation_window_days"`
}

func (c Config) withDefaults() Config {
	if c.ServiceName == "" {
		c.ServiceName = "quickstart"
	}
	if c.Auth.UserJWTExpireHours <= 0 {
		c.Auth.UserJWTExpireHours = 168
	}
	if c.Auth.WSTicketTTLSeconds <= 0 {
		c.Auth.WSTicketTTLSeconds = 300
	}
	if c.Auth.Email.Code.TTLMinutes <= 0 {
		c.Auth.Email.Code.TTLMinutes = 10
	}
	if c.Auth.Email.Code.MinResendGapSeconds <= 0 {
		c.Auth.Email.Code.MinResendGapSeconds = 60
	}
	if c.Auth.Email.Code.DailyCap <= 0 {
		c.Auth.Email.Code.DailyCap = 10
	}
	if c.Auth.Email.Code.MaxAttempts <= 0 {
		c.Auth.Email.Code.MaxAttempts = 5
	}
	if c.Auth.Google.StateTTLMin <= 0 {
		c.Auth.Google.StateTTLMin = 20
	}
	if c.Auth.GitHub.StateTTLMin <= 0 {
		c.Auth.GitHub.StateTTLMin = 20
	}
	if c.Email.Provider == "" {
		c.Email.Provider = "log"
	}
	if c.Billing.Provider == "" {
		c.Billing.Provider = "stripe"
	}
	if c.Billing.Stripe.Credits.PerPackage <= 0 {
		c.Billing.Stripe.Credits.PerPackage = 100
	}
	if c.Referral.Prefix == "" {
		c.Referral.Prefix = "INV"
	}
	return c
}

func boolDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func (c Config) AuthEnabled() bool { return boolDefault(c.Auth.Enabled, true) }

func (c Config) EmailAuthEnabled() bool {
	return c.AuthEnabled() && boolDefault(c.Auth.Email.Enabled, true)
}

func (c Config) GoogleEnabled() bool {
	configured := c.Auth.Google.ClientID != "" || c.Auth.Google.ClientSecret != ""
	return c.AuthEnabled() && boolDefault(c.Auth.Google.Enabled, configured)
}

func (c Config) GitHubEnabled() bool {
	configured := c.Auth.GitHub.ClientID != "" || c.Auth.GitHub.ClientSecret != ""
	return c.AuthEnabled() && boolDefault(c.Auth.GitHub.Enabled, configured)
}

// OAuthFlowCookieSecure derives the safe cookie transport mode from all
// enabled OAuth callback URLs unless the operator explicitly overrides it.
func (c Config) OAuthFlowCookieSecure() bool {
	if c.Auth.OAuthCookieSecure != nil {
		return *c.Auth.OAuthCookieSecure
	}
	callbackURLs := make([]string, 0, 2)
	if c.GoogleEnabled() {
		callbackURLs = append(callbackURLs, c.Auth.Google.RedirectURL)
	}
	if c.GitHubEnabled() {
		callbackURLs = append(callbackURLs, c.Auth.GitHub.RedirectURL)
	}
	if len(callbackURLs) == 0 {
		return false
	}
	for _, rawURL := range callbackURLs {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
			return false
		}
	}
	return true
}

func (c Config) BillingEnabled() bool {
	configured := c.Billing.Stripe.SecretKey != "" || c.Billing.Stripe.WebhookSecret != ""
	return boolDefault(c.Billing.Enabled, configured)
}

func (c Config) ReferralEnabled() bool { return boolDefault(c.Referral.Enabled, true) }

func (c Config) ReferralActivationWindow() time.Duration {
	if c.Referral.ActivationWindowDays <= 0 {
		return 0
	}
	return time.Duration(c.Referral.ActivationWindowDays) * 24 * time.Hour
}
