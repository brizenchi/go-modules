package platform

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authmodule "github.com/brizenchi/go-modules/modules/auth"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type capabilitiesEnvelope struct {
	Code int `json:"code"`
	Data struct {
		Auth struct {
			EmailEnabled   bool     `json:"email_enabled"`
			OAuthProviders []string `json:"oauth_providers"`
		} `json:"auth"`
		Account struct {
			Enabled bool `json:"enabled"`
		} `json:"account"`
		Billing struct {
			Enabled  bool   `json:"enabled"`
			Provider string `json:"provider"`
			Offers   struct {
				Subscriptions []struct {
					Plan      string   `json:"plan"`
					Intervals []string `json:"intervals"`
				} `json:"subscriptions"`
				Lifetime bool `json:"lifetime"`
				Credits  bool `json:"credits"`
			} `json:"offers"`
		} `json:"billing"`
		Referral struct {
			Enabled bool `json:"enabled"`
		} `json:"referral"`
	} `json:"data"`
}

func TestRespondAuthErrorDoesNotLeakUnknownInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPost, "/auth/exchange-token", nil)

	respondAuthError(context, errors.New("database dsn secret-must-not-leak"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-must-not-leak") || !strings.Contains(response.Body.String(), "internal authentication error") {
		t.Fatalf("unsafe response body: %s", response.Body.String())
	}
}

func TestCapabilitiesExposeConfiguredBillingOffersWithoutPriceIDs(t *testing.T) {
	db := openPlatformTestDB(t)
	cfg := Config{
		Email: EmailConfig{Provider: "log"},
		Auth: AuthConfig{
			UserJWTSecret:    "test-jwt-secret",
			FrontendRedirect: "http://localhost:3000",
			Google:           GoogleConfig{Enabled: boolPointer(false)},
			GitHub:           GitHubConfig{Enabled: boolPointer(false)},
		},
		Billing: BillingConfig{
			Enabled:  boolPointer(true),
			Provider: "stripe",
			Stripe: StripeConfig{
				SecretKey:     "sk_test_private",
				WebhookSecret: "whsec_private",
				Prices: StripePricesConfig{
					StarterMonthly: "price_1TBFpjQ4kdQzysE6eeAA5D38",
					ProYearly:      "price_1TBFpsQ4kdQzysE6GVEz01dA",
					Lifetime:       "price_1TBZGJQ4kdQzysE60zIoEmSP",
					Credits:        []string{"price_1TBZGSQ4kdQzysE60OOpWjYs"},
				},
			},
		},
	}
	if err := Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := New(db, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	modules.Mount(router.Group("/api/v1"), router.Group("/api/v1"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body capabilitiesEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if !body.Data.Billing.Enabled || !body.Data.Billing.Offers.Lifetime || !body.Data.Billing.Offers.Credits {
		t.Fatalf("unexpected billing offers: %#v", body.Data.Billing)
	}
	if !body.Data.Auth.EmailEnabled || len(body.Data.Auth.OAuthProviders) != 0 {
		t.Fatalf("unexpected auth capabilities: %#v", body.Data.Auth)
	}
	if got := body.Data.Billing.Offers.Subscriptions; len(got) != 2 || got[0].Plan != "starter" || got[1].Plan != "pro" {
		t.Fatalf("subscriptions=%#v", got)
	}
	if strings.Contains(response.Body.String(), "price_") || strings.Contains(response.Body.String(), "sk_test") || strings.Contains(response.Body.String(), "whsec_") {
		t.Fatalf("capability response leaked billing credentials: %s", response.Body.String())
	}
}

func TestAuthWrappersRejectOversizedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	modules := &Modules{Auth: &authmodule.Module{}}
	router := gin.New()
	router.POST("/verify", modules.verifyCode())
	router.POST("/exchange", modules.exchangeToken())

	largeValue := strings.Repeat("a", 40<<10)
	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/verify", body: `{"email":"user@example.com","code":"` + largeValue + `"}`},
		{path: "/exchange", body: `{"code":"` + largeValue + `"}`},
	} {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestConfiguredAuthCapabilityUsesStableProviderOrder(t *testing.T) {
	modules := &Modules{
		Auth: &authmodule.Module{Deps: authmodule.Deps{
			IdentityProviders: map[string]authport.IdentityProvider{
				"github": nil,
				"google": nil,
			},
		}},
		emailAuthEnabled: true,
	}

	capability := modules.configuredAuthCapability()
	if !capability.EmailEnabled {
		t.Fatal("email capability should be enabled")
	}
	if got, want := strings.Join(capability.OAuthProviders, ","), "google,github"; got != want {
		t.Fatalf("oauth providers=%q, want %q", got, want)
	}
}

func TestConfiguredBillingOffersExcludeInvalidPriceIDs(t *testing.T) {
	offers := configuredBillingOffers(StripePricesConfig{
		StarterMonthly: "price_...",
		ProYearly:      "price_1TBFpsQ4kdQzysE6GVEz01dA",
		Lifetime:       "prod_1TBZGJQ4kdQzysE60zIoEmSP",
		Credits:        []string{"price_credits_placeholder"},
	})
	if len(offers.Subscriptions) != 1 || offers.Subscriptions[0].Plan != "pro" || len(offers.Subscriptions[0].Intervals) != 1 || offers.Subscriptions[0].Intervals[0] != "yearly" {
		t.Fatalf("subscriptions = %#v, want only pro yearly", offers.Subscriptions)
	}
	if offers.Lifetime || offers.Credits {
		t.Fatalf("invalid one-time offers must be hidden: %#v", offers)
	}
}

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

func TestOAuthFlowCookieSecureDerivesFromCallbackSchemesAndAllowsExplicitLocalOverride(t *testing.T) {
	cfg := Config{Auth: AuthConfig{
		Google: GoogleConfig{Enabled: boolPointer(true), ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost:8080/api/v1/auth/google/callback"},
		GitHub: GitHubConfig{Enabled: boolPointer(false)},
	}}
	if cfg.OAuthFlowCookieSecure() {
		t.Fatal("local HTTP callback must derive Secure=false")
	}
	cfg.Auth.Google.RedirectURL = "https://api.example.com/api/v1/auth/google/callback"
	if !cfg.OAuthFlowCookieSecure() {
		t.Fatal("HTTPS callback must derive Secure=true")
	}
	explicitLocal := false
	cfg.Auth.OAuthCookieSecure = &explicitLocal
	if cfg.OAuthFlowCookieSecure() {
		t.Fatal("explicit local override was ignored")
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
	_, challenge := wrapperVerifier(9)
	router.ServeHTTP(response, httptest.NewRequest(
		http.MethodGet, "/api/v1/auth/github/authorize?challenge="+challenge, nil,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("github authorize status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestMountPublishesCapabilitiesAndClearDisabledBillingError(t *testing.T) {
	db := openPlatformTestDB(t)
	cfg := Config{
		Email: EmailConfig{Provider: "log"},
		Auth: AuthConfig{
			UserJWTSecret: "test-jwt-secret",
			Google:        GoogleConfig{Enabled: boolPointer(false)},
			GitHub:        GitHubConfig{Enabled: boolPointer(false)},
		},
		Billing: BillingConfig{
			Enabled:  boolPointer(false),
			Provider: "stripe",
			Stripe: StripeConfig{Prices: StripePricesConfig{
				StarterMonthly: "price_1TBFpjQ4kdQzysE6eeAA5D38",
				Lifetime:       "price_1TBZGJQ4kdQzysE60zIoEmSP",
				Credits:        []string{"price_1TBZGSQ4kdQzysE60OOpWjYs"},
			}},
		},
	}
	if err := Migrate(db, cfg); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	modules, err := New(db, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	public := router.Group("/api/v1")
	userGroup := router.Group("/api/v1")
	modules.Mount(public, userGroup)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("capabilities status=%d body=%s", response.Code, response.Body.String())
	}
	var body capabilitiesEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if !body.Data.Account.Enabled || body.Data.Billing.Enabled || body.Data.Billing.Provider != "stripe" || !body.Data.Referral.Enabled {
		t.Fatalf("unexpected capabilities: %#v", body.Data)
	}
	if !body.Data.Auth.EmailEnabled || len(body.Data.Auth.OAuthProviders) != 0 {
		t.Fatalf("unexpected auth capabilities: %#v", body.Data.Auth)
	}
	if len(body.Data.Billing.Offers.Subscriptions) != 0 || body.Data.Billing.Offers.Lifetime || body.Data.Billing.Offers.Credits {
		t.Fatalf("disabled billing must expose empty offers: %#v", body.Data.Billing.Offers)
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stripe/subscription", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled billing status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "billing is not configured") {
		t.Fatalf("disabled billing response=%s", response.Body.String())
	}
}
