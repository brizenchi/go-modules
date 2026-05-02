package http

import "github.com/gin-gonic/gin"

// Mount attaches the billing routes to the given router groups.
//
//   - publicGroup: receives the webhook endpoint (no auth)
//   - userGroup:   receives the authenticated endpoints; the host should
//     attach its auth middleware to this group before calling Mount.
//
// Conventional paths (relative to the supplied groups):
//
//	POST  {public}/stripe/webhook
//	POST  {user}/stripe/checkout/session
//	POST  {user}/stripe/subscription/preview
//	POST  {user}/stripe/subscription/change
//	POST  {user}/stripe/subscription/cancel
//	POST  {user}/stripe/subscription/reactivate
//	POST  {user}/stripe/portal/session
//	GET   {user}/stripe/subscription
//	GET   {user}/stripe/invoices
//
// Hosts that want different paths can call the handler methods directly
// instead of using Mount.
func Mount(h *Handler, publicGroup, userGroup *gin.RouterGroup) {
	publicGroup.POST("/stripe/webhook", h.HandleWebhook)

	userGroup.POST("/stripe/checkout/session", h.CreateCheckoutSession)
	userGroup.POST("/stripe/subscription/preview", h.PreviewSubscriptionChange)
	userGroup.POST("/stripe/subscription/change", h.ChangeSubscription)
	userGroup.POST("/stripe/subscription/cancel", h.CancelSubscription)
	userGroup.POST("/stripe/subscription/reactivate", h.ReactivateSubscription)
	userGroup.POST("/stripe/portal/session", h.CreateBillingPortalSession)
	userGroup.GET("/stripe/subscription", h.GetSubscription)
	userGroup.GET("/stripe/invoices", h.ListInvoices)
}
