package operations

import (
	"errors"
	"regexp"
	"strings"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/port"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/creditpurchase"
	"github.com/gin-gonic/gin"
)

var checkoutSessionID = regexp.MustCompile(`^cs_[A-Za-z0-9_]{1,252}$`)

func (m *Module) creditCheckoutResult(c *gin.Context)  { m.creditCheckout(c, false) }
func (m *Module) confirmCreditCheckout(c *gin.Context) { m.creditCheckout(c, true) }

func (m *Module) creditCheckout(c *gin.Context, confirm bool) {
	owner := userID(c)
	if owner == "" {
		return
	}
	id := c.Param("id")
	if !checkoutSessionID.MatchString(id) {
		httpresp.BadRequest(c, "invalid checkout session id")
		return
	}
	if confirm {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" || len(key) > 255 || strings.ContainsAny(key, "\r\n") {
			httpresp.BadRequest(c, "Idempotency-Key required (up to 255 characters)")
			return
		}
	}
	db := m.db(c)
	if db == nil {
		return
	}
	service := creditpurchase.New(db, m.deps.Credits)
	var result creditpurchase.Result
	var err error
	if confirm {
		var reader port.CreditsCheckoutReader
		if m.deps.Modules != nil && m.deps.Modules.Billing != nil {
			reader, _ = m.deps.Modules.Billing.Provider.(port.CreditsCheckoutReader)
		}
		result, err = service.Confirm(c.Request.Context(), owner, id, reader)
	} else {
		result, err = service.Status(c.Request.Context(), owner, id)
	}
	if creditCheckoutError(c, err) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	httpresp.OK(c, result)
}

func creditCheckoutError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, creditpurchase.ErrNotFound):
		httpresp.NotFound(c, "checkout not found")
	case errors.Is(err, creditpurchase.ErrInvalid), errors.Is(err, billingdomain.ErrInvalidInput), errors.Is(err, billingdomain.ErrInvalidPriceID):
		httpresp.Custom(c, 400, 400, "invalid credit checkout", gin.H{"reason": "CREDIT_CHECKOUT_INVALID"})
	case errors.Is(err, creditpurchase.ErrConflict):
		httpresp.Custom(c, 409, 409, "credit checkout conflict", gin.H{"reason": "CREDIT_CHECKOUT_CONFLICT"})
	case errors.Is(err, creditpurchase.ErrUnavailable), errors.Is(err, billingdomain.ErrProviderDisabled), errors.Is(err, credit.ErrMigrationRequired):
		httpresp.Custom(c, 503, 503, "credit confirmation unavailable", gin.H{"reason": "CREDIT_CHECKOUT_UNAVAILABLE"})
	default:
		queryFailed(c, err)
	}
	return true
}
