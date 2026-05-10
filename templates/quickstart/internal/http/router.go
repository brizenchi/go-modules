package http

import (
	stdhttp "net/http"

	billinghandler "github.com/brizenchi/quickstart-template/internal/http/handler/billing"
	billingroutes "github.com/brizenchi/quickstart-template/internal/http/routes"
	"github.com/gin-gonic/gin"
)

type RouteRegistrar interface {
	RequireUser() gin.HandlerFunc
	Mount(publicGroup, userGroup *gin.RouterGroup)
}

type Router struct {
	shared RouteRegistrar
	topUp  *billinghandler.StripeTopUpHandler
}

func NewRouter(shared RouteRegistrar, topUp *billinghandler.StripeTopUpHandler) *Router {
	return &Router{shared: shared, topUp: topUp}
}

func (r *Router) RequireUser() gin.HandlerFunc {
	return r.shared.RequireUser()
}

func (r *Router) Mount(publicGroup, userGroup *gin.RouterGroup) {
	r.shared.Mount(publicGroup, userGroup)
	billingroutes.RegisterTopUpRoutes(publicGroup, userGroup, r.topUp)
}

func HealthHandler(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
}
