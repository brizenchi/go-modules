package platform

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	stripeadapter "github.com/brizenchi/go-modules/modules/billing/adapter/stripe"
	billinghttp "github.com/brizenchi/go-modules/modules/billing/http"
	referralhttp "github.com/brizenchi/go-modules/modules/referral/http"
	"github.com/gin-gonic/gin"
)

type referralCodeContextKey struct{}

// WithReferralCode carries a normalized referral code through the final auth
// operation so template-owned signup/login listeners can persist attribution.
func WithReferralCode(ctx context.Context, code string) context.Context {
	code = strings.TrimSpace(code)
	if code == "" {
		return ctx
	}
	return context.WithValue(ctx, referralCodeContextKey{}, code)
}

// ReferralCodeFromContext is used by the template's signup listener.
func ReferralCodeFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	code, _ := ctx.Value(referralCodeContextKey{}).(string)
	return strings.TrimSpace(code)
}

type verifyCodeRequest struct {
	Email        string `json:"email"`
	Code         string `json:"code"`
	ReferralCode string `json:"referral_code"`
}

type exchangeTokenRequest struct {
	Code          string `json:"code"`
	OAuthVerifier string `json:"oauth_verifier"`
	ReferralCode  string `json:"referral_code"`
}

type capabilityState struct {
	Enabled bool `json:"enabled"`
}

type authCapability struct {
	EmailEnabled   bool     `json:"email_enabled"`
	OAuthProviders []string `json:"oauth_providers"`
}

type billingCapability struct {
	Enabled  bool          `json:"enabled"`
	Provider string        `json:"provider"`
	Offers   billingOffers `json:"offers"`
}

type billingOffers struct {
	Subscriptions []subscriptionOffer `json:"subscriptions"`
	Lifetime      bool                `json:"lifetime"`
	Credits       bool                `json:"credits"`
}

type subscriptionOffer struct {
	Plan      string   `json:"plan"`
	Intervals []string `json:"intervals"`
}

// Mount exposes enabled capabilities, a public discovery endpoint, and a
// structured 503 fallback for billing when no provider is configured.
func (m *Modules) Mount(publicGroup, userGroup *gin.RouterGroup) {
	if m == nil {
		return
	}
	publicGroup.GET("/capabilities", m.getCapabilities)
	if m.Auth != nil {
		if m.emailAuthEnabled {
			publicGroup.POST("/auth/send-code", m.Auth.Handler.SendCode)
			publicGroup.POST("/auth/verify-code", m.verifyCode())
		}
		if m.oauthEnabled {
			publicGroup.GET("/auth/:provider/authorize", m.Auth.Handler.StartOAuth)
			publicGroup.GET("/auth/:provider/callback", m.Auth.Handler.OAuthCallback)
			publicGroup.POST("/auth/exchange-token", m.exchangeToken())
		}
		userGroup.POST("/auth/refresh", m.Auth.Handler.Refresh)
		userGroup.POST("/auth/logout", m.Auth.Handler.Logout)
		userGroup.POST("/websocket/ticket", m.Auth.Handler.IssueWSTicket)
	}
	if m.Billing != nil {
		billinghttp.Mount(m.Billing.Handler, publicGroup, userGroup)
	} else {
		mountBillingUnavailable(publicGroup, userGroup)
	}
	if m.Referral != nil {
		referralhttp.Mount(m.Referral.Handler, userGroup)
	}
}

func (m *Modules) getCapabilities(c *gin.Context) {
	offers := billingOffers{Subscriptions: make([]subscriptionOffer, 0)}
	if m.Billing != nil {
		offers = configuredBillingOffers(m.Config.Billing.Stripe.Prices)
	}
	httpresp.OK(c, gin.H{
		"auth":    m.configuredAuthCapability(),
		"account": capabilityState{Enabled: m.Auth != nil && m.Users != nil},
		"billing": billingCapability{
			Enabled:  m.Billing != nil,
			Provider: strings.ToLower(strings.TrimSpace(m.Config.Billing.Provider)),
			Offers:   offers,
		},
		"referral": capabilityState{Enabled: m.Referral != nil},
	})
}

func (m *Modules) configuredAuthCapability() authCapability {
	capability := authCapability{OAuthProviders: make([]string, 0, 2)}
	if m == nil || m.Auth == nil {
		return capability
	}

	capability.EmailEnabled = m.emailAuthEnabled
	// Keep the public contract deterministic instead of exposing Go map order.
	// These are the providers the quickstart frontend knows how to render.
	for _, provider := range []string{"google", "github"} {
		if _, ok := m.Auth.Deps.IdentityProviders[provider]; ok {
			capability.OAuthProviders = append(capability.OAuthProviders, provider)
		}
	}
	return capability
}

func configuredBillingOffers(prices StripePricesConfig) billingOffers {
	offers := billingOffers{
		Subscriptions: make([]subscriptionOffer, 0, 3),
		Lifetime:      stripeadapter.ValidPriceID(prices.Lifetime),
	}
	for _, value := range prices.Credits {
		if stripeadapter.ValidPriceID(value) {
			offers.Credits = true
			break
		}
	}
	for _, plan := range []struct {
		name    string
		monthly string
		yearly  string
	}{
		{name: "starter", monthly: prices.StarterMonthly, yearly: prices.StarterYearly},
		{name: "pro", monthly: prices.ProMonthly, yearly: prices.ProYearly},
		{name: "premium", monthly: prices.PremiumMonthly, yearly: prices.PremiumYearly},
	} {
		intervals := make([]string, 0, 2)
		if stripeadapter.ValidPriceID(plan.monthly) {
			intervals = append(intervals, "monthly")
		}
		if stripeadapter.ValidPriceID(plan.yearly) {
			intervals = append(intervals, "yearly")
		}
		if len(intervals) > 0 {
			offers.Subscriptions = append(offers.Subscriptions, subscriptionOffer{
				Plan:      plan.name,
				Intervals: intervals,
			})
		}
	}
	return offers
}

func mountBillingUnavailable(publicGroup, userGroup *gin.RouterGroup) {
	publicGroup.POST("/stripe/webhook", billingUnavailable)
	userGroup.POST("/stripe/checkout/session", billingUnavailable)
	userGroup.POST("/stripe/subscription/preview", billingUnavailable)
	userGroup.POST("/stripe/subscription/change", billingUnavailable)
	userGroup.POST("/stripe/subscription/cancel", billingUnavailable)
	userGroup.POST("/stripe/subscription/reactivate", billingUnavailable)
	userGroup.POST("/stripe/portal/session", billingUnavailable)
	userGroup.GET("/stripe/subscription", billingUnavailable)
	userGroup.GET("/stripe/invoices", billingUnavailable)
}

func billingUnavailable(c *gin.Context) {
	httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "billing is not configured", gin.H{
		"capability": "billing",
		"enabled":    false,
	})
}

func (m *Modules) verifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request verifyCodeRequest
		if !authhttp.BindJSONBody(c, &request, false) {
			return
		}
		ctx := WithReferralCode(c.Request.Context(), request.ReferralCode)
		result, err := m.Auth.Login.VerifyCode(ctx, request.Email, request.Code)
		if err != nil {
			respondAuthError(c, err)
			return
		}
		httpresp.OK(c, verifyResultJSON(result))
	}
}

func (m *Modules) exchangeToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request exchangeTokenRequest
		if !authhttp.BindJSONBody(c, &request, false) {
			return
		}
		ctx := WithReferralCode(c.Request.Context(), request.ReferralCode)
		result, err := m.Auth.OAuth.ExchangeToken(ctx, request.Code, request.OAuthVerifier)
		if err != nil {
			respondAuthError(c, err)
			return
		}
		httpresp.OK(c, verifyResultJSON(result))
	}
}

func verifyResultJSON(result *authapp.VerifyResult) gin.H {
	return gin.H{
		"token":      result.Token.Value,
		"expires_at": result.Token.ExpiresAt,
		"user": gin.H{
			"id":       result.Identity.UserID,
			"email":    result.Identity.Email,
			"username": result.Identity.Username,
			"avatar":   result.Identity.AvatarURL,
			"role":     result.Identity.Role,
			"is_new":   result.Identity.IsNew,
		},
	}
}

func respondAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, authdomain.ErrInvalidEmail),
		errors.Is(err, authdomain.ErrInvalidCode),
		errors.Is(err, authdomain.ErrInvalidExchange),
		errors.Is(err, authdomain.ErrInvalidState),
		errors.Is(err, authdomain.ErrCodeMaxAttempts):
		httpresp.BadRequest(c, err.Error())
	case errors.Is(err, authdomain.ErrCodeRateLimited):
		httpresp.TooManyRequests(c, err.Error())
	case errors.Is(err, authdomain.ErrInvalidToken),
		errors.Is(err, authdomain.ErrInvalidWSTicket),
		errors.Is(err, authdomain.ErrUnauthorized):
		httpresp.Unauthorized(c, err.Error())
	case errors.Is(err, authdomain.ErrUserNotFound):
		httpresp.NotFound(c, err.Error())
	case errors.Is(err, authdomain.ErrProviderUnavailable),
		errors.Is(err, authdomain.ErrOAuthFlowUnavailable):
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error(), nil)
	default:
		slog.ErrorContext(c.Request.Context(), "auth: internal error", "error", err)
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal authentication error", nil)
	}
}

func (m *Modules) RequireUser() gin.HandlerFunc {
	if m == nil || m.Auth == nil {
		return authUnavailable
	}
	return authhttp.MiddlewareForUserGroup(m.Auth.Session)
}

func (m *Modules) RequireAdmin() gin.HandlerFunc {
	if m == nil || m.Auth == nil {
		return authUnavailable
	}
	return authhttp.RequireAdmin(m.Auth.Session)
}

func authUnavailable(c *gin.Context) {
	httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "authentication disabled", nil)
	c.Abort()
}

func userIDFromGin(c *gin.Context) (string, bool) {
	identity := authhttp.Authenticated(c)
	if identity == nil || strings.TrimSpace(identity.UserID) == "" {
		return "", false
	}
	return strings.TrimSpace(identity.UserID), true
}
