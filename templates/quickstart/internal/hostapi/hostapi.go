// Package hostapi defines the contract between the template's HTTP layer
// and your features: what a feature is given (Deps) and where it can
// mount routes (Groups).
//
// It is a leaf package on purpose — features import it, it imports no
// feature — so adding a feature never creates an import cycle.
//
// TEMPLATE-OWNED, with one exception: adding a field to Deps is expected
// and safe.
package hostapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/brizenchi/quickstart-template/internal/audit"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	"github.com/brizenchi/quickstart-template/internal/platform"
	"github.com/brizenchi/quickstart-template/internal/serviceconfig"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Groups are the mount points available to host routes, one per access
// level. All three share the /api/v1 prefix.
type Groups struct {
	Public *gin.RouterGroup // no auth
	User   *gin.RouterGroup // valid bearer token required
	Admin  *gin.RouterGroup // valid bearer token + admin role required
}

// Deps is everything a host feature can be built from. Add a field here
// when a feature needs something the template already has; never reach
// for package-level globals.
type Deps struct {
	// DB is shared by the host and enabled modules.
	DB *gorm.DB

	// Modules is this SaaS's template-owned combination. Any field can be nil
	// when the corresponding capability is disabled in config.
	Modules *platform.Modules

	// Users is this SaaS's own repository, not a shared user module.
	Users *user.Repository

	// Credits is the shared entry point for credit business operations.
	Credits *credit.Service
	Audit   *audit.Service

	// Config is your own business config from internal/hostcfg.
	Config hostcfg.Config

	// HTTPClient is the shared instrumented outbound client. Features that
	// call third-party APIs should copy it (never mutate it); nil means the
	// feature builds its own default from platform.NewOutboundHTTPClient.
	HTTPClient *http.Client

	// ServiceSettings manages administrator-owned provider configuration. Saved
	// changes apply on the next backend start; nil disables configuration editing.
	ServiceSettings *serviceconfig.Manager
}

// WithServices supports composition roots and existing hosts that supply Users.
// Call once during construction; explicitly injected services are preserved.
func (d Deps) WithServices() Deps {
	if d.Audit == nil {
		d.Audit = audit.New()
	}
	if d.Credits == nil && d.Users != nil {
		recorder := d.Audit
		d.Credits = credit.New(d.Users.WithCreditAudit(func(ctx context.Context, tx *gorm.DB, entry *user.CreditTransaction) error {
			return recorder.Record(ctx, tx, audit.Operation{ActorID: entry.ActorID, IdempotencyKey: fmt.Sprintf("credit:%d", entry.ID), Action: "credits." + entry.Kind, TargetID: entry.UserID, Reason: entry.Reason, RequestHash: audit.Hash(entry.ID)}, map[string]any{"transaction_id": entry.ID, "before": entry.BalanceAfter - entry.Amount, "after": entry.BalanceAfter, "amount": entry.Amount})
		}))
	}
	return d
}
