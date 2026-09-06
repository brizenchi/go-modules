package platform

import (
	"fmt"

	"github.com/brizenchi/go-modules/modules/auth"
	"github.com/brizenchi/go-modules/modules/billing"
	"github.com/brizenchi/go-modules/modules/email"
	"github.com/brizenchi/go-modules/modules/referral"
	"github.com/brizenchi/go-modules/modules/referral/adapter/codegen"
	referraleventbus "github.com/brizenchi/go-modules/modules/referral/adapter/eventbus"
	referralgormrepo "github.com/brizenchi/go-modules/modules/referral/adapter/gormrepo"
	"github.com/brizenchi/quickstart-template/internal/user"
	"gorm.io/gorm"
)

// Modules is this SaaS's assembled set of reusable capabilities.
// It lives in the template so every copied SaaS can change the combination.
type Modules struct {
	Config   Config
	DB       *gorm.DB
	Users    *user.Repository
	Email    *email.Module
	Auth     *auth.Module
	Billing  *billing.Module
	Referral *referral.Module

	emailAuthEnabled bool
	oauthEnabled     bool
	adminPassword    *adminPasswordLogin
}

func New(db *gorm.DB, cfg Config) (*Modules, error) {
	if db == nil {
		return nil, fmt.Errorf("platform: db required")
	}
	cfg = cfg.withDefaults()
	if err := cfg.ValidateAdminPassword(); err != nil {
		return nil, err
	}
	users := user.NewRepository(db)
	emailModule, err := buildEmail(cfg.Email)
	if err != nil {
		return nil, err
	}
	modules := &Modules{Config: cfg, DB: db, Users: users, Email: emailModule}

	if cfg.AuthEnabled() {
		modules.Auth, err = buildAuth(db, cfg, users, emailModule)
		if err != nil {
			return nil, err
		}
		modules.emailAuthEnabled = cfg.EmailAuthEnabled()
		modules.oauthEnabled = len(modules.Auth.Deps.IdentityProviders) > 0
		modules.adminPassword, err = newAdminPasswordLogin(cfg, modules.Auth)
		if err != nil {
			return nil, err
		}
	}
	// Keep only the bcrypt hash in the running authentication handler.
	modules.Config.Auth.AdminPassword = ""
	if cfg.BillingEnabled() {
		modules.Billing, err = buildBilling(db, cfg, users)
		if err != nil {
			return nil, err
		}
	}
	if cfg.ReferralEnabled() {
		modules.Referral = referral.New(referral.Deps{
			Codes:            referralgormrepo.NewCodeRepo(db),
			Referrals:        referralgormrepo.NewReferralRepo(db),
			Generator:        codegen.NewRandom(cfg.Referral.Prefix, 10),
			Bus:              referraleventbus.NewInProc(),
			GetUserID:        userIDFromGin,
			BaseLink:         cfg.Referral.BaseLink,
			ActivationWindow: cfg.ReferralActivationWindow(),
		})
	}
	return modules, nil
}
