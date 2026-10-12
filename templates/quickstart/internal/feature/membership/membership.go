// Package membership owns this product's entitlement policy. Customize Evaluate
// and protect paid routes with Require; never authorize from browser state.
package membership

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type View struct {
	Plan       string     `json:"plan"`
	Active     bool       `json:"active"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	Reason     string     `json:"reason"`
}

// Evaluate defaults to no payment-failure grace period. Canceling retains paid
// access until the earlier of the paid period end and explicit cancellation.
// Missing dates, unknown states and expired periods fail closed.
func Evaluate(row domain.BillingSubscription, now time.Time) View {
	v := View{Plan: "free", Reason: "membership_required"}
	if row.Status != "active" && row.Status != "trialing" && row.Status != "canceling" {
		return v
	}
	if row.Plan == "lifetime" && row.Status == "active" {
		return View{Plan: "lifetime", Active: true, Reason: "active"}
	}
	if row.Plan != "starter" && row.Plan != "pro" && row.Plan != "premium" {
		return v
	}
	end := row.PeriodEnd
	if row.CancelEffectiveAt != nil && (end == nil || row.CancelEffectiveAt.Before(*end)) {
		end = row.CancelEffectiveAt
	}
	v.ValidUntil = end
	if end == nil || !now.Before(*end) {
		v.Reason = "membership_expired"
		return v
	}
	v.Plan = row.Plan
	v.Active = true
	v.Reason = "active"
	return v
}

func Load(ctx context.Context, db *gorm.DB, uid string) (View, error) {
	var row domain.BillingSubscription
	if db == nil {
		return View{Plan: "free", Reason: "membership_required"}, nil
	}
	err := db.WithContext(ctx).Where("user_id = ? AND provider = ?", uid, "stripe").Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
	}
	if err != nil {
		return View{}, err
	}
	return Evaluate(row, time.Now().UTC()), nil
}

func read(c *gin.Context, db *gorm.DB) (View, bool) {
	id := authhttp.Authenticated(c)
	if id == nil || id.UserID == "" {
		httpresp.Unauthorized(c, "unauthorized")
		c.Abort()
		return View{}, false
	}
	v, err := Load(c.Request.Context(), db, id.UserID)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "membership lookup failed", "error", err)
		httpresp.InternalError(c, "membership unavailable")
		c.Abort()
		return View{}, false
	}
	return v, true
}

// Require enforces membership and minimum plan on each request. Lifetime grants
// the highest tier. Use after authentication: g.User.GET("/paid", Require(db,"pro"), handler).
func Require(db *gorm.DB, minimum string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := read(c, db)
		if !ok {
			return
		}
		ranks := map[string]int{"starter": 1, "pro": 2, "premium": 3, "lifetime": 3}
		rank, known := ranks[minimum]
		if !known || !v.Active || ranks[v.Plan] < rank {
			httpresp.Custom(c, 403, 403, "membership required", gin.H{"reason": "membership_required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func Register(deps hostapi.Deps, g hostapi.Groups) {
	if g.User == nil || deps.Modules == nil || deps.Modules.Billing == nil {
		return
	}
	g.User.GET("/account/membership", func(c *gin.Context) {
		if v, ok := read(c, deps.DB); ok {
			httpresp.OK(c, v)
		}
	})
}
