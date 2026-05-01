package saascore

import (
	"errors"
	"net/http"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authapp "github.com/brizenchi/go-modules/modules/auth/app"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	billinghttp "github.com/brizenchi/go-modules/modules/billing/http"
	referralhttp "github.com/brizenchi/go-modules/modules/referral/http"
	"github.com/gin-gonic/gin"
)

type verifyCodeRequest struct {
	Email        string `json:"email"`
	Code         string `json:"code"`
	ReferralCode string `json:"referral_code"`
}

type exchangeTokenRequest struct {
	Code         string `json:"code"`
	ReferralCode string `json:"referral_code"`
}

func (s *Stack) Mount(publicGroup, userGroup *gin.RouterGroup) {
	publicGroup.POST("/auth/send-code", s.Auth.Handler.SendCode)
	publicGroup.POST("/auth/verify-code", s.verifyCode())
	publicGroup.GET("/auth/:provider/authorize", s.Auth.Handler.StartOAuth)
	publicGroup.GET("/auth/:provider/callback", s.Auth.Handler.OAuthCallback)
	publicGroup.POST("/auth/exchange-token", s.exchangeToken())

	userGroup.POST("/auth/refresh", s.Auth.Handler.Refresh)
	userGroup.POST("/auth/logout", s.Auth.Handler.Logout)
	userGroup.POST("/websocket/ticket", s.Auth.Handler.IssueWSTicket)

	billinghttp.Mount(s.Billing.Handler, publicGroup, userGroup)
	referralhttp.Mount(s.Referral.Handler, userGroup)
}

func (s *Stack) verifyCode() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req verifyCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresp.BadRequest(c, "invalid body")
			return
		}
		ctx := withReferralCode(c.Request.Context(), req.ReferralCode)
		res, err := s.Auth.Login.VerifyCode(ctx, req.Email, req.Code)
		if err != nil {
			respondAuthError(c, err)
			return
		}
		httpresp.OK(c, verifyResultToJSON(res))
	}
}

func (s *Stack) exchangeToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req exchangeTokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpresp.BadRequest(c, "invalid body")
			return
		}
		ctx := withReferralCode(c.Request.Context(), req.ReferralCode)
		res, err := s.Auth.OAuth.ExchangeToken(ctx, req.Code)
		if err != nil {
			respondAuthError(c, err)
			return
		}
		httpresp.OK(c, verifyResultToJSON(res))
	}
}

func verifyResultToJSON(res *authapp.VerifyResult) gin.H {
	return gin.H{
		"token":      res.Token.Value,
		"expires_at": res.Token.ExpiresAt,
		"user": gin.H{
			"id":       res.Identity.UserID,
			"email":    res.Identity.Email,
			"username": res.Identity.Username,
			"avatar":   res.Identity.AvatarURL,
			"role":     res.Identity.Role,
			"is_new":   res.Identity.IsNew,
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

func (s *Stack) RequireUser() gin.HandlerFunc {
	return authhttp.MiddlewareForUserGroup(s.Auth.Session)
}
