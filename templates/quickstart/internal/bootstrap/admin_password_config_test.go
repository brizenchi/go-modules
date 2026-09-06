package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	foundationconfig "github.com/brizenchi/go-modules/foundation/config"
)

func TestAdminPasswordEnvironmentWorksWithExistingConfiguration(t *testing.T) {
	// Older copied config files have no admin_email/admin_password YAML keys.
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("auth:\n  user_jwt_secret: test-jwt\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_AUTH_ADMIN_EMAIL", "owner@example.test")
	t.Setenv("APP_AUTH_ADMIN_PASSWORD", "Isolated-config-pass-42!")
	var cfg AppConfig
	if err := foundationconfig.Load(path, "QUICKSTART_ADMIN_ENV_TEST", &cfg); err != nil {
		t.Fatal(err)
	}
	applyAdminPasswordEnvironment(&cfg)
	if cfg.Auth.AdminEmail != "owner@example.test" || cfg.Auth.AdminPassword != "Isolated-config-pass-42!" {
		t.Fatal("admin credential environment overrides were not applied")
	}
	// An explicit empty environment value also clears a stale file value.
	t.Setenv("APP_AUTH_ADMIN_EMAIL", "")
	t.Setenv("APP_AUTH_ADMIN_PASSWORD", "")
	applyAdminPasswordEnvironment(&cfg)
	if cfg.Auth.AdminEmail != "" || cfg.Auth.AdminPassword != "" {
		t.Fatal("empty environment did not disable file credentials")
	}
}

func TestAdminPasswordValidationRunsDuringStartupConfigValidation(t *testing.T) {
	for _, tc := range []struct {
		name, email, password string
		valid                 bool
	}{
		{name: "unset", valid: true},
		{name: "valid", email: "owner@example.test", password: "Isolated-config-pass-42!", valid: true},
		{name: "partial", email: "owner@example.test"},
		{name: "bad email", email: "not-an-email", password: "Isolated-config-pass-42!"},
		{name: "weak", email: "owner@example.test", password: "short"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := productionConfigForTest()
			cfg.Auth.AdminEmail, cfg.Auth.AdminPassword = tc.email, tc.password
			err := cfg.Validate()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
			if err != nil && !strings.Contains(err.Error(), "auth.admin_") {
				t.Fatalf("wrong config error: %v", err)
			}
		})
	}
}
