package routes

import (
	billinghandler "github.com/brizenchi/quickstart-template/internal/http/handler/billing"
	stripeintegration "github.com/brizenchi/quickstart-template/internal/integration/stripe"
	"github.com/gin-gonic/gin"
)

func RegisterTopUpRoutes(publicGroup, userGroup *gin.RouterGroup, handler *billinghandler.StripeTopUpHandler) {
	if handler == nil {
		return
	}
	publicGroup.Use(handler.WebhookMiddleware())
	userGroup.POST(stripeintegration.TopUpPaymentIntentPath, handler.CreatePaymentIntent)
}
