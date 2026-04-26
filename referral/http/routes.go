package http

import "github.com/gin-gonic/gin"

// Mount registers referral routes on a single user-authenticated group.
//
// Mounted paths (relative to userGroup):
//
//	GET /referral/code   — issue/return user's referral code
//	GET /referral/list   — list pending+activated referrals where the user is the referrer
//	GET /referral/stats  — aggregate counts for the user's dashboard
//
// Hosts that need different paths can call individual handler methods.
func Mount(h *Handler, userGroup *gin.RouterGroup) {
	userGroup.GET("/referral/code", h.GetMyCode)
	userGroup.GET("/referral/list", h.ListMyReferrals)
	userGroup.GET("/referral/stats", h.GetMyStats)
}
