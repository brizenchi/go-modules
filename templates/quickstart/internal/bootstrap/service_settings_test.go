package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/foundation/config"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestStripeSourceEnvironmentWorksWithOlderCopiedConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Chdir(t.TempDir())
	path := filepath.Join(t.TempDir(), "config.yaml")
	// No config_source key: an existing SaaS should only need the environment switch.
	if err := os.WriteFile(path, []byte("auth:\n  enabled: false\nbilling:\n  enabled: false\nreferral:\n  enabled: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG", path)
	t.Setenv("APP_BILLING_STRIPE_CONFIG_SOURCE", "environment")
	cfg, err := loadConfig(false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Billing.Stripe.ConfigSource != "environment" {
		t.Fatal("deployment source switch was ignored by an older config file")
	}
}

func TestStripeSourceRejectsTyposBeforeProviderInitialization(t *testing.T) {
	for _, source := range []string{"env", "database", "Environment"} {
		t.Run(source, func(t *testing.T) {
			cfg := productionConfigForTest()
			cfg.Billing.Stripe.ConfigSource = source
			if err := cfg.validateDeployment(); err == nil || !strings.Contains(err.Error(), "config_source") {
				t.Fatalf("invalid source was not rejected while billing is disabled: %v", err)
			}
		})
	}
}

func TestDeploymentStripeNeverFallsBackToStoredCredentials(t *testing.T) {
	db := serviceSettingsDB(t)
	// An ignored legacy payload must neither supply credentials nor block startup parsing.
	if err := db.Create(&serviceconfig.Record{Provider: "stripe", Source: "database", Version: 1, Payload: "legacy-payload"}).Error; err != nil {
		t.Fatal(err)
	}
	cfg := productionBillingConfigForTest()
	cfg.Billing.Stripe.ConfigSource = "environment"
	cfg.Billing.Stripe.SecretKey = ""
	if _, err := loadServiceSettings(context.Background(), db, &cfg); err == nil || !strings.Contains(err.Error(), "billing.stripe.secret_key required") {
		t.Fatalf("missing deployment credential did not fail closed: %v", err)
	}
}

func TestStartupCanExplicitlyUseDeploymentStripeInsteadOfLegacyDatabasePrices(t *testing.T) {
	for _, source := range []string{"", "environment"} {
		t.Run("source="+source, func(t *testing.T) {
			db := serviceSettingsDB(t)
			cfg := productionBillingConfigForTest()
			cfg.Env = "dev"
			path := filepath.Join(t.TempDir(), "stripe.yaml")
			if err := os.WriteFile(path, []byte("config_source: \""+source+"\"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.Load(path, "", &cfg.Billing.Stripe); err != nil {
				t.Fatal(err)
			}
			cfg.Billing.Stripe.SecretKey = "sk_test_not-a-real-deployment-key"
			cfg.Billing.Stripe.WebhookSecret = "whsec_not-a-real-deployment-secret"
			cfg.Billing.Stripe.Prices.Lifetime = "price_newLifetime"
			stored := map[string]any{
				"enabled": true,
				"fields":  map[string]string{"mode": "test", "publishable_key": "", "starter_monthly": "", "starter_yearly": "", "pro_monthly": "price_legacyPro", "pro_yearly": "", "premium_monthly": "", "premium_yearly": "", "lifetime": "price_oldLifetime", "credit_price_ids": "", "trial_days": "0", "credits_per_package": "100"},
				"secrets": map[string]string{"secret_key": "sk_test_not-a-real-database-key", "webhook_secret": "whsec_not-a-real-database-secret"},
			}
			raw, err := json.Marshal(stored)
			if err != nil {
				t.Fatal(err)
			}
			row := serviceconfig.Record{Provider: "stripe", Source: "database", Version: 1, Payload: string(raw)}
			if err := db.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
			manager, err := loadServiceSettings(context.Background(), db, &cfg)
			if err != nil {
				t.Fatal(err)
			}
			wantPrice, wantSource, wantKey := "price_oldLifetime", "database", "sk_test_not-a-real-database-key"
			if source == "environment" {
				wantPrice, wantSource, wantKey = "price_newLifetime", "environment", "sk_test_not-a-real-deployment-key"
			}
			if cfg.Billing.Stripe.Prices.Lifetime != wantPrice || cfg.Billing.Stripe.SecretKey != wantKey {
				t.Fatalf("Stripe source %q loaded Lifetime %q, want %q; credentials must use the same source", source, cfg.Billing.Stripe.Prices.Lifetime, wantPrice)
			}
			snapshot, err := manager.Snapshot(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, provider := range snapshot.Providers {
				if provider.Provider == "stripe" && (provider.Source != wantSource || provider.ActiveSource != wantSource || provider.Fields["lifetime"] != wantPrice || provider.RestartRequired) {
					t.Fatal("Stripe status did not match the effective configuration")
				}
			}
			var after serviceconfig.Record
			if err := db.Where("provider = ?", "stripe").Take(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after.Source != row.Source || after.Version != row.Version || after.Payload != row.Payload {
				t.Fatal("startup changed the legacy configuration record")
			}
		})
	}
}

func TestDeploymentValidationDefersOnlyManagedProviderChecks(t *testing.T) {
	cfg := productionBillingConfigForTest()
	cfg.Billing.Stripe.SecretKey = ""
	cfg.Email.Resend.APIKey = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("full validation accepted incomplete provider settings")
	}
	if err := cfg.validateDeployment(); err != nil {
		t.Fatalf("provider checks must wait for database overrides: %v", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*AppConfig)
		want   string
	}{
		{"jwt", func(c *AppConfig) { c.Auth.UserJWTSecret = "short" }, "user_jwt_secret"},
		{"database TLS", func(c *AppConfig) { c.DB.SSLMode = "disable" }, "ssl_mode"},
		{"CORS", func(c *AppConfig) { c.HTTP.AllowedOrigins = "*" }, "allowed_origins"},
		{"administrator password", func(c *AppConfig) { c.Auth.AdminEmail = "owner@example.test"; c.Auth.AdminPassword = "short" }, "admin"},
		{"OAuth", func(c *AppConfig) { c.Auth.Google.Enabled = truePtr() }, "google"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := cfg
			tc.mutate(&copy)
			if err := copy.validateDeployment(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("deployment safety check was deferred: %v", err)
			}
		})
	}
}

func serviceSettingsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "service-settings.sqlite")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if err := db.AutoMigrate(serviceconfig.Models()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStartupAppliesDatabaseProvidersBeforeModuleInitialization(t *testing.T) {
	db := serviceSettingsDB(t)
	cfg := productionConfigForTest()
	// The deployment deliberately has no Resend credential and disables billing.
	// Persisted settings must be loaded before those providers are validated.
	cfg.Email.Resend.APIKey = ""
	rows := map[string]map[string]any{
		"resend": {
			"enabled": true,
			"fields":  map[string]string{"sender_email": "sender@example.test", "sender_name": "Launch", "email_auth_enabled": "true"},
			"secrets": map[string]string{"api_key": "re_1234567890abcdef1234567890"},
		},
		"stripe": {
			"enabled": true,
			"fields":  map[string]string{"mode": "test", "publishable_key": "", "starter_monthly": "", "starter_yearly": "", "pro_monthly": "price_1234567890abcdef", "pro_yearly": "", "premium_monthly": "", "premium_yearly": "", "lifetime": "", "credit_price_ids": "", "trial_days": "0", "credits_per_package": "100"},
			"secrets": map[string]string{"secret_key": "sk_test_1234567890abcdef", "webhook_secret": "whsec_1234567890abcdef"},
		},
	}
	for name, payload := range rows {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&serviceconfig.Record{Provider: name, Source: "database", Version: 1, Payload: string(raw)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	manager, err := loadServiceSettings(context.Background(), db, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ModuleConfig().BillingEnabled() || cfg.Email.Resend.APIKey != "re_1234567890abcdef1234567890" || !cfg.ModuleConfig().EmailAuthEnabled() || cfg.Auth.Email.Debug {
		t.Fatal("effective startup configuration did not apply stored providers")
	}
	if err := platform.Migrate(db, cfg.ModuleConfig()); err != nil {
		t.Fatal(err)
	}
	modules, err := platform.New(db, cfg.ModuleConfig())
	if err != nil {
		t.Fatal(err)
	}
	if modules.Billing == nil || modules.Email == nil || modules.Auth == nil {
		t.Fatal("effective provider modules were not initialized")
	}
	if !db.Migrator().HasTable("billing_subscriptions") {
		t.Fatal("billing enabled through admin settings did not receive its schema")
	}
	status, err := manager.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range status.Providers {
		if provider.Source != "database" || provider.ActiveSource != "database" || !provider.ActiveEnabled || provider.RestartRequired {
			t.Fatal("startup status does not match the active stored settings")
		}
	}
}

func TestStartupWithoutOverridesStillRejectsIncompleteEffectiveConfig(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.Email.Resend.APIKey = ""
	if _, err := loadServiceSettings(context.Background(), serviceSettingsDB(t), &cfg); err == nil {
		t.Fatal("deferred validation allowed incomplete effective email configuration")
	}
}

func TestAdminSaveCannotPersistConfigurationRejectedByDeployment(t *testing.T) {
	db := serviceSettingsDB(t)
	cfg := productionConfigForTest()
	manager, err := loadServiceSettings(context.Background(), db, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		_, err := manager.Update(context.Background(), tx, "stripe", serviceconfig.Patch{
			Version: 0, Source: "database", Enabled: truePtr(),
			Fields:  map[string]string{"mode": "test", "pro_monthly": "price_1234567890abcdef"},
			Secrets: map[string]string{"secret_key": "sk_test_placeholder1234567890", "webhook_secret": "whsec_1234567890abcdef"},
		})
		return err
	})
	if err == nil {
		t.Fatal("admin saved a template credential forbidden by deployment validation")
	}
	if strings.Contains(err.Error(), "placeholder1234567890") {
		t.Fatal("candidate validation returned submitted secret")
	}
	var count int64
	if err := db.Model(&serviceconfig.Record{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("invalid candidate was persisted")
	}
}

func TestProductionStripeRestrictedKeysMatchAdminConfiguration(t *testing.T) {
	for _, prefix := range []string{"rk_test_", "rk_live_"} {
		cfg := productionBillingConfigForTest()
		cfg.Billing.Stripe.SecretKey = prefix + "1234567890abcdef"
		if err := cfg.Validate(); err != nil {
			t.Fatalf("restricted key format rejected: %v", err)
		}
	}
}
