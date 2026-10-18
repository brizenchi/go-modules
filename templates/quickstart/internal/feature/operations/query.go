package operations

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	refdomain "github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/feature/membership"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type pageQuery struct {
	page, limit int
	query       string
}

func pagination(c *gin.Context) (pageQuery, bool) {
	p := pageQuery{page: 1, limit: 20, query: strings.TrimSpace(c.Query("query"))}
	for key, target := range map[string]*int{"page": &p.page, "limit": &p.limit} {
		if raw := c.Query(key); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 {
				httpresp.BadRequest(c, "page and limit must be positive integers")
				return p, false
			}
			*target = value
		}
	}
	if p.limit > 100 || p.page > 1000000 || len(p.query) > 200 {
		httpresp.BadRequest(c, "pagination or search exceeds allowed limit")
		return p, false
	}
	return p, true
}
func pageResponse(c *gin.Context, p pageQuery, items any, total int64) {
	httpresp.OK(c, gin.H{"items": items, "total": total, "page": p.page, "limit": p.limit})
}
func search(db *gorm.DB, term string, columns ...string) *gorm.DB {
	if term == "" {
		return db
	}
	term = strings.ToLower(term)
	term = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(term)
	parts := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, fmt.Sprintf("LOWER(COALESCE(%s, '')) LIKE ? ESCAPE '!'", column))
		args = append(args, "%"+term+"%")
	}
	return db.Where("("+strings.Join(parts, " OR ")+")", args...)
}
func (m *Module) db(c *gin.Context) *gorm.DB {
	if m.deps.DB == nil {
		httpresp.Custom(c, 503, 503, "database unavailable", nil)
		return nil
	}
	return m.deps.DB.WithContext(c.Request.Context())
}
func queryFailed(c *gin.Context, err error) bool {
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "operator query failed", "error", err)
		httpresp.InternalError(c, "operation failed")
		return true
	}
	return false
}

func (m *Module) users(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	q := search(db.Model(&user.User{}), p.query, "email", "username", "id")
	var total int64
	if queryFailed(c, q.Count(&total).Error) {
		return
	}
	items := []user.User{}
	if queryFailed(c, q.Order("created_at DESC, id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Find(&items).Error) {
		return
	}
	credits := m.deps.Credits
	if credits == nil {
		credits = credit.New(user.NewRepository(db))
	}
	for i := range items {
		if items[i].CreditsVersion != 1 {
			continue
		}
		summary, err := credits.GetCreditSummary(c.Request.Context(), items[i].ID)
		if queryFailed(c, err) {
			return
		}
		items[i].Credits = summary.Balance
	}
	ids := make([]string, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	states, err := membership.LoadStates(c.Request.Context(), db, ids, time.Now().UTC(), m.deps.Modules == nil || m.deps.Modules.Billing != nil)
	if queryFailed(c, err) {
		return
	}
	type userView struct {
		user.User
		Plan         string            `json:"plan"`
		Membership   membership.View   `json:"membership"`
		Subscription *subscriptionView `json:"subscription"`
	}
	views := make([]userView, 0, len(items))
	for _, u := range items {
		v := states[u.ID].Membership
		views = append(views, userView{User: u, Plan: v.Plan, Membership: v, Subscription: operatorSubscription(states[u.ID].Subscription, time.Now().UTC())})
	}
	pageResponse(c, p, views, total)
}

type subscriptionView struct {
	ID                     uint                           `json:"id"`
	UserID                 string                         `json:"user_id"`
	Email                  string                         `json:"email"`
	Provider               string                         `json:"provider"`
	ProviderSubscriptionID string                         `json:"provider_subscription_id"`
	Plan                   string                         `json:"plan"`
	Status                 string                         `json:"status"`
	BillingInterval        string                         `json:"billing_interval"`
	PeriodEnd              *time.Time                     `json:"period_end"`
	CancelAtPeriodEnd      bool                           `json:"cancel_at_period_end"`
	CancelEffectiveAt      *time.Time                     `json:"cancel_effective_at,omitempty"`
	ScheduledChange        *billingdomain.ScheduledChange `json:"scheduled_change,omitempty"`
	PaymentPending         bool                           `json:"payment_pending"`
	CreatedAt              time.Time                      `json:"created_at"`
	UpdatedAt              time.Time                      `json:"updated_at"`
}

func (m *Module) subscriptions(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	if m.deps.Modules != nil && m.deps.Modules.Billing == nil {
		pageResponse(c, p, []subscriptionView{}, 0)
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	q := search(db.Table("billing_subscriptions AS s").Joins("LEFT JOIN users u ON u.id = s.user_id"), p.query, "u.email", "s.user_id", "s.provider_subscription_id")
	var total int64
	if queryFailed(c, q.Count(&total).Error) {
		return
	}
	rows := []struct {
		billingdomain.BillingSubscription
		Email string
	}{}
	if queryFailed(c, q.Select("s.*, u.email").Order("s.created_at DESC, s.id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Scan(&rows).Error) {
		return
	}
	items := make([]subscriptionView, 0, len(rows))
	for _, row := range rows {
		view := operatorSubscription(&row.BillingSubscription, time.Now().UTC())
		view.Email = row.Email
		items = append(items, *view)
	}
	pageResponse(c, p, items, total)
}

type referralView struct {
	ID            uint64           `json:"id"`
	Code          string           `json:"code"`
	ReferrerID    string           `json:"referrer_id"`
	RefereeID     string           `json:"referee_id"`
	ReferrerEmail string           `json:"referrer_email"`
	RefereeEmail  string           `json:"referee_email"`
	Status        refdomain.Status `json:"status"`
	ActivatedAt   *time.Time       `json:"activated_at"`
	ExpiresAt     *time.Time       `json:"expires_at"`
	RewardCredits int              `json:"reward_credits"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

func (m *Module) referrals(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	if m.deps.Modules != nil && m.deps.Modules.Referral == nil {
		pageResponse(c, p, []referralView{}, 0)
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != "pending" && status != "activated" && status != "expired" {
		httpresp.BadRequest(c, "invalid referral status")
		return
	}
	now := time.Now().UTC()
	effectiveStatus := "CASE WHEN r.status = 'pending' AND r.expires_at IS NOT NULL AND r.expires_at <= ? THEN 'expired' ELSE r.status END"
	q := search(db.Table("referrals AS r").Joins("LEFT JOIN users a ON a.id = r.referrer_id").Joins("LEFT JOIN users b ON b.id = r.referee_id"), p.query, "r.code", "r.referrer_id", "r.referee_id", "a.email", "b.email")
	if status != "" {
		q = q.Where("("+effectiveStatus+") = ?", now, status)
	}
	if uid := strings.TrimSpace(c.Query("user_id")); uid != "" {
		q = q.Where("r.referrer_id = ?", uid)
	}
	var total int64
	if queryFailed(c, q.Count(&total).Error) {
		return
	}
	items := []referralView{}
	if queryFailed(c, q.Select("r.id,r.code,r.referrer_id,r.referee_id,a.email AS referrer_email,b.email AS referee_email,"+effectiveStatus+" AS status,r.activated_at,r.expires_at,r.reward_credits,r.created_at,r.updated_at", now).Order("r.created_at DESC,r.id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Scan(&items).Error) {
		return
	}
	pageResponse(c, p, items, total)
}

func (m *Module) audit(c *gin.Context) {
	p, ok := pagination(c)
	if !ok {
		return
	}
	db := m.db(c)
	if db == nil {
		return
	}
	q := search(db.Model(&AuditEvent{}), p.query, "actor_id", "action", "target_id", "reason")
	if status := c.Query("status"); status != "" {
		if status != "succeeded" && status != "failed" && status != "pending" {
			httpresp.BadRequest(c, "invalid audit status")
			return
		}
		q = q.Where("status = ?", status)
	}
	var total int64
	if queryFailed(c, q.Count(&total).Error) {
		return
	}
	items := []AuditEvent{}
	if queryFailed(c, q.Order("created_at DESC,id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Find(&items).Error) {
		return
	}
	pageResponse(c, p, items, total)
}
