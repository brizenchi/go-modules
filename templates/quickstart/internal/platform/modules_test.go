package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func boolPointer(value bool) *bool { return &value }

func openPlatformTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestCapabilitySwitchDefaultsAndCredentialInference(t *testing.T) {
	cfg := Config{}
	if !cfg.AuthEnabled() || !cfg.EmailAuthEnabled() || !cfg.ReferralEnabled() {
		t.Fatal("auth、邮箱登录和邀请默认应该启用")
	}
	if cfg.GoogleEnabled() || cfg.GitHubEnabled() || cfg.BillingEnabled() {
		t.Fatal("没有凭据时 OAuth 和支付默认不应该启用")
	}

	cfg.Auth.Google.ClientID = "google-client"
	cfg.Auth.GitHub.ClientSecret = "github-secret"
	cfg.Billing.Stripe.SecretKey = "stripe-secret"
	if !cfg.GoogleEnabled() || !cfg.GitHubEnabled() || !cfg.BillingEnabled() {
		t.Fatal("省略 enabled 时应该根据已有凭据启用能力")
	}

	cfg.Auth.Google.Enabled = boolPointer(false)
	cfg.Auth.GitHub.Enabled = boolPointer(false)
	cfg.Billing.Enabled = boolPointer(false)
	if cfg.GoogleEnabled() || cfg.GitHubEnabled() || cfg.BillingEnabled() {
		t.Fatal("显式 enabled=false 必须覆盖凭据推断")
	}
}

func TestDisabledCapabilitiesAreNotBuiltOrMigrated(t *testing.T) {
	db := openPlatformTestDB(t)
	cfg := Config{
		Email:    EmailConfig{Provider: "none"},
		Auth:     AuthConfig{Enabled: boolPointer(false)},
		Billing:  BillingConfig{Enabled: boolPointer(false)},
		Referral: ReferralConfig{Enabled: boolPointer(false)},
	}
	if err := Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := New(db, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if modules.Auth != nil || modules.Email != nil || modules.Billing != nil || modules.Referral != nil {
		t.Fatalf("unexpected enabled modules: %#v", modules)
	}
	for _, table := range []string{"users", "user_identities"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("host table %q was not migrated", table)
		}
	}
	for _, table := range []string{"auth_email_codes", "billing_customers", "referrals"} {
		if db.Migrator().HasTable(table) {
			t.Fatalf("disabled module table %q should not exist", table)
		}
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/private", modules.RequireUser(), func(c *gin.Context) { c.Status(http.StatusOK) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/private", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled auth middleware status=%d, want 503", response.Code)
	}
}

func TestGitHubOnlyAuthMountsOAuthWithoutEmailLogin(t *testing.T) {
	db := openPlatformTestDB(t)
	cfg := Config{
		ServiceName: "github-only",
		Email:       EmailConfig{Provider: "none"},
		Auth: AuthConfig{
			UserJWTSecret: "test-jwt-secret",
			Email:         AuthEmailConfig{Enabled: boolPointer(false)},
			Google:        GoogleConfig{Enabled: boolPointer(false)},
			GitHub: GitHubConfig{
				Enabled:      boolPointer(true),
				ClientID:     "github-client",
				ClientSecret: "github-secret",
				RedirectURL:  "https://api.example.com/api/v1/auth/github/callback",
				StateSecret:  "github-state-secret",
			},
		},
		Billing:  BillingConfig{Enabled: boolPointer(false)},
		Referral: ReferralConfig{Enabled: boolPointer(false)},
	}
	if err := Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := New(db, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if modules.Auth == nil || len(modules.Auth.Deps.IdentityProviders) != 1 {
		t.Fatalf("identity providers=%v", modules.Auth)
	}
	if _, ok := modules.Auth.Deps.IdentityProviders["github"]; !ok {
		t.Fatalf("github provider not registered: %v", modules.Auth.Deps.IdentityProviders)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	public := router.Group("/api/v1")
	user := router.Group("/api/v1")
	modules.Mount(public, user)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/auth/send-code", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("email route status=%d, want 404", response.Code)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/auth/github/authorize", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("github authorize status=%d body=%s", response.Code, response.Body.String())
	}
}
