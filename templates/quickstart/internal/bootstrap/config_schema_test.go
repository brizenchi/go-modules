package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	foundationconfig "github.com/brizenchi/go-modules/foundation/config"
)

func falsePtr() *bool {
	v := false
	return &v
}

func truePtr() *bool {
	v := true
	return &v
}

func TestOperatorAndUploadEnvironmentConfiguration(t *testing.T) {
	t.Setenv("QUICKSTART_OPERATIONS_AUTH_ADMIN_EMAILS", "owner@example.test,second@example.test")
	t.Setenv("QUICKSTART_OPERATIONS_HOST_UPLOADS_ENABLED", "true")
	t.Setenv("QUICKSTART_OPERATIONS_HOST_UPLOADS_PROVIDER", "s3")
	t.Setenv("QUICKSTART_OPERATIONS_HOST_UPLOADS_BUCKET", "private-fixture")
	t.Setenv("QUICKSTART_OPERATIONS_HOST_UPLOADS_REGION", "auto")
	source, err := os.ReadFile("../../deploy/config.yaml.example")
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, source, 0600); err != nil {
		t.Fatal(err)
	}
	var cfg AppConfig
	if err := foundationconfig.Load(configPath, "QUICKSTART_OPERATIONS", &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Auth.AdminEmails) != 2 || cfg.Auth.AdminEmails[1] != "second@example.test" {
		t.Fatalf("admin email mapping: %v", cfg.Auth.AdminEmails)
	}
	if !cfg.Host.Uploads.Enabled || cfg.Host.Uploads.Provider != "s3" || cfg.Host.Uploads.Bucket != "private-fixture" || cfg.Host.Uploads.Region != "auto" {
		t.Fatalf("upload config mapping: %+v", cfg.Host.Uploads)
	}
}

func TestDeployConfigExampleMatchesAppConfig(t *testing.T) {
	source, err := os.ReadFile("../../deploy/config.yaml.example")
	if err != nil {
		t.Fatalf("读取 deploy/config.yaml.example: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatalf("复制示例配置: %v", err)
	}

	var cfg AppConfig
	if err := foundationconfig.Load(configPath, "QUICKSTART_EXAMPLE", &cfg); err != nil {
		t.Fatalf("加载 deploy/config.yaml.example: %v", err)
	}

	modules := cfg.ModuleConfig()
	if !modules.AuthEnabled() || !modules.EmailAuthEnabled() {
		t.Fatal("示例配置应该启用 auth 和邮箱验证码登录")
	}
	if modules.GoogleEnabled() || modules.GitHubEnabled() || modules.BillingEnabled() {
		t.Fatal("没有第三方凭据的示例配置不应该启用 OAuth 或支付")
	}
	if cfg.Billing.Enabled != nil {
		t.Fatal("示例配置的 billing.enabled 应留空，以便提供凭据后自动启用")
	}
	if cfg.Auth.OAuthCookieSecure != nil {
		t.Fatal("示例配置的 oauth_cookie_secure 应留空，以便按 callback URL 推导")
	}
	if !modules.ReferralEnabled() {
		t.Fatal("示例配置应该启用邀请模块")
	}
	if cfg.Host.WelcomeEmail.Enabled || cfg.Host.SignupCredits != 0 {
		t.Fatal("示例配置默认不应该发送欢迎邮件或发放注册积分")
	}
}

func TestApplyDefaultsDerivesReferralBaseLinkFromFrontendRedirect(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.FrontendRedirect = "https://template.daobang.tech/login?source=oauth"

	applyDefaults(&cfg)

	if got, want := cfg.Referral.BaseLink, "https://template.daobang.tech/invite?ref="; got != want {
		t.Fatalf("referral base link = %q, want %q", got, want)
	}
}

func TestValidateProductionRejectsInvalidReferralBaseLink(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Referral.BaseLink = "http://app.example.com/invite?ref="
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "referral.base_link") {
		t.Fatalf("expected referral base link error, got %v", err)
	}
}

func TestValidateRejectsUserCapabilitiesWithoutAuth(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*AppConfig)
		want      string
	}{
		{
			name: "billing",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Enabled = truePtr()
				cfg.Referral.Enabled = falsePtr()
			},
			want: "billing requires auth.enabled=true",
		},
		{
			name: "referral",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Enabled = falsePtr()
				cfg.Referral.Enabled = truePtr()
			},
			want: "referral requires auth.enabled=true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AppConfig{}
			applyDefaults(&cfg)
			cfg.Auth.Enabled = falsePtr()
			tt.configure(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error=%v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateProductionRejectsUnsafeDefaults(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Auth.UserJWTSecret = "CHANGE-ME-32-RANDOM-CHARS"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "user_jwt_secret") {
		t.Fatalf("expected unsafe JWT secret error, got %v", err)
	}
}

func TestValidateProductionRequiresDatabaseTLS(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		mode string
	}{
		{name: "missing mode"},
		{name: "structured disable", mode: "disable"},
		{name: "keyword DSN disable", dsn: "host=db.example.com dbname=app sslmode=disable"},
		{name: "URL DSN missing mode", dsn: "postgres://user:pass@db.example.com/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := productionConfigForTest()
			cfg.DB.DSN = tt.dsn
			cfg.DB.SSLMode = tt.mode
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "db.ssl_mode") {
				t.Fatalf("Validate error=%v, want production database TLS rejection", err)
			}
		})
	}
}

func TestValidateProductionAcceptsDatabaseTLSInDSN(t *testing.T) {
	for _, dsn := range []string{
		"host=db.example.com dbname=app sslmode=require",
		"postgres://user:pass@db.example.com/app?sslmode=verify-full",
	} {
		cfg := productionConfigForTest()
		cfg.DB.DSN = dsn
		cfg.DB.SSLMode = ""
		if err := cfg.Validate(); err != nil {
			t.Fatalf("Validate DSN %q: %v", dsn, err)
		}
	}
}

func TestValidateProductionRejectsDebugAndLogEmail(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Auth.Email.Debug = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "debug") {
		t.Fatalf("expected debug email error, got %v", err)
	}

	cfg = productionConfigForTest()
	cfg.Email.Provider = "log"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "email.provider") {
		t.Fatalf("expected production email provider error, got %v", err)
	}
}

func TestValidateProductionRejectsPlaceholderProviderCredentials(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Email.Resend.APIKey = "CHANGE-ME"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("expected placeholder email credential error, got %v", err)
	}

	cfg = productionConfigForTest()
	cfg.Auth.Google.Enabled = truePtr()
	cfg.Auth.Google.ClientID = "google-client"
	cfg.Auth.Google.ClientSecret = "CHANGE-ME"
	cfg.Auth.Google.RedirectURL = "https://api.example.com/api/v1/auth/google/callback"
	cfg.Auth.Google.StateSecret = "abcdef0123456789abcdef0123456789"
	cfg.Auth.FrontendRedirect = "https://app.example.com/login"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "client_id and client_secret") {
		t.Fatalf("expected placeholder OAuth credential error, got %v", err)
	}
}

func TestValidateProductionRejectsWildcardCORS(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.HTTP.AllowedOrigins = "*"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "allowed_origins") {
		t.Fatalf("expected wildcard CORS error, got %v", err)
	}
}

func TestApplyDefaultsDerivesCORSFromOAuthFrontendRedirect(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.FrontendRedirect = "https://template.daobang.tech/login?source=oauth"

	applyDefaults(&cfg)

	if got, want := cfg.HTTP.AllowedOrigins, "https://template.daobang.tech"; got != want {
		t.Fatalf("allowed origins = %q, want %q", got, want)
	}
}

func TestValidateRejectsOAuthFrontendMissingFromCORS(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.HTTP.AllowedOrigins = "https://another.example.com"
	enableGoogleForTest(&cfg)
	cfg.Auth.FrontendRedirect = "https://template.daobang.tech/login"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "APP_HTTP_ALLOWED_ORIGINS") {
		t.Fatalf("expected OAuth frontend CORS error, got %v", err)
	}
}

func TestValidateAcceptsOAuthFrontendInCORSList(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.HTTP.AllowedOrigins = "https://admin.example.com,https://template.daobang.tech"
	enableGoogleForTest(&cfg)
	cfg.Auth.FrontendRedirect = "https://template.daobang.tech/login"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateProductionRejectsInsecureOAuthCookieOverride(t *testing.T) {
	cfg := productionConfigForTest()
	enableGoogleForTest(&cfg)
	insecure := false
	cfg.Auth.OAuthCookieSecure = &insecure
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "oauth_cookie_secure") {
		t.Fatalf("Validate error=%v, want production secure-cookie rejection", err)
	}
}

func TestValidateOAuthRequiresExplicitCORSOriginWhenCredentialsEnabled(t *testing.T) {
	cfg := productionConfigForTest()
	enableGoogleForTest(&cfg)
	cfg.HTTP.AllowedOrigins = "*"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "allowed_origins") {
		t.Fatalf("Validate error=%v, want wildcard rejection", err)
	}
}

func TestValidateRejectsIncompleteStripeConfig(t *testing.T) {
	cfg := productionConfigForTest()
	enabled := true
	cfg.Billing.Enabled = &enabled
	cfg.Billing.Stripe.SecretKey = "sk_live_valid"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "webhook_secret") {
		t.Fatalf("expected incomplete Stripe error, got %v", err)
	}
}

func TestValidateProductionRejectsInvalidStripePriceIDs(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*AppConfig)
		want      string
	}{
		{
			name: "subscription placeholder",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.Prices.StarterMonthly = "price_..."
			},
			want: "starter_monthly",
		},
		{
			name: "wrong object type",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.Prices.StarterMonthly = "prod_1TBFpjQ4kdQzysE6eeAA5D38"
			},
			want: "starter_monthly",
		},
		{
			name: "credits placeholder",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.Prices.StarterMonthly = ""
				cfg.Billing.Stripe.Prices.Credits = []string{"price_credits_placeholder"}
			},
			want: "credits[0]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := productionBillingConfigForTest()
			tt.configure(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error = %v, want field %q", err, tt.want)
			}
		})
	}
}

func TestValidateProductionRejectsInvalidStripeCredentialFormats(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*AppConfig)
		want      string
	}{
		{
			name: "secret key",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.SecretKey = "definitely-not-stripe"
			},
			want: "secret_key",
		},
		{
			name: "webhook signing secret",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.WebhookSecret = "definitely-not-stripe"
			},
			want: "webhook_secret",
		},
		{
			name: "secret key with surrounding whitespace",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.SecretKey = " sk_live_1234567890abcdef "
			},
			want: "secret_key",
		},
		{
			name: "webhook secret with invalid characters",
			configure: func(cfg *AppConfig) {
				cfg.Billing.Stripe.WebhookSecret = "whsec_12345678!"
			},
			want: "webhook_secret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := productionBillingConfigForTest()
			tt.configure(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error = %v, want field %q", err, tt.want)
			}
		})
	}
}

func TestValidateProductionAcceptsRealisticStripePriceIDs(t *testing.T) {
	cfg := productionBillingConfigForTest()
	cfg.Billing.Stripe.Prices.Lifetime = "price_1TBZGJQ4kdQzysE60zIoEmSP"
	cfg.Billing.Stripe.Prices.Credits = []string{"price_1TBZGSQ4kdQzysE60OOpWjYs"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateProductionRejectsDuplicateStripePriceIDsAcrossOffers(t *testing.T) {
	const duplicate = "price_1DuplicateOffer123456789"
	tests := []struct {
		name      string
		configure func(*AppConfig)
	}{
		{name: "subscription plans", configure: func(cfg *AppConfig) {
			cfg.Billing.Stripe.Prices.StarterMonthly = duplicate
			cfg.Billing.Stripe.Prices.ProYearly = duplicate
		}},
		{name: "lifetime and subscription", configure: func(cfg *AppConfig) {
			cfg.Billing.Stripe.Prices.StarterMonthly = duplicate
			cfg.Billing.Stripe.Prices.Lifetime = duplicate
		}},
		{name: "credits and subscription", configure: func(cfg *AppConfig) {
			cfg.Billing.Stripe.Prices.StarterMonthly = duplicate
			cfg.Billing.Stripe.Prices.Credits = []string{duplicate}
		}},
		{name: "credits list", configure: func(cfg *AppConfig) {
			cfg.Billing.Stripe.Prices.Credits = []string{duplicate, duplicate}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := productionBillingConfigForTest()
			tt.configure(&cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "duplicates") {
				t.Fatalf("Validate error=%v, want duplicate price rejection", err)
			}
		})
	}
}

func TestValidateBillingRequiresExplicitFrontendOrigin(t *testing.T) {
	cfg := productionBillingConfigForTest()
	cfg.Auth.FrontendRedirect = ""
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "frontend_redirect") {
		t.Fatalf("Validate error=%v, want frontend origin rejection", err)
	}
}

func TestValidateAcceptsMinimalProductionConfig(t *testing.T) {
	cfg := productionConfigForTest()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func productionConfigForTest() AppConfig {
	cfg := AppConfig{Env: "production"}
	cfg.Auth.FrontendRedirect = "https://app.example.com/login"
	applyDefaults(&cfg)
	cfg.HTTP.AllowedOrigins = "https://app.example.com"
	cfg.Auth.UserJWTSecret = "0123456789abcdef0123456789abcdef"
	cfg.Auth.Google.Enabled = falsePtr()
	cfg.Auth.GitHub.Enabled = falsePtr()
	cfg.Auth.Email.Debug = false
	cfg.Email.Provider = "resend"
	cfg.Email.Resend.APIKey = "re_live_valid"
	cfg.Email.Resend.SenderEmail = "no-reply@example.com"
	cfg.Billing.Enabled = falsePtr()
	cfg.DB.SSLMode = "require"
	return cfg
}

func productionBillingConfigForTest() AppConfig {
	cfg := productionConfigForTest()
	cfg.Billing.Enabled = truePtr()
	cfg.Billing.Provider = "stripe"
	cfg.Billing.Stripe.SecretKey = "sk_live_1234567890abcdef"
	cfg.Billing.Stripe.WebhookSecret = "whsec_1234567890abcdef"
	cfg.Billing.Stripe.Prices.StarterMonthly = "price_1TBFpjQ4kdQzysE6eeAA5D38"
	return cfg
}

func enableGoogleForTest(cfg *AppConfig) {
	cfg.Auth.Google.Enabled = truePtr()
	cfg.Auth.Google.ClientID = "google-client"
	cfg.Auth.Google.ClientSecret = "google-secret"
	cfg.Auth.Google.RedirectURL = "https://api.example.com/api/v1/auth/google/callback"
	cfg.Auth.Google.StateSecret = "abcdef0123456789abcdef0123456789"
}
