package bootstrap

import (
	"context"
	"fmt"

	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"gorm.io/gorm"
)

// loadServiceSettings is separate from New so startup precedence and validation
// can be verified with a temporary SQLite database, without deployment services.
func loadServiceSettings(ctx context.Context, db *gorm.DB, cfg *AppConfig) (*serviceconfig.Manager, error) {
	deployment := *cfg
	manager := serviceconfig.NewManager(db, cfg.ModuleConfig(), false)
	manager.SetValidator(func(candidate platform.Config) error {
		check := deployment
		check.Auth, check.Email, check.Billing = candidate.Auth, candidate.Email, candidate.Billing
		return check.Validate()
	})
	effective, err := manager.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load service settings: %w", err)
	}
	cfg.Auth, cfg.Email, cfg.Billing = effective.Auth, effective.Email, effective.Billing
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("effective config: %w", err)
	}
	return manager, nil
}
