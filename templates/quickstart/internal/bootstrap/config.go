package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/config"
	"github.com/brizenchi/go-modules/foundation/pgx"
	"github.com/subosito/gotenv"
)

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
	StateTTLMin  int    `mapstructure:"state_ttl_minutes"`
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
	TopUp          StripeTopUpConfig   `mapstructure:"topup"`
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

type StripeTopUpConfig struct {
	MinAmountUSD  float64 `mapstructure:"min_amount_usd"`
	MaxAmountUSD  float64 `mapstructure:"max_amount_usd"`
	CreditsPerUSD int64   `mapstructure:"credits_per_usd"`
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

func LoadConfig() (AppConfig, error) {
	if err := LoadDotEnv(".env"); err != nil {
		return AppConfig{}, err
	}

	path := os.Getenv("CONFIG")
	if path == "" {
		path = "deploy/config.yaml"
	}

	var cfg AppConfig
	if err := config.LoadGlobal(path, "APP", &cfg); err != nil {
		return AppConfig{}, err
	}

	applyDefaults(&cfg)
	return cfg, nil
}

func LoadDotEnv(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return gotenv.Load(path)
}

func applyDefaults(cfg *AppConfig) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Name == "" {
		cfg.Server.Name = "quickstart"
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

func (c TracingConfig) ExporterHeaders() map[string]string {
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
