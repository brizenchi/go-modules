package http

import (
	"github.com/brizenchi/go-modules/auth/app"
	"github.com/gin-gonic/gin"
)

// Mount registers auth routes onto two route groups:
//
//	publicGroup — no auth required (signup/login endpoints)
//	userGroup   — must already have RequireUser(session) middleware applied
//
// Mounted paths (relative to each group):
//
//	public:
//	  POST  /auth/send-code
//	  POST  /auth/verify-code
//	  GET   /auth/:provider/authorize
//	  GET   /auth/:provider/callback
//	  POST  /auth/exchange-token
//	user:
//	  POST  /auth/refresh
//	  POST  /auth/logout
//	  POST  /websocket/ticket
//
// Hosts that need different paths can call individual handler methods.
func Mount(h *Handler, publicGroup, userGroup *gin.RouterGroup) {
	publicGroup.POST("/auth/send-code", h.SendCode)
	publicGroup.POST("/auth/verify-code", h.VerifyCode)
	publicGroup.GET("/auth/:provider/authorize", h.StartOAuth)
	publicGroup.GET("/auth/:provider/callback", h.OAuthCallback)
	publicGroup.POST("/auth/exchange-token", h.ExchangeToken)

	userGroup.POST("/auth/refresh", h.Refresh)
	userGroup.POST("/auth/logout", h.Logout)
	userGroup.POST("/websocket/ticket", h.IssueWSTicket)
}

// MiddlewareForUserGroup is a convenience that returns the
// RequireUser middleware that hosts should attach to userGroup
// before calling Mount.
func MiddlewareForUserGroup(session *app.SessionService) gin.HandlerFunc {
	return RequireUser(session)
}
