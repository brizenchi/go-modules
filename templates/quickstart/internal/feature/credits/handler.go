package credits

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func identity(c *gin.Context, admin bool) *authdomain.Identity {
	id := authhttp.Authenticated(c)
	if id == nil || strings.TrimSpace(id.UserID) == "" {
		httpresp.Unauthorized(c, "authentication required")
		return nil
	}
	if admin && id.Role != authdomain.RoleAdmin {
		httpresp.Forbidden(c, "admin role required")
		return nil
	}
	return id
}

// Reject unknown fields (including actor_id and role) instead of silently
// accepting a spoofed operator identity. Payload size is bounded here too.
func decode(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpresp.BadRequest(c, "invalid request body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpresp.BadRequest(c, "invalid request body")
		return false
	}
	return true
}

func pagination(c *gin.Context) (int, int, bool) {
	page, errPage := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, errLimit := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if errPage != nil || errLimit != nil || page < 1 || page > 1_000_000 || limit < 1 || limit > 100 {
		httpresp.BadRequest(c, "invalid pagination")
		return 0, 0, false
	}
	return page, limit, true
}

func failure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		httpresp.NotFound(c, "not found")
	case errors.Is(err, credit.ErrInvalidOperation):
		httpresp.BadRequest(c, err.Error())
	case errors.Is(err, credit.ErrInsufficientCredits), errors.Is(err, credit.ErrConflict), errors.Is(err, credit.ErrAlreadyRefunded), errors.Is(err, ErrExportPriceChanged):
		httpresp.Conflict(c, err.Error())
	case errors.Is(err, credit.ErrMigrationRequired):
		httpresp.Custom(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, err.Error(), nil)
	default:
		httpresp.InternalError(c, "credit operation failed")
	}
}

func (m *Module) summary(c *gin.Context) {
	id := identity(c, false)
	if id == nil {
		return
	}
	result, err := m.service.GetCreditSummary(c.Request.Context(), id.UserID)
	if err != nil {
		failure(c, err)
		return
	}
	httpresp.OK(c, result)
}
func (m *Module) transactions(c *gin.Context) {
	id := identity(c, false)
	if id == nil {
		return
	}
	m.list(c, id.UserID, false)
}
func (m *Module) adminTransactions(c *gin.Context) {
	if identity(c, true) == nil {
		return
	}
	m.list(c, strings.TrimSpace(c.Query("user_id")), true)
}
func (m *Module) list(c *gin.Context, userID string, admin bool) {
	page, limit, ok := pagination(c)
	if !ok {
		return
	}
	list := m.service.ListCreditTransactions
	if admin {
		list = m.service.ListAdminTransactions
	}
	result, err := list(c.Request.Context(), userID, page, limit)
	if err != nil {
		failure(c, err)
		return
	}
	httpresp.OK(c, result)
}

type grantRequest struct {
	UserID         string     `json:"user_id"`
	Amount         int64      `json:"amount"`
	ExpiresAt      *time.Time `json:"expires_at"`
	Reason         string     `json:"reason"`
	IdempotencyKey string     `json:"idempotency_key"`
}
type refundRequest struct {
	TransactionID  uint64 `json:"transaction_id"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

func validRequestKey(value string) bool {
	if value != strings.TrimSpace(value) || len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, ch := range value {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}

func (m *Module) grant(c *gin.Context) {
	id := identity(c, true)
	if id == nil {
		return
	}
	var req grantRequest
	if !decode(c, &req) {
		return
	}
	if !validRequestKey(req.IdempotencyKey) || strings.TrimSpace(req.Reason) == "" {
		httpresp.BadRequest(c, "reason and idempotency_key (8–128 characters) are required")
		return
	}
	entry, err := m.service.GrantCreditsWithExpiry(c.Request.Context(), credit.Grant{UserID: req.UserID, Amount: req.Amount, ExpiresAt: req.ExpiresAt, Reason: req.Reason, ActorID: id.UserID, Source: "admin_grant", SourceID: id.UserID + ":" + req.IdempotencyKey})
	if err != nil {
		failure(c, err)
		return
	}
	m.mutationResult(c, entry)
}
func (m *Module) refund(c *gin.Context) {
	id := identity(c, true)
	if id == nil {
		return
	}
	var req refundRequest
	if !decode(c, &req) {
		return
	}
	if !validRequestKey(req.IdempotencyKey) || strings.TrimSpace(req.Reason) == "" {
		httpresp.BadRequest(c, "reason and idempotency_key (8–128 characters) are required")
		return
	}
	entry, err := m.service.RefundByAdmin(c.Request.Context(), req.TransactionID, id.UserID, req.IdempotencyKey, req.Reason)
	if err != nil {
		failure(c, err)
		return
	}
	m.mutationResult(c, entry)
}
func (m *Module) mutationResult(c *gin.Context, entry *credit.Transaction) {
	summary, err := m.service.GetCreditSummary(c.Request.Context(), entry.UserID)
	if err != nil {
		failure(c, err)
		return
	}
	httpresp.OK(c, gin.H{"transaction": entry, "balance": summary.Balance})
}
