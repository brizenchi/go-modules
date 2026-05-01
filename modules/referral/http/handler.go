// Package http exposes Gin handlers for the referral module.
//
// Like modules/billing/http and modules/email/* don't, this module's handlers
// don't take a UserIDFunc — instead the host attaches its auth
// middleware to the user group BEFORE calling Mount, and the host
// passes the user-id extractor in via Deps.
package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	"github.com/brizenchi/go-modules/modules/referral/app"
	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/gin-gonic/gin"
)

// UserIDFunc resolves the authenticated user ID from a Gin context.
type UserIDFunc func(c *gin.Context) (string, bool)

// Handler bundles HTTP endpoints for the referral module.
type Handler struct {
	codes     *app.CodeService
	attribute *app.AttributeService
	query     *app.QueryService
	getUserID UserIDFunc
	baseLink  string
}

type Deps struct {
	Codes     *app.CodeService
	Attribute *app.AttributeService
	Query     *app.QueryService
	GetUserID UserIDFunc

	// BaseLink is the URL prefix appended to a code to build a shareable
	// invite link, e.g. "https://app.example.com/invite?ref=".
	BaseLink string
}

func NewHandler(d Deps) *Handler {
	return &Handler{
		codes:     d.Codes,
		attribute: d.Attribute,
		query:     d.Query,
		getUserID: d.GetUserID,
		baseLink:  d.BaseLink,
	}
}

// GetMyCode returns the authenticated user's code (creating it lazily).
func (h *Handler) GetMyCode(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	code, err := h.codes.GetOrCreate(c.Request.Context(), userID)
	if err != nil {
		respondAppError(c, err)
		return
	}
	link := ""
	if h.baseLink != "" {
		link = h.baseLink + code.Value
	}
	httpresp.OK(c, gin.H{
		"code": code.Value,
		"link": link,
	})
}

// ListMyReferrals returns paginated referrals where the user is the referrer.
func (h *Handler) ListMyReferrals(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, total, err := h.query.ListByReferrer(c.Request.Context(), userID, page, limit)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetMyStats returns the authenticated user's referral aggregates.
func (h *Handler) GetMyStats(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	stats, err := h.query.Stats(c.Request.Context(), userID)
	if err != nil {
		respondAppError(c, err)
		return
	}
	httpresp.OK(c, stats)
}

// --- helpers ----------------------------------------------------------

func (h *Handler) userID(c *gin.Context) (string, bool) {
	if h.getUserID == nil {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	id, ok := h.getUserID(c)
	id = strings.TrimSpace(id)
	if !ok || id == "" {
		respondError(c, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return id, true
}

func respondError(c *gin.Context, status int, msg string) {
	switch status {
	case http.StatusBadRequest:
		httpresp.BadRequest(c, msg)
	case http.StatusUnauthorized:
		httpresp.Unauthorized(c, msg)
	case http.StatusNotFound:
		httpresp.NotFound(c, msg)
	default:
		httpresp.Custom(c, status, status, msg, nil)
	}
}

func respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidUser),
		errors.Is(err, domain.ErrInvalidCode),
		errors.Is(err, domain.ErrSelfReferral),
		errors.Is(err, domain.ErrAlreadyAttributed),
		errors.Is(err, domain.ErrAlreadyActivated):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, err.Error())
	}
}
