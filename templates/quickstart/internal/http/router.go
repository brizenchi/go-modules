// TEMPLATE-OWNED — avoid editing; changes here conflict on upgrade.
// Register your own routes in host_routes.go instead.
package http

import (
	stdhttp "net/http"

	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/gin-gonic/gin"
)

// PlatformRoutes is the subset of platform.Modules consumed by HTTP wiring.
type PlatformRoutes interface {
	RequireUser() gin.HandlerFunc
	RequireAdmin() gin.HandlerFunc
	Mount(publicGroup, userGroup *gin.RouterGroup)
}

type Router struct {
	platform PlatformRoutes
	deps     hostapi.Deps
}

func NewRouter(platform PlatformRoutes, deps hostapi.Deps) *Router {
	return &Router{platform: platform, deps: deps}
}

func (r *Router) RequireUser() gin.HandlerFunc {
	if r == nil || r.platform == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return r.platform.RequireUser()
}

func (r *Router) RequireAdmin() gin.HandlerFunc {
	if r == nil || r.platform == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return r.platform.RequireAdmin()
}

func (r *Router) Mount(g hostapi.Groups) {
	if r != nil && r.platform != nil {
		r.platform.Mount(g.Public, g.User)
	}
	registerHostRoutes(r.deps, g)
}

func HealthHandler(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
}
