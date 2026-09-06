package bootstrap

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/config"
	"github.com/brizenchi/go-modules/foundation/pgx"
	stripeadapter "github.com/brizenchi/go-modules/modules/billing/adapter/stripe"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	"github.com/brizenchi/quickstart-template/internal/platform"
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
	HTTP     HTTPConfig              `mapstructure:"http"`
	Tracing  TracingConfig           `mapstructure:"tracing"`
	DB       DBConfig                `mapstructure:"db"`
	Auth     platform.AuthConfig     `mapstructure:"auth"`
	Email    platform.EmailConfig    `mapstructure:"email"`
	Billing  platform.BillingConfig  `mapstructure:"billing"`
	Referral platform.ReferralConfig `mapstructure:"referral"`

	// Host is your own business configuration. Add fields in
	// internal/hostcfg instead of editing this file.
	Host hostcfg.Config `mapstructure:"host"`
}

type HTTPConfig struct {
	// AllowedOrigins is a comma-separated list so it can be overridden by one
	// environment variable: APP_HTTP_ALLOWED_ORIGINS.
	AllowedOrigins          string `mapstructure:"allowed_origins"`
	ReadHeaderTimeoutSecond int    `mapstructure:"read_header_timeout_seconds"`
	ReadTimeoutSeconds      int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds     int    `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds      int    `mapstructure:"idle_timeout_seconds"`
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
	if err := cfg.Validate(); err != nil {
		return AppConfig{}, err
	}
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
	if strings.TrimSpace(cfg.HTTP.AllowedOrigins) == "" {
		cfg.HTTP.AllowedOrigins = frontendOriginOrDefault(cfg.Auth.FrontendRedirect)
	}
	if strings.TrimSpace(cfg.Referral.BaseLink) == "" {
		cfg.Referral.BaseLink = frontendOriginOrDefault(cfg.Auth.FrontendRedirect) + "/invite?ref="
	}
	if cfg.HTTP.ReadHeaderTimeoutSecond <= 0 {
		cfg.HTTP.ReadHeaderTimeoutSecond = 10
	}
	if cfg.HTTP.ReadTimeoutSeconds <= 0 {
		cfg.HTTP.ReadTimeoutSeconds = 15
	}
	if cfg.HTTP.WriteTimeoutSeconds <= 0 {
		cfg.HTTP.WriteTimeoutSeconds = 30
	}
	if cfg.HTTP.IdleTimeoutSeconds <= 0 {
		cfg.HTTP.IdleTimeoutSeconds = 60
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

func (c AppConfig) Validate() error {
	modules := c.ModuleConfig()
	production := isProduction(c.Env)
	if production {
		if err := validateProductionDBTLS(c.DB); err != nil {
			return err
		}
	}

	origins := c.HTTP.AllowedOriginList()
	if len(origins) == 0 {
		return fmt.Errorf("http.allowed_origins required")
	}
	if production {
		for _, origin := range origins {
			if origin == "*" {
				return fmt.Errorf("http.allowed_origins must not contain * in production")
			}
			if err := requireHTTPS("http.allowed_origins", origin); err != nil {
				return err
			}
		}
	}
	if !modules.AuthEnabled() {
		if modules.BillingEnabled() {
			return fmt.Errorf("billing requires auth.enabled=true")
		}
		if modules.ReferralEnabled() {
			return fmt.Errorf("referral requires auth.enabled=true")
		}
	}

	if modules.AuthEnabled() {
		secret := strings.TrimSpace(c.Auth.UserJWTSecret)
		if secret == "" {
			return fmt.Errorf("auth.user_jwt_secret required when auth enabled")
		}
		if production && unsafeSecret(secret) {
			return fmt.Errorf("auth.user_jwt_secret must be at least 32 random characters and must not be a template value")
		}
		if production && c.Auth.Email.Debug {
			return fmt.Errorf("auth.email.debug must be false in production")
		}
	}

	if modules.EmailAuthEnabled() {
		provider := strings.ToLower(strings.TrimSpace(c.Email.Provider))
		if provider == "" {
			provider = "log"
		}
		if production && (provider == "log" || provider == "none" || provider == "disabled" || provider == "off") {
			return fmt.Errorf("email.provider must be resend or brevo when email login is enabled in production")
		}
		if err := validateEmailProvider(provider, c.Email, production); err != nil {
			return err
		}
	}

	if modules.GoogleEnabled() {
		if err := validateOAuth("google", c.Auth.Google.ClientID, c.Auth.Google.ClientSecret, c.Auth.Google.RedirectURL, c.Auth.Google.StateSecret, c.Auth.FrontendRedirect, c.Auth.UserJWTSecret, production); err != nil {
			return err
		}
	}
	if modules.GitHubEnabled() {
		if err := validateOAuth("github", c.Auth.GitHub.ClientID, c.Auth.GitHub.ClientSecret, c.Auth.GitHub.RedirectURL, c.Auth.GitHub.StateSecret, c.Auth.FrontendRedirect, c.Auth.UserJWTSecret, production); err != nil {
			return err
		}
	}
	if modules.GoogleEnabled() || modules.GitHubEnabled() {
		if err := validateOAuthFrontendCORS(c.Auth.FrontendRedirect, origins); err != nil {
			return err
		}
		if production && !modules.OAuthFlowCookieSecure() {
			return fmt.Errorf("auth.oauth_cookie_secure must be true in production (omit it to derive from HTTPS callback URLs)")
		}
	}

	if modules.BillingEnabled() {
		if err := validateOAuthFrontendCORS(c.Auth.FrontendRedirect, origins); err != nil {
			return fmt.Errorf("billing return URL origin: %w", err)
		}
		stripe := c.Billing.Stripe
		if strings.TrimSpace(stripe.SecretKey) == "" {
			return fmt.Errorf("billing.stripe.secret_key required when billing enabled")
		}
		if strings.TrimSpace(stripe.WebhookSecret) == "" {
			return fmt.Errorf("billing.stripe.webhook_secret required when billing enabled")
		}
		if !hasStripePrice(stripe.Prices) {
			return fmt.Errorf("billing.stripe.prices must configure at least one Checkout price")
		}
		if production {
			if err := validateStripeCredentials(stripe.SecretKey, stripe.WebhookSecret); err != nil {
				return err
			}
			if err := validateStripePriceIDs(stripe.Prices); err != nil {
				return err
			}
		}
	}

	if modules.ReferralEnabled() {
		if err := validateReferralBaseLink(c.Referral.BaseLink, production); err != nil {
			return err
		}
	}

	return nil
}

func (c HTTPConfig) AllowedOriginList() []string {
	parts := strings.Split(c.AllowedOrigins, ",")
	origins := make([]string, 0, len(parts))
	for _, value := range parts {
		if value = strings.TrimSpace(value); value != "" {
			origins = append(origins, value)
		}
	}
	return origins
}

func frontendOriginOrDefault(frontendRedirect string) string {
	origin, err := absoluteOrigin(frontendRedirect)
	if err == nil {
		return origin
	}
	return "http://localhost:3000"
}

func validateOAuthFrontendCORS(frontendRedirect string, allowedOrigins []string) error {
	origin, err := absoluteOrigin(frontendRedirect)
	if err != nil {
		return fmt.Errorf("auth.frontend_redirect must be an absolute URL for CORS validation: %w", err)
	}
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return nil
		}
	}
	return fmt.Errorf("http.allowed_origins must explicitly include OAuth frontend origin %q; set APP_HTTP_ALLOWED_ORIGINS=%s", origin, origin)
}

func absoluteOrigin(rawURL string) (string, error) {
	if rawURL == "" || rawURL != strings.TrimSpace(rawURL) || strings.HasPrefix(rawURL, "//") {
		return "", fmt.Errorf("absolute http(s) URL required")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" {
		return "", fmt.Errorf("absolute http(s) URL without userinfo required")
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), nil
}

func isProduction(env string) bool {
	env = strings.ToLower(strings.TrimSpace(env))
	return env == "prod" || env == "production"
}

func validateProductionDBTLS(cfg DBConfig) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.SSLMode))
	if dsn := strings.TrimSpace(cfg.DSN); dsn != "" {
		mode = ""
		if parsed, err := url.Parse(dsn); err == nil &&
			(strings.EqualFold(parsed.Scheme, "postgres") || strings.EqualFold(parsed.Scheme, "postgresql")) {
			mode = strings.ToLower(strings.TrimSpace(parsed.Query().Get("sslmode")))
		} else {
			for _, field := range strings.Fields(dsn) {
				key, value, ok := strings.Cut(field, "=")
				if ok && strings.EqualFold(strings.TrimSpace(key), "sslmode") {
					mode = strings.ToLower(strings.Trim(strings.TrimSpace(value), "'\""))
					break
				}
			}
		}
	}

	switch mode {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return fmt.Errorf("db.ssl_mode must be require, verify-ca, or verify-full in production")
	}
}

func unsafeSecret(value string) bool {
	return len(strings.TrimSpace(value)) < 32 || unsafeCredential(value)
}

func unsafeCredential(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{"change-me", "changeme", "placeholder", "example", "test-secret", "your-secret"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func validateEmailProvider(provider string, cfg platform.EmailConfig, production bool) error {
	var apiKey, sender string
	switch provider {
	case "log":
		return nil
	case "resend":
		apiKey, sender = cfg.Resend.APIKey, cfg.Resend.SenderEmail
	case "brevo":
		apiKey, sender = cfg.Brevo.APIKey, cfg.Brevo.SenderEmail
	case "none", "disabled", "off":
		return fmt.Errorf("email.provider cannot be disabled when email login is enabled")
	default:
		return fmt.Errorf("unsupported email.provider %q", provider)
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("email.%s.api_key required", provider)
	}
	if production && unsafeCredential(apiKey) {
		return fmt.Errorf("email.%s.api_key must not use a template value in production", provider)
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(sender)); err != nil {
		return fmt.Errorf("email.%s.sender_email invalid: %w", provider, err)
	}
	return nil
}

func validateOAuth(provider, clientID, clientSecret, redirectURL, stateSecret, frontendRedirect, jwtSecret string, production bool) error {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return fmt.Errorf("auth.%s client_id and client_secret required when enabled", provider)
	}
	if production && (unsafeCredential(clientID) || unsafeCredential(clientSecret)) {
		return fmt.Errorf("auth.%s client_id and client_secret must not use template values in production", provider)
	}
	if strings.TrimSpace(redirectURL) == "" || strings.TrimSpace(frontendRedirect) == "" {
		return fmt.Errorf("auth.%s redirect_url and auth.frontend_redirect required when enabled", provider)
	}
	if strings.TrimSpace(stateSecret) == "" {
		return fmt.Errorf("auth.%s.state_secret required when enabled", provider)
	}
	if stateSecret == jwtSecret {
		return fmt.Errorf("auth.%s.state_secret must differ from auth.user_jwt_secret", provider)
	}
	if production {
		if unsafeSecret(stateSecret) {
			return fmt.Errorf("auth.%s.state_secret must be at least 32 random characters and must not be a template value", provider)
		}
		if err := requireHTTPS("auth."+provider+".redirect_url", redirectURL); err != nil {
			return err
		}
		if err := requireHTTPS("auth.frontend_redirect", frontendRedirect); err != nil {
			return err
		}
	}
	return nil
}

func requireHTTPS(name, value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute https URL in production", name)
	}
	return nil
}

func validateReferralBaseLink(value string, production bool) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("referral.base_link must be an absolute http(s) URL")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("referral.base_link must not contain a fragment")
	}
	if production {
		return requireHTTPS("referral.base_link", value)
	}
	return nil
}

func hasStripePrice(prices platform.StripePricesConfig) bool {
	values := []string{
		prices.StarterMonthly, prices.StarterYearly,
		prices.ProMonthly, prices.ProYearly,
		prices.PremiumMonthly, prices.PremiumYearly,
		prices.Lifetime,
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	for _, value := range prices.Credits {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func validateStripeCredentials(secretKey, webhookSecret string) error {
	if unsafeCredential(secretKey) || !validStripeSecretKey(secretKey) {
		return fmt.Errorf("billing.stripe.secret_key must be a real Stripe secret key (sk_live_... or sk_test_...) in production")
	}
	if unsafeCredential(webhookSecret) || !validStripeCredential(webhookSecret, "whsec_") {
		return fmt.Errorf("billing.stripe.webhook_secret must be a real Stripe signing secret (whsec_...) in production")
	}
	return nil
}

func validStripeSecretKey(value string) bool {
	return validStripeCredential(value, "sk_live_") || validStripeCredential(value, "sk_test_")
}

func validStripeCredential(value, prefix string) bool {
	if value != strings.TrimSpace(value) || !strings.HasPrefix(value, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(value, prefix)
	if len(suffix) < 8 {
		return false
	}
	for _, char := range suffix {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func validateStripePriceIDs(prices platform.StripePricesConfig) error {
	configured := []struct {
		name  string
		value string
	}{
		{name: "starter_monthly", value: prices.StarterMonthly},
		{name: "starter_yearly", value: prices.StarterYearly},
		{name: "pro_monthly", value: prices.ProMonthly},
		{name: "pro_yearly", value: prices.ProYearly},
		{name: "premium_monthly", value: prices.PremiumMonthly},
		{name: "premium_yearly", value: prices.PremiumYearly},
		{name: "lifetime", value: prices.Lifetime},
	}
	seen := make(map[string]string, len(configured)+len(prices.Credits))
	validateUnique := func(name, priceID string) error {
		if strings.TrimSpace(priceID) == "" {
			return nil
		}
		if !stripeadapter.ValidPriceID(priceID) {
			return fmt.Errorf("billing.stripe.prices.%s must be a real Stripe Price ID (price_...)", name)
		}
		if previous, exists := seen[priceID]; exists {
			return fmt.Errorf("billing.stripe.prices.%s duplicates %s; every subscription, lifetime, and credits Price ID must be unique", name, previous)
		}
		seen[priceID] = name
		return nil
	}
	for _, price := range configured {
		if err := validateUnique(price.name, price.value); err != nil {
			return err
		}
	}
	for index, priceID := range prices.Credits {
		if err := validateUnique(fmt.Sprintf("credits[%d]", index), priceID); err != nil {
			return err
		}
	}
	return nil
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
