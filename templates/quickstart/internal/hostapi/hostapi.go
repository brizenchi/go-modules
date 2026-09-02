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
	"github.com/brizenchi/quickstart-template/internal/hostcfg"
	"github.com/brizenchi/quickstart-template/internal/platform"
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

	// Config is your own business config from internal/hostcfg.
	Config hostcfg.Config
}
