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
	if !modules.ReferralEnabled() {
		t.Fatal("示例配置应该启用邀请模块")
	}
	if cfg.Host.WelcomeEmail.Enabled || cfg.Host.SignupCredits != 0 {
		t.Fatal("示例配置默认不应该发送欢迎邮件或发放注册积分")
	}
}

func TestValidateProductionRejectsUnsafeDefaults(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Auth.UserJWTSecret = "CHANGE-ME-32-RANDOM-CHARS"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "user_jwt_secret") {
		t.Fatalf("expected unsafe JWT secret error, got %v", err)
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

func TestValidateRejectsIncompleteStripeConfig(t *testing.T) {
	cfg := productionConfigForTest()
	enabled := true
	cfg.Billing.Enabled = &enabled
	cfg.Billing.Stripe.SecretKey = "sk_live_valid"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "webhook_secret") {
		t.Fatalf("expected incomplete Stripe error, got %v", err)
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
	return cfg
}
