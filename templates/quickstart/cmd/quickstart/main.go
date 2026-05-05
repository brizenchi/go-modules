// Quickstart server: wires every go-modules package into a runnable Gin app.
//
// Boot order (mirrored in initialization here):
//
//	logger → config → DB → email → auth → billing → referral → routes → listen
//
// Each step is small enough to read top-to-bottom. To make this your
// project, copy this directory, initialize your own module, and replace
// the host-specific reward / business hooks.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/brizenchi/go-modules/foundation/config"
	"github.com/brizenchi/go-modules/foundation/ginx"
	"github.com/brizenchi/go-modules/foundation/pgx"
	fslog "github.com/brizenchi/go-modules/foundation/slog"
	"github.com/brizenchi/go-modules/foundation/tracing"
	"github.com/brizenchi/go-modules/stacks/saascore"
	"github.com/subosito/gotenv"

	"github.com/gin-gonic/gin"
)

type routeStack interface {
	RequireUser() gin.HandlerFunc
	Mount(publicGroup, userGroup *gin.RouterGroup)
}

func main() {
	// 1. Logger — must be first so everything else lands in slog.
	cfg := loadConfig()
	fslog.Setup(fslog.Config{
		Level:    cfg.Log.Level,
		Format:   fslog.Format(cfg.Log.Format),
		Defaults: logDefaults(cfg),
	})

	traceShutdown, err := tracing.Setup(tracing.Config{
		ServiceName: cfg.Server.Name,
		Project:     cfg.Project,
		Environment: cfg.Env,
		Endpoint:    cfg.Tracing.Endpoint,
		Protocol:    cfg.Tracing.Protocol,
		Insecure:    cfg.Tracing.Insecure,
		SampleRate:  cfg.Tracing.SampleRate,
		Headers:     cfg.Tracing.headers(),
		URLPath:     cfg.Tracing.URLPath,
	})
	if err != nil {
		log.Fatalf("tracing.Setup: %v", err)
	}
	defer tracing.Shutdown(context.Background(), traceShutdown)

	// 2. DB — every shared package that persists takes *gorm.DB.
	db, err := pgx.Open(cfg.DB.PGXConfig(cfg.Project, cfg.Env))
	if err != nil {
		log.Fatalf("pgx.Open: %v", err)
	}
	if err := pgx.HealthCheck(context.Background(), db); err != nil {
		log.Fatalf("pgx.HealthCheck: %v", err)
	}
	slog.Info("db ready", "dsn_safe", cfg.DB.SafeString())

	stack, err := saascore.New(
		db,
		cfg.SaaSCoreConfig(),
		saascore.HostHooks{
			OnReferralActivated: func(ctx context.Context, event saascore.ReferralActivatedEvent) error {
				return referralReward(ctx, db, event)
			},
		},
		saascore.PolicyHooks{},
	)
	if err != nil {
		log.Fatalf("saascore.New: %v", err)
	}

	// 8. HTTP.
	r := buildRouter(cfg, stack)

	// 9. Listen + graceful shutdown.
	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Server.Port), Handler: r}
	go func() {
		slog.Info("listening", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "error", err)
	}
}

func buildRouter(cfg AppConfig, stack routeStack) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(
		ginx.Recover(),
		ginx.RequestID(),
		tracing.Trace(cfg.Server.Name),
		ginx.AccessLog(ginx.AccessLogConfig{SkipPaths: []string{"/health"}}),
	)
	r.Use(ginx.CORS(ginx.CORSConfig{AllowedOrigins: []string{"*"}}))
	r.Use(ginx.NoCache(), ginx.Secure(ginx.SecureConfig{}))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	if stack != nil {
		publicGroup := r.Group("/api/v1")
		userGroup := r.Group("/api/v1")
		userGroup.Use(stack.RequireUser())
		stack.Mount(publicGroup, userGroup)
	}

	return r
}

// AppConfig is the project-specific config shape.
type AppConfig struct {
	Project string `mapstructure:"project"`
	Env     string `mapstructure:"env"`
	Server  struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"server"`
	Log struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"log"`
	Tracing  TracingConfig         `mapstructure:"tracing"`
	DB       DBConfig              `mapstructure:"db"`
	Auth     AuthConfig            `mapstructure:"auth"`
	Email    EmailPlatformConfig   `mapstructure:"email"`
	Billing  BillingPlatformConfig `mapstructure:"billing"`
	Referral ReferralConfig        `mapstructure:"referral"`
}

type TracingConfig struct {
	Endpoint      string            `mapstructure:"endpoint"`
	Protocol      string            `mapstructure:"protocol"`
	Insecure      bool              `mapstructure:"insecure"`
	SampleRate    float64           `mapstructure:"sample_rate"`
	Authorization string            `mapstructure:"authorization"`
	Headers       map[string]string `mapstructure:"headers"`
	URLPath       string            `mapstructure:"url_path"`
}

type AuthConfig struct {
	UserJWTSecret      string          `mapstructure:"user_jwt_secret"`
	UserJWTExpireHours int             `mapstructure:"user_jwt_expire_hours"`
	WSTicketTTLSeconds int             `mapstructure:"ws_ticket_ttl_seconds"`
	AdminEmails        []string        `mapstructure:"admin_emails"`
	FrontendRedirect   string          `mapstructure:"frontend_redirect"`
	Email              AuthEmailConfig `mapstructure:"email"`
	Google             GoogleConfig    `mapstructure:"google"`
}

type AuthEmailConfig struct {
	Debug bool                `mapstructure:"debug"`
	Code  AuthEmailCodeConfig `mapstructure:"code"`
}

type AuthEmailCodeConfig struct {
	TTLMinutes          int `mapstructure:"ttl_minutes"`
	MinResendGapSeconds int `mapstructure:"min_resend_gap_seconds"`
	DailyCap            int `mapstructure:"daily_cap"`
	MaxAttempts         int `mapstructure:"max_attempts"`
}

type GoogleConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	StateSecret  string `mapstructure:"state_secret"`
	Scope        string `mapstructure:"scope"`
}

type EmailPlatformConfig struct {
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

type BillingPlatformConfig struct {
	Stripe StripeConfig `mapstructure:"stripe"`
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
	Prefix           string `mapstructure:"prefix"`
	BaseLink         string `mapstructure:"base_link"`
	ActivationReward int    `mapstructure:"activation_reward"`
}

type DBConfig struct {
	DSN                string `mapstructure:"dsn"`
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	Name               string `mapstructure:"name"`
	SSLMode            string `mapstructure:"ssl_mode"`
	TimeZone           string `mapstructure:"time_zone"`
	LogLevel           string `mapstructure:"log_level"`
	SlowQueryMS        int    `mapstructure:"slow_query_ms"`
	SlowQueryThreshold string `mapstructure:"slow_query_threshold"`
}

func (c DBConfig) PGXConfig(project, env string) pgx.Config {
	slow := 0 * time.Millisecond
	switch {
	case c.SlowQueryMS > 0:
		slow = time.Duration(c.SlowQueryMS) * time.Millisecond
	case strings.TrimSpace(c.SlowQueryThreshold) != "":
		if d, err := time.ParseDuration(strings.TrimSpace(c.SlowQueryThreshold)); err == nil {
			slow = d
		}
	}
	if strings.TrimSpace(c.DSN) != "" {
		return pgx.Config{
			DSN:                c.DSN,
			LogLevel:           c.LogLevel,
			SlowQueryThreshold: slow,
			Project:            project,
			Environment:        env,
		}
	}
	return pgx.Config{
		Host:               c.Host,
		Port:               c.Port,
		User:               c.User,
		Password:           c.Password,
		Database:           c.Name,
		SSLMode:            c.SSLMode,
		TimeZone:           c.TimeZone,
		LogLevel:           c.LogLevel,
		SlowQueryThreshold: slow,
		Project:            project,
		Environment:        env,
	}
}

// SafeString returns a logging-friendly representation of the database
// config with password masked.
func (c DBConfig) SafeString() string {
	if strings.TrimSpace(c.DSN) != "" {
		return "<dsn-masked>"
	}
	host := c.Host
	if host == "" {
		host = "<empty-host>"
	}
	user := c.User
	if user == "" {
		user = "<empty-user>"
	}
	name := c.Name
	if name == "" {
		name = "<empty-db>"
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	timeZone := c.TimeZone
	if timeZone == "" {
		timeZone = "UTC"
	}
	return fmt.Sprintf("host=%s user=%s password=*** dbname=%s port=%d sslmode=%s TimeZone=%s",
		host, user, name, port, sslMode, timeZone)
}

func loadConfig() AppConfig {
	loadDotEnv(".env")

	path := os.Getenv("CONFIG")
	if path == "" {
		path = "deploy/config.yaml"
	}
	var cfg AppConfig
	if err := config.LoadGlobal(path, "APP", &cfg); err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Project == "" {
		cfg.Project = cfg.Server.Name
	}
	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Server.Name == "" {
		cfg.Server.Name = "quickstart"
	}
	if cfg.Project == "" {
		cfg.Project = cfg.Server.Name
	}
	if cfg.Tracing.Protocol == "" {
		cfg.Tracing.Protocol = "http"
	}
	if cfg.Tracing.SampleRate == 0 {
		cfg.Tracing.SampleRate = parseSampleRate(os.Getenv("APP_TRACING_SAMPLE_RATE"), cfg.Tracing.SampleRate)
	}
	if cfg.DB.LogLevel == "" {
		cfg.DB.LogLevel = "warn"
	}
	if cfg.DB.SlowQueryMS == 0 {
		cfg.DB.SlowQueryMS = 200
	}
	return cfg
}

func loadDotEnv(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	if err := gotenv.Load(path); err != nil {
		log.Fatalf("dotenv: %v", err)
	}
}

func (c TracingConfig) headers() map[string]string {
	headers := make(map[string]string, len(c.Headers)+1)
	for k, v := range c.Headers {
		headers[k] = v
	}
	if strings.TrimSpace(c.Authorization) != "" {
		headers["Authorization"] = strings.TrimSpace(c.Authorization)
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func logDefaults(cfg AppConfig) map[string]any {
	defaults := map[string]any{
		"service": cfg.Server.Name,
	}
	if cfg.Project != "" {
		defaults["project"] = cfg.Project
	}
	if cfg.Env != "" {
		defaults["env"] = cfg.Env
	}
	return defaults
}

func parseSampleRate(raw string, fallback float64) float64 {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return fallback
	}
	return v
}

func (c AppConfig) SaaSCoreConfig() saascore.Config {
	stripeEnabled := strings.TrimSpace(c.Billing.Stripe.SecretKey) != "" &&
		strings.TrimSpace(c.Billing.Stripe.WebhookSecret) != ""

	return saascore.Config{
		ServiceName: c.Server.Name,
		Auth: saascore.AuthConfig{
			FrontendRedirect: c.Auth.FrontendRedirect,
			UserJWTSecret:    c.Auth.UserJWTSecret,
			UserJWTExpire:    time.Duration(intWithDefault(c.Auth.UserJWTExpireHours, 168)) * time.Hour,
			WSTicketTTL:      time.Duration(intWithDefault(c.Auth.WSTicketTTLSeconds, 300)) * time.Second,
			AdminEmails:      c.Auth.AdminEmails,
			EmailCode: saascore.EmailCodeConfig{
				Debug:        c.Auth.Email.Debug,
				TTL:          time.Duration(intWithDefault(c.Auth.Email.Code.TTLMinutes, 10)) * time.Minute,
				MinResendGap: time.Duration(intWithDefault(c.Auth.Email.Code.MinResendGapSeconds, 60)) * time.Second,
				DailyCap:     intWithDefault(c.Auth.Email.Code.DailyCap, 10),
				MaxAttempts:  intWithDefault(c.Auth.Email.Code.MaxAttempts, 5),
			},
			Google: saascore.GoogleOAuthConfig{
				ClientID:     c.Auth.Google.ClientID,
				ClientSecret: c.Auth.Google.ClientSecret,
				RedirectURL:  c.Auth.Google.RedirectURL,
				StateSecret:  c.Auth.Google.StateSecret,
				Scope:        c.Auth.Google.Scope,
			},
		},
		Email: saascore.EmailConfig{
			Provider: c.Email.Provider,
			Brevo: saascore.BrevoConfig{
				APIKey:      c.Email.Brevo.APIKey,
				SenderEmail: c.Email.Brevo.SenderEmail,
				SenderName:  c.Email.Brevo.SenderName,
			},
			Resend: saascore.ResendConfig{
				APIKey:      c.Email.Resend.APIKey,
				SenderEmail: c.Email.Resend.SenderEmail,
				SenderName:  c.Email.Resend.SenderName,
			},
		},
		Billing: saascore.BillingConfig{
			Stripe: saascore.StripeConfig{
				Enabled:               stripeEnabled,
				SecretKey:             c.Billing.Stripe.SecretKey,
				PublishableKey:        c.Billing.Stripe.PublishableKey,
				WebhookSecret:         c.Billing.Stripe.WebhookSecret,
				TrialDays:             c.Billing.Stripe.TrialDays,
				StarterMonthlyPriceID: c.Billing.Stripe.Prices.StarterMonthly,
				StarterYearlyPriceID:  c.Billing.Stripe.Prices.StarterYearly,
				ProMonthlyPriceID:     c.Billing.Stripe.Prices.ProMonthly,
				ProYearlyPriceID:      c.Billing.Stripe.Prices.ProYearly,
				PremiumMonthlyPriceID: c.Billing.Stripe.Prices.PremiumMonthly,
				PremiumYearlyPriceID:  c.Billing.Stripe.Prices.PremiumYearly,
				LifetimePriceID:       c.Billing.Stripe.Prices.Lifetime,
				CreditsPriceIDs:       c.Billing.Stripe.Prices.Credits,
				CreditsPerPackage:     c.Billing.Stripe.Credits.PerPackage,
			},
		},
		Referral: saascore.ReferralConfig{
			Prefix:           c.Referral.Prefix,
			BaseLink:         c.Referral.BaseLink,
			ActivationReward: c.Referral.ActivationReward,
		},
	}
}

func intWithDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
