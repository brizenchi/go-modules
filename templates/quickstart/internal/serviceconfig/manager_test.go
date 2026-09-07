package serviceconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/brizenchi/quickstart-template/internal/platform"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testDB(t *testing.T, migrate bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if migrate {
		if err := db.AutoMigrate(Models()...); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func boolean(v bool) *bool { return &v }

func baseConfig() platform.Config {
	return platform.Config{
		Auth:  platform.AuthConfig{Enabled: boolean(true), AdminPassword: "deployment-admin-secret", Email: platform.AuthEmailConfig{Enabled: boolean(true), Debug: true}},
		Email: platform.EmailConfig{Provider: "log", Resend: platform.ResendConfig{APIKey: "re_EnvironmentCredential", SenderEmail: "env@example.com", SenderName: "Environment"}},
		Billing: platform.BillingConfig{Enabled: boolean(false), Stripe: platform.StripeConfig{
			SecretKey: "sk_test_EnvironmentCredential", WebhookSecret: "whsec_EnvironmentCredential", Prices: platform.StripePricesConfig{ProMonthly: "price_Environment"},
		}},
	}
}

func update(t *testing.T, m *Manager, db *gorm.DB, provider string, patch Patch) (Provider, error) {
	t.Helper()
	var result Provider
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = m.Update(context.Background(), tx, provider, patch)
		return err
	})
	return result, err
}

func providerSnapshot(t *testing.T, m *Manager, name string) Provider {
	t.Helper()
	result, err := m.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, provider := range result.Providers {
		if provider.Provider == name {
			return provider
		}
	}
	t.Fatal("missing provider")
	return Provider{}
}

func TestPlaintextWriteOnlyPreservationAndRestartIsolation(t *testing.T) {
	db := testDB(t, true)
	env := baseConfig()
	m := NewManager(db, env, false)
	active, err := m.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	const key = "re_PrivateDatabaseCredential"
	result, err := update(t, m, db, "resend", Patch{Source: "database", Enabled: boolean(true), Secrets: map[string]string{"api_key": key}, Fields: map[string]string{"sender_email": "hello@example.com", "email_auth_enabled": "true"}})
	if err != nil || !result.RestartRequired || result.ActiveEnabled || !result.Enabled || !result.Secrets["api_key"].Configured {
		t.Fatalf("first save status=%+v error=%v", result, err)
	}
	if active.Email.Provider != "log" || active.Auth.AdminPassword != env.Auth.AdminPassword || active.Auth.Email.Debug != true {
		t.Fatal("save changed the active configuration")
	}
	result, err = update(t, m, db, "resend", Patch{Version: 1, Source: "database", Fields: map[string]string{"sender_name": "New Brand"}})
	if err != nil || result.Version != 2 {
		t.Fatal(err)
	}
	var row Record
	if err := db.Where("provider = ?", "resend").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(row.Payload, key) || strings.Contains(row.Payload, env.Auth.AdminPassword) {
		t.Fatal("credential is not stored as intended or deployment secret was copied")
	}
	for _, safe := range []any{row, result, providerSnapshot(t, m, "resend")} {
		raw, _ := json.Marshal(safe)
		if strings.Contains(string(raw), key) || strings.Contains(string(raw), "re_EnvironmentCredential") {
			t.Fatal("credential leaked through a JSON representation")
		}
	}
	restarted := NewManager(db, env, false)
	loaded, err := restarted.Load(context.Background())
	if err != nil || loaded.Email.Provider != "resend" || loaded.Email.Resend.APIKey != key || loaded.Auth.Email.Debug || !loaded.EmailAuthEnabled() {
		t.Fatalf("restart did not use saved settings: %v", err)
	}
	if next := providerSnapshot(t, restarted, "resend"); next.RestartRequired || !next.ActiveEnabled || next.ActiveSource != "database" {
		t.Fatalf("active status after restart: %+v", next)
	}
	if _, err := update(t, restarted, db, "resend", Patch{Version: 2, Source: "database", Secrets: map[string]string{"api_key": ""}}); err == nil {
		t.Fatal("cleared an enabled credential")
	}
	cleared, err := update(t, restarted, db, "resend", Patch{Version: 2, Source: "database", Enabled: boolean(false), Secrets: map[string]string{"api_key": ""}})
	if err != nil || cleared.Secrets["api_key"].Configured || cleared.Fields["email_auth_enabled"] != "false" || !cleared.ActiveEnabled {
		t.Fatalf("disabled clear: %+v %v", cleared, err)
	}
}

func TestDatabaseSnapshotCopiesOmittedEnvironmentSecretAndResetRestoresEnvironment(t *testing.T) {
	db := testDB(t, true)
	env := baseConfig()
	m := NewManager(db, env, false)
	if _, err := m.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, err := update(t, m, db, "stripe", Patch{Source: "database", Enabled: boolean(true), Fields: map[string]string{"pro_monthly": "price_Database"}})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewManager(db, env, false)
	loaded, err := restarted.Load(context.Background())
	if err != nil || loaded.Billing.Stripe.SecretKey != env.Billing.Stripe.SecretKey || loaded.Billing.Stripe.Prices.ProMonthly != "price_Database" || !loaded.BillingEnabled() {
		t.Fatal("omitted environment credential was not retained")
	}
	result, err := update(t, restarted, db, "stripe", Patch{Version: 1, Source: "environment"})
	if err != nil || result.Enabled || !result.ActiveEnabled || !result.RestartRequired || result.Source != "environment" {
		t.Fatalf("reset response %+v %v", result, err)
	}
	var row Record
	db.Where("provider = ?", "stripe").Take(&row)
	if row.Payload != "" || row.Version != 2 {
		t.Fatal("reset retained the database credential payload")
	}
	loaded, err = NewManager(db, env, false).Load(context.Background())
	if err != nil || loaded.BillingEnabled() || loaded.Billing.Stripe.Prices.ProMonthly != "price_Environment" {
		t.Fatal("reset did not restore original environment")
	}
}

func TestPreviewNeverActivatesSavedExternalProviders(t *testing.T) {
	db := testDB(t, true)
	env := baseConfig()
	m := NewManager(db, env, true)
	for _, provider := range providerNames {
		if _, err := update(t, m, db, provider, Patch{Source: "database", Enabled: boolean(true)}); err != nil {
			t.Fatal(err)
		}
	}
	preview := NewManager(db, env, true)
	loaded, err := preview.Load(context.Background())
	if err != nil || loaded.BillingEnabled() || loaded.Email.Provider != "log" {
		t.Fatal("isolated preview activated an external provider")
	}
	result, err := preview.Snapshot(context.Background())
	if err != nil || !result.LocalPreview {
		t.Fatal(err)
	}
	for _, provider := range result.Providers {
		if provider.ActiveEnabled || !provider.Enabled || !provider.RestartRequired {
			t.Fatalf("preview status misrepresented: %+v", provider)
		}
	}
}

func TestSchemaMissingIsDifferentFromDatabaseUnavailable(t *testing.T) {
	db := testDB(t, false)
	m := NewManager(db, baseConfig(), false)
	if _, err := m.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	result, err := m.Snapshot(context.Background())
	if err != nil || result.StorageReady {
		t.Fatal("missing schema did not preserve environment fallback")
	}
	if _, err := update(t, m, db, "stripe", Patch{Source: "database"}); !errors.Is(err, ErrSchema) {
		t.Fatalf("missing schema write: %v", err)
	}
	pool, _ := db.DB()
	_ = pool.Close()
	if _, err := m.Load(context.Background()); !errors.Is(err, ErrStorage) {
		t.Fatalf("connection failure silently fell back to environment: %v", err)
	}
	if _, err := m.Snapshot(context.Background()); !errors.Is(err, ErrStorage) {
		t.Fatal("unavailable database was reported as a missing schema")
	}
}

func TestValidationAndOptimisticConflictsDoNotModifyRows(t *testing.T) {
	db := testDB(t, true)
	m := NewManager(db, baseConfig(), false)
	invalid := []struct {
		provider string
		patch    Patch
	}{
		{"stripe", Patch{Source: "database", Fields: map[string]string{"mode": "live"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"publishable_key": "pk_live_WrongMode"}}},
		{"stripe", Patch{Source: "database", Secrets: map[string]string{"webhook_secret": "bad"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"pro_monthly": "prod_wrong"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"lifetime": "price_Environment"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"credit_price_ids": "price_One,price_One"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"trial_days": "731"}}},
		{"stripe", Patch{Source: "database", Fields: map[string]string{"credits_per_package": "0"}}},
		{"stripe", Patch{Source: "database", Enabled: boolean(true), Fields: map[string]string{"pro_monthly": ""}}},
		{"stripe", Patch{Source: "database", Secrets: map[string]string{"admin_password": "private"}}},
		{"resend", Patch{Source: "database", Enabled: boolean(true), Secrets: map[string]string{"api_key": ""}}},
		{"resend", Patch{Source: "database", Fields: map[string]string{"sender_email": "Name <owner@example.com>"}}},
		{"resend", Patch{Source: "database", Fields: map[string]string{"sender_name": strings.Repeat("字", 101)}}},
		{"resend", Patch{Source: "database", Secrets: map[string]string{"api_key": "re_a\nb"}}},
		{"resend", Patch{Source: "environment", Enabled: boolean(false)}},
		{"resend", Patch{Source: "file"}},
	}
	for _, item := range invalid {
		if _, err := update(t, m, db, item.provider, item.patch); err == nil {
			t.Fatalf("invalid patch accepted for %s", item.provider)
		}
	}
	var count int64
	db.Model(&Record{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid configuration persisted")
	}
	if _, err := update(t, m, db, "stripe", Patch{Source: "database"}); err != nil {
		t.Fatal(err)
	}
	if _, err := update(t, m, db, "stripe", Patch{Source: "database", Fields: map[string]string{"trial_days": "7"}}); !errors.Is(err, ErrVersion) {
		t.Fatalf("stale version accepted: %v", err)
	}
	if _, err := update(t, m, db, "unknown", Patch{Source: "database"}); !errors.Is(err, ErrProvider) {
		t.Fatalf("unknown provider: %v", err)
	}
}

func TestSQLLoggingAndHostValidationNeverExposeCredentials(t *testing.T) {
	db := testDB(t, true)
	var logs bytes.Buffer
	loud := db.Session(&gorm.Session{Logger: logger.New(log.New(&logs, "", 0), logger.Config{LogLevel: logger.Info})})
	m := NewManager(loud, baseConfig(), false)
	m.SetValidator(func(platform.Config) error { return errors.New("unexpected secret sk_test_DoNotExposeThis") })
	if _, err := update(t, m, loud, "stripe", Patch{Source: "database", Secrets: map[string]string{"secret_key": "sk_test_DoNotExposeThis"}}); err == nil || strings.Contains(err.Error(), "DoNotExposeThis") {
		t.Fatal("host validator did not reject safely")
	}
	m.SetValidator(nil)
	if _, err := update(t, m, loud, "stripe", Patch{Source: "database", Secrets: map[string]string{"secret_key": "sk_test_DoNotExposeThis"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "DoNotExposeThis") || strings.Contains(logs.String(), "service_provider_settings") {
		t.Fatal("service configuration used the deployment SQL logger")
	}
}
