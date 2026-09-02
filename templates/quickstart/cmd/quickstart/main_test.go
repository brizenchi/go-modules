package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/brizenchi/go-modules/foundation/ginx"
	"github.com/brizenchi/quickstart-template/internal/bootstrap"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	apphttp "github.com/brizenchi/quickstart-template/internal/http"
	qmiddleware "github.com/brizenchi/quickstart-template/internal/http/middleware"
	"github.com/gin-gonic/gin"
)

type smokeStack struct{}

func (smokeStack) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

func (smokeStack) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer admin" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func (smokeStack) Mount(publicGroup, userGroup *gin.RouterGroup) {
	publicGroup.GET("/public-ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	userGroup.GET("/user-ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func TestDBConfigPGXConfigFromFields(t *testing.T) {
	cfg := bootstrap.DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "app",
		Password: "secret",
		Name:     "quickstart",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	pgxCfg := cfg.PGXConfig("daobang", "prod")
	if pgxCfg.DSN != "" {
		t.Fatalf("expected empty DSN, got %q", pgxCfg.DSN)
	}
	if pgxCfg.Host != "localhost" || pgxCfg.Port != 5432 || pgxCfg.User != "app" || pgxCfg.Database != "quickstart" {
		t.Fatalf("unexpected PGX config: %+v", pgxCfg)
	}
	if pgxCfg.Project != "daobang" || pgxCfg.Environment != "prod" {
		t.Fatalf("project/env mismatch: %+v", pgxCfg)
	}
}

func TestDBConfigPGXConfigPrefersDSN(t *testing.T) {
	cfg := bootstrap.DBConfig{
		DSN:  "host=db user=app password=secret dbname=quickstart port=5432 sslmode=disable TimeZone=UTC",
		Host: "ignored",
	}

	pgxCfg := cfg.PGXConfig("daobang", "prod")
	if pgxCfg.DSN != cfg.DSN {
		t.Fatalf("expected DSN %q, got %q", cfg.DSN, pgxCfg.DSN)
	}
}

func TestDBConfigSafeStringStructured(t *testing.T) {
	cfg := bootstrap.DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "app",
		Password: "secret",
		Name:     "quickstart",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	got := cfg.SafeString()
	want := "host=localhost user=app password=*** dbname=quickstart port=5432 sslmode=disable TimeZone=UTC"
	if got != want {
		t.Fatalf("SafeString = %q, want %q", got, want)
	}
}

func TestDBConfigSafeStringDSN(t *testing.T) {
	cfg := bootstrap.DBConfig{DSN: "host=db user=app password=secret"}
	if got := cfg.SafeString(); got != "<dsn-masked>" {
		t.Fatalf("SafeString = %q, want <dsn-masked>", got)
	}
}

func TestAppConfigModuleConfig(t *testing.T) {
	cfg := bootstrap.AppConfig{}
	cfg.Server.Name = "quickstart"
	cfg.Auth.UserJWTSecret = "jwt-secret"
	cfg.Email.Provider = "resend"
	cfg.Email.Resend.APIKey = "api-key"
	cfg.Email.Resend.SenderEmail = "noreply@example.com"
	cfg.Auth.Google.StateTTLMin = 35
	cfg.Billing.Stripe.SecretKey = "sk_test_123"
	cfg.Billing.Stripe.WebhookSecret = "whsec_123"
	cfg.Billing.Stripe.Prices.ProMonthly = "price_pro_month"
	cfg.Billing.Stripe.Prices.PremiumMonthly = "price_premium_month"
	cfg.Billing.Stripe.Prices.PremiumYearly = "price_premium_year"
	cfg.Referral.BaseLink = "http://localhost:3000/invite?ref="
	cfg.Referral.ActivationReward = 50

	sc := cfg.ModuleConfig()
	if sc.Auth.UserJWTSecret != "jwt-secret" {
		t.Fatalf("jwt secret mismatch: %q", sc.Auth.UserJWTSecret)
	}
	if sc.Email.Provider != "resend" {
		t.Fatalf("email provider = %q, want resend", sc.Email.Provider)
	}
	if sc.Email.Resend.SenderEmail != "noreply@example.com" {
		t.Fatalf("resend sender email mismatch: %q", sc.Email.Resend.SenderEmail)
	}
	if sc.Auth.Google.StateTTLMin != 35 {
		t.Fatalf("google state ttl mismatch: %d", sc.Auth.Google.StateTTLMin)
	}
	if sc.Billing.Stripe.Prices.ProMonthly != "price_pro_month" {
		t.Fatalf("pro monthly price mismatch: %q", sc.Billing.Stripe.Prices.ProMonthly)
	}
	if sc.Billing.Stripe.Prices.PremiumMonthly != "price_premium_month" {
		t.Fatalf("premium monthly price mismatch: %q", sc.Billing.Stripe.Prices.PremiumMonthly)
	}
	if sc.Billing.Stripe.Prices.PremiumYearly != "price_premium_year" {
		t.Fatalf("premium yearly price mismatch: %q", sc.Billing.Stripe.Prices.PremiumYearly)
	}
	if sc.Referral.ActivationReward != 50 {
		t.Fatalf("activation reward mismatch: %d", sc.Referral.ActivationReward)
	}
}

func TestTracingConfigFields(t *testing.T) {
	cfg := bootstrap.AppConfig{}
	cfg.Tracing.Endpoint = "localhost:4318"
	cfg.Tracing.Protocol = "grpc"
	cfg.Tracing.Insecure = true
	cfg.Tracing.SampleRate = 0.5
	cfg.Tracing.Authorization = "Basic abc"
	cfg.Tracing.Headers = map[string]string{"X-Test": "ok"}

	if cfg.Tracing.Endpoint != "localhost:4318" {
		t.Fatalf("endpoint mismatch: %q", cfg.Tracing.Endpoint)
	}
	if cfg.Tracing.Protocol != "grpc" {
		t.Fatalf("protocol mismatch: %q", cfg.Tracing.Protocol)
	}
	if !cfg.Tracing.Insecure {
		t.Fatal("expected insecure=true")
	}
	if cfg.Tracing.SampleRate != 0.5 {
		t.Fatalf("sample rate mismatch: %v", cfg.Tracing.SampleRate)
	}
	headers := cfg.Tracing.ExporterHeaders()
	if headers["Authorization"] != "Basic abc" || headers["X-Test"] != "ok" {
		t.Fatalf("headers mismatch: %+v", headers)
	}
}

func TestLoadDotEnvLoadsMissingVariablesOnly(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := []byte("APP_DB_HOST=10.0.0.8\nAPP_DB_PORT=6543\n")
	if err := os.WriteFile(envPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("APP_DB_PORT", "7777")
	if err := os.Unsetenv("APP_DB_HOST"); err != nil {
		t.Fatalf("Unsetenv APP_DB_HOST: %v", err)
	}

	if err := bootstrap.LoadDotEnv(envPath); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("APP_DB_HOST"); got != "10.0.0.8" {
		t.Fatalf("APP_DB_HOST = %q, want 10.0.0.8", got)
	}
	if got := os.Getenv("APP_DB_PORT"); got != "7777" {
		t.Fatalf("APP_DB_PORT = %q, want 7777", got)
	}
}

func TestLoadDotEnvMissingFileDoesNothing(t *testing.T) {
	if err := bootstrap.LoadDotEnv(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("LoadDotEnv missing file: %v", err)
	}
}

func TestBuildRouter_HealthAndMountedRoutes(t *testing.T) {
	cfg := bootstrap.AppConfig{}
	cfg.Server.Name = "quickstart"
	cfg.Tracing.SampleRate = 1

	router := qmiddleware.BuildRouter(
		qmiddleware.RouterConfig{ServiceName: cfg.Server.Name},
		apphttp.NewRouter(smokeStack{}, hostapi.Deps{}),
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200", res.Code)
	}
	if got := res.Header().Get(ginx.HeaderRequestID); got == "" {
		t.Fatal("expected X-Request-ID on /health response")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/public-ping", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("/api/v1/public-ping status = %d, want 200", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/user-ping", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("/api/v1/user-ping without auth status = %d, want 401", res.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/user-ping", nil)
	req.Header.Set("Authorization", "Bearer test")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("/api/v1/user-ping with auth status = %d, want 200", res.Code)
	}
}

// TestBuildRouter_HostSlots asserts the host extension points are wired:
// a host route reaches the router, the user group rejects anonymous
// callers, and the admin group rejects non-admins.
func TestBuildRouter_HostSlots(t *testing.T) {
	router := qmiddleware.BuildRouter(
		qmiddleware.RouterConfig{ServiceName: "quickstart"},
		apphttp.NewRouter(smokeStack{}, hostapi.Deps{}),
	)

	cases := []struct {
		name  string
		path  string
		token string
		want  int
	}{
		{"user route without token", "/api/v1/notes", "", http.StatusUnauthorized},
		{"admin route with user token", "/api/v1/admin/notes/count", "Bearer test", http.StatusForbidden},
		{"admin route without token", "/api/v1/admin/notes/count", "", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.want {
				t.Fatalf("%s %s = %d, want %d", http.MethodGet, tc.path, res.Code, tc.want)
			}
		})
	}
}
