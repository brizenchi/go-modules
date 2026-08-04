package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type RouteRegistrar interface {
	RequireUser() gin.HandlerFunc
	Mount(publicGroup, userGroup *gin.RouterGroup)
}

type Router struct {
	shared RouteRegistrar
}

func NewRouter(shared RouteRegistrar) *Router {
	return &Router{shared: shared}
}

func (r *Router) RequireUser() gin.HandlerFunc {
	if r == nil || r.shared == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return r.shared.RequireUser()
}

func (r *Router) Mount(publicGroup, userGroup *gin.RouterGroup) {
	if r != nil && r.shared != nil {
		r.shared.Mount(publicGroup, userGroup)
	}
	registerHostRoutes(publicGroup, userGroup)
}

func HealthHandler(c *gin.Context) {
	c.JSON(stdhttp.StatusOK, gin.H{"status": "ok"})
}
