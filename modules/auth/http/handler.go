package http

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	login             *app.LoginService
	oauth             *app.OAuthService
	session           *app.SessionService
	frontendU         string // optional frontend URL for OAuth callback redirects
	oauthCookieSecure bool
}

// Deps gathers handler dependencies.
type Deps struct {
	Login             *app.LoginService
	OAuth             *app.OAuthService
	Session           *app.SessionService
	FrontendURL       string // when set, OAuthCallback redirects browser to this URL with #code=...
	OAuthCookieSecure bool
}

func NewHandler(d Deps) *Handler {
	return &Handler{
		login: d.Login, oauth: d.OAuth, session: d.Session,
		frontendU: d.FrontendURL, oauthCookieSecure: d.OAuthCookieSecure,
	}
}

// --- email-code flow ---------------------------------------------------

type sendCodeReq struct {
	Email string `json:"email"`
}

const maxAuthJSONBodyBytes int64 = 32 << 10

// BindJSONBody applies the auth API's request-size limit before decoding.
// It writes a consistent error response itself so host wrappers can reuse the
// same limit without duplicating response handling.
func BindJSONBody(c *gin.Context, target any, optional bool) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAuthJSONBodyBytes)
	err := c.ShouldBindJSON(target)
	if err == nil || (optional && errors.Is(err, io.EOF)) {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respondError(c, http.StatusRequestEntityTooLarge, "request body too large")
		return false
	}
	respondError(c, http.StatusBadRequest, "invalid body")
	return false
}

func (h *Handler) SendCode(c *gin.Context) {
	var req sendCodeReq
	if !BindJSONBody(c, &req, false) {
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
	if !BindJSONBody(c, &req, false) {
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
	result, err := h.oauth.StartOAuth(c.Request.Context(), provider, c.Query("challenge"))
	if err != nil {
		respondAppError(c, err)
		return
	}
	h.setOAuthFlowCookie(c, provider, result)
	if c.Query("redirect") == "1" {
		c.Redirect(http.StatusFound, result.AuthorizeURL)
		return
	}
	httpresp.OK(c, gin.H{"redirect_url": result.AuthorizeURL})
}

// OAuthCallback handles the provider redirect.
// On success: if FrontendURL is configured, redirects browser to
// FrontendURL with ?code=<exchange_code> appended; otherwise returns JSON.
func (h *Handler) OAuthCallback(c *gin.Context) {
	provider := c.Param("provider")
	state := c.Query("state")
	cookieName := oauthFlowCookieName(provider, app.OAuthFlowID(state))
	binding := ""
	if cookie, err := c.Request.Cookie(cookieName); err == nil {
		binding = cookie.Value
	}
	h.clearOAuthFlowCookie(c, cookieName)
	res, err := h.oauth.OAuthCallback(c.Request.Context(), provider, c.Request.URL.Query(), binding)
	if err != nil {
		if errorCode, ok := oauthCallbackErrorCode(err); ok {
			if h.frontendU != "" {
				flowID := ""
				if res != nil {
					flowID = res.FlowID
				}
				redirectURL, redirectErr := oauthFrontendErrorRedirect(h.frontendU, errorCode, flowID)
				if redirectErr != nil {
					respondAppError(c, redirectErr)
					return
				}
				c.Redirect(http.StatusFound, redirectURL)
				return
			}
			respondError(c, http.StatusBadRequest, oauthCallbackPublicMessage(errorCode))
			return
		}
		respondAppError(c, err)
		return
	}
	if h.frontendU != "" {
		redirectURL, err := oauthFrontendRedirect(h.frontendU, res.ExchangeCode, res.FlowID)
		if err != nil {
			respondAppError(c, err)
			return
		}
		c.Redirect(http.StatusFound, redirectURL)
		return
	}
	httpresp.OK(c, gin.H{
		"exchange_code": res.ExchangeCode,
		"flow_id":       res.FlowID,
		"is_new":        res.Identity.IsNew,
	})
}

func oauthFrontendRedirect(frontendURL, exchangeCode, flowID string) (string, error) {
	return oauthFrontendRedirectWithQuery(frontendURL, func(query url.Values) {
		query.Del("oauth_error")
		query.Set("code", exchangeCode)
		query.Set("flow", flowID)
	})
}

func oauthFrontendErrorRedirect(frontendURL, errorCode, flowID string) (string, error) {
	return oauthFrontendRedirectWithQuery(frontendURL, func(query url.Values) {
		query.Del("code")
		query.Set("oauth_error", errorCode)
		if flowID == "" {
			query.Del("flow")
		} else {
			query.Set("flow", flowID)
		}
	})
}

func oauthFrontendRedirectWithQuery(frontendURL string, mutate func(url.Values)) (string, error) {
	parsed, err := url.Parse(frontendURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("auth: invalid OAuth frontend redirect URL")
	}
	query := parsed.Query()
	mutate(query)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func oauthCallbackErrorCode(err error) (string, bool) {
	switch {
	case errors.Is(err, domain.ErrOAuthDenied):
		return "access_denied", true
	case errors.Is(err, domain.ErrOAuthCallback):
		return "callback_failed", true
	default:
		return "", false
	}
}

func oauthCallbackPublicMessage(errorCode string) string {
	if errorCode == "access_denied" {
		return domain.ErrOAuthDenied.Error()
	}
	return domain.ErrOAuthCallback.Error()
}

type exchangeReq struct {
	Code          string `json:"code"`
	OAuthVerifier string `json:"oauth_verifier"`
}

// ExchangeToken consumes the exchange code → returns a session token.
func (h *Handler) ExchangeToken(c *gin.Context) {
	var req exchangeReq
	if !BindJSONBody(c, &req, false) {
		return
	}
	res, err := h.oauth.ExchangeToken(c.Request.Context(), req.Code, req.OAuthVerifier)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, verifyResultToJSON(res))
}

func oauthFlowCookieName(provider, flowID string) string {
	// Hash the provider together with the already hashed state so the cookie
	// name is both safe and unique across parallel provider flows.
	sum := sha256.Sum256([]byte(provider + "\x00" + flowID))
	return "oauth_flow_" + hex.EncodeToString(sum[:16])
}

func (h *Handler) setOAuthFlowCookie(c *gin.Context, provider string, result *app.OAuthStartResult) {
	path := oauthCallbackPath(c.Request.URL.Path)
	maxAge := int(time.Until(result.ExpiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthFlowCookieName(provider, result.FlowID),
		Value:    result.BrowserBinding,
		Path:     path,
		Expires:  result.ExpiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.oauthCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearOAuthFlowCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     c.Request.URL.Path,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.oauthCookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func oauthCallbackPath(authorizePath string) string {
	const suffix = "/authorize"
	if strings.HasSuffix(authorizePath, suffix) {
		return strings.TrimSuffix(authorizePath, suffix) + "/callback"
	}
	return authorizePath
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
	if !BindJSONBody(c, &req, true) {
		return
	}
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
		errors.Is(err, domain.ErrOAuthDenied),
		errors.Is(err, domain.ErrOAuthCallback),
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
	case errors.Is(err, domain.ErrProviderUnavailable),
		errors.Is(err, domain.ErrOAuthFlowUnavailable):
		respondError(c, http.StatusServiceUnavailable, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "auth: internal error", "error", err)
		respondError(c, http.StatusInternalServerError, "internal authentication error")
	}
}
