package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/gin-gonic/gin"
)

// Handler bundles Gin endpoints for the auth module.
//
// Each provider gets its own /auth/{provider}/{authorize,callback} routes;
// the email-code flow uses /auth/send-code + /auth/verify-code.
//
// JSON responses use foundation/httpresp's envelope:
// { "code": <int>, "msg": "<string>", "data": <any|null> }.
//
// The one exception is OAuthCallback when FrontendURL is configured:
// that path issues an HTTP redirect instead of JSON.
type Handler struct {
	login     *app.LoginService
	oauth     *app.OAuthService
	session   *app.SessionService
	frontendU string // optional frontend URL for OAuth callback redirects
}

// Deps gathers handler dependencies.
type Deps struct {
	Login       *app.LoginService
	OAuth       *app.OAuthService
	Session     *app.SessionService
	FrontendURL string // when set, OAuthCallback redirects browser to this URL with #code=...
}

func NewHandler(d Deps) *Handler {
	return &Handler{login: d.Login, oauth: d.OAuth, session: d.Session, frontendU: d.FrontendURL}
}

// --- email-code flow ---------------------------------------------------

type sendCodeReq struct {
	Email string `json:"email"`
}

func (h *Handler) SendCode(c *gin.Context) {
	var req sendCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	res, err := h.login.SendCode(c.Request.Context(), req.Email)
	if err != nil {
		respondAppError(c, err)
		return
	}
	body := gin.H{
		"email":      res.Email,
		"expires_at": res.ExpiresAt,
	}
	if res.DebugCode != "" {
		body["debug_code"] = res.DebugCode
	}
	httpresp.OK(c, body)
}

type verifyCodeReq struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *Handler) VerifyCode(c *gin.Context) {
	var req verifyCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	res, err := h.login.VerifyCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, verifyResultToJSON(res))
}

// --- OAuth flow --------------------------------------------------------

// StartOAuth returns the authorize URL. Path: /auth/:provider/authorize.
// Frontends can either GET this and redirect, or follow the URL directly.
func (h *Handler) StartOAuth(c *gin.Context) {
	provider := c.Param("provider")
	url, err := h.oauth.StartOAuth(c.Request.Context(), provider)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{"redirect_url": url})
}

// OAuthCallback handles the provider redirect.
// On success: if FrontendURL is configured, redirects browser to
// FrontendURL with ?code=<exchange_code> appended; otherwise returns JSON.
func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	res, err := h.oauth.OAuthCallback(c.Request.Context(), provider, c.Request.URL.Query())
	if err != nil {
		respondAppError(c, err)
		return
	}
	if h.frontendU != "" {
		sep := "?"
		if strings.Contains(h.frontendU, "?") {
			sep = "&"
		}
		c.Redirect(http.StatusFound, h.frontendU+sep+"code="+res.ExchangeCode)
		return
	}
	httpresp.OK(c, gin.H{
		"exchange_code": res.ExchangeCode,
		"is_new":        res.Identity.IsNew,
	})
}

type exchangeReq struct {
	Code string `json:"code"`
}

// ExchangeToken consumes the exchange code → returns a session token.
func (h *Handler) ExchangeToken(c *gin.Context) {
	var req exchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid body")
		return
	}
	res, err := h.oauth.ExchangeToken(c.Request.Context(), req.Code)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, verifyResultToJSON(res))
}

// --- session ----------------------------------------------------------

func (h *Handler) Refresh(c *gin.Context) {
	id := Authenticated(c)
	if id == nil {
		respondError(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	res, err := h.session.Refresh(c.Request.Context(), id.UserID)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, verifyResultToJSON(res))
}

func (h *Handler) Logout(c *gin.Context) {
	// Stateless JWT: nothing to do server-side. Clients drop the token.
	httpresp.OK(c, gin.H{"ok": true})
}

type wsTicketReq struct {
	Scope map[string]string `json:"scope"`
}

func (h *Handler) IssueWSTicket(c *gin.Context) {
	id := Authenticated(c)
	if id == nil {
		respondError(c, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var req wsTicketReq
	_ = c.ShouldBindJSON(&req)
	ticket, err := h.session.IssueWSTicket(c.Request.Context(), id.UserID, req.Scope)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{
		"ticket":     ticket.Value,
		"expires_at": ticket.ExpiresAt,
	})
}

// --- helpers ----------------------------------------------------------

func verifyResultToJSON(res *app.VerifyResult) gin.H {
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

func respondError(c *gin.Context, status int, msg string) {
	switch status {
	case http.StatusBadRequest:
		httpresp.BadRequest(c, msg)
	case http.StatusUnauthorized:
		httpresp.Unauthorized(c, msg)
	case http.StatusNotFound:
		httpresp.NotFound(c, msg)
	case http.StatusTooManyRequests:
		httpresp.TooManyRequests(c, msg)
	case http.StatusServiceUnavailable:
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, msg, nil)
	default:
		httpresp.Custom(c, status, status, msg, nil)
	}
}

func respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrInvalidExchange),
		errors.Is(err, domain.ErrInvalidState),
		errors.Is(err, domain.ErrCodeMaxAttempts):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrCodeRateLimited):
		respondError(c, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrInvalidWSTicket),
		errors.Is(err, domain.ErrUnauthorized):
		respondError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrProviderUnavailable):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, err.Error())
	}
}
