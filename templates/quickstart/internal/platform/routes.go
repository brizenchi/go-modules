package platform

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	billinghttp "github.com/brizenchi/go-modules/modules/billing/http"
	referralhttp "github.com/brizenchi/go-modules/modules/referral/http"
	"github.com/gin-gonic/gin"
)

type referralCodeContextKey struct{}

func withReferralCode(ctx context.Context, code string) context.Context {
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
	Code         string `json:"code"`
	ReferralCode string `json:"referral_code"`
}

// Mount exposes only routes for capabilities enabled in this SaaS.
func (m *Modules) Mount(publicGroup, userGroup *gin.RouterGroup) {
	if m == nil {
		return
	}
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
	}
	if m.Referral != nil {
		referralhttp.Mount(m.Referral.Handler, userGroup)
	}
}

func (m *Modules) verifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		var request verifyCodeRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			httpresp.BadRequest(c, "invalid body")
			return
		}
		ctx := withReferralCode(c.Request.Context(), request.ReferralCode)
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
		if err := c.ShouldBindJSON(&request); err != nil {
			httpresp.BadRequest(c, "invalid body")
			return
		}
		ctx := withReferralCode(c.Request.Context(), request.ReferralCode)
		result, err := m.Auth.OAuth.ExchangeToken(ctx, request.Code)
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
	case errors.Is(err, authdomain.ErrProviderUnavailable):
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error(), nil)
	default:
		httpresp.Custom(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error(), nil)
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
