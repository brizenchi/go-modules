package operations

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	refdomain "github.com/brizenchi/go-modules/modules/referral/domain"
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
		httpresp.InternalError(c, "operation failed")
		return true
	}
	return false
}

func (m *Module) overview(c *gin.Context) {
	db := m.db(c)
	if db == nil {
		return
	}
	counts := map[string]int64{}
	queries := map[string]*gorm.DB{
		"users":                db.Model(&user.User{}),
		"subscriptions":        db.Model(&billingdomain.BillingSubscription{}),
		"active_subscriptions": db.Model(&billingdomain.BillingSubscription{}).Where("status IN ?", []string{"active", "trialing", "canceling"}),
		"billing_events":       db.Model(&billingdomain.BillingEvent{}),
		"referrals":            db.Table("referrals"),
		"pending_referrals":    db.Table("referrals").Where("status = ? AND (expires_at IS NULL OR expires_at > ?)", "pending", time.Now().UTC()),
		"activated_referrals":  db.Table("referrals").Where("status = ?", "activated"),
	}
	for key, q := range queries {
		if (strings.Contains(key, "subscription") || key == "billing_events") && m.deps.Modules != nil && m.deps.Modules.Billing == nil {
			counts[key] = 0
			continue
		}
		if strings.Contains(key, "referral") && m.deps.Modules != nil && m.deps.Modules.Referral == nil {
			counts[key] = 0
			continue
		}
		var count int64
		if queryFailed(c, q.Count(&count).Error) {
			return
		}
		counts[key] = count
	}
	httpresp.OK(c, counts)
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
	users := m.deps.Users
	if users == nil {
		users = user.NewRepository(db)
	}
	for i := range items {
		if items[i].CreditsVersion != 1 {
			continue
		}
		summary, err := users.GetCreditSummary(c.Request.Context(), items[i].ID)
		if queryFailed(c, err) {
			return
		}
		items[i].Credits = summary.Balance
	}
	pageResponse(c, p, items, total)
}

type subscriptionView struct {
	ID                     uint       `json:"id"`
	UserID                 string     `json:"user_id"`
	Email                  string     `json:"email"`
	Provider               string     `json:"provider"`
	ProviderSubscriptionID string     `json:"provider_subscription_id"`
	Plan                   string     `json:"plan"`
	Status                 string     `json:"status"`
	BillingInterval        string     `json:"billing_interval"`
	PeriodEnd              *time.Time `json:"period_end"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
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
	items := []subscriptionView{}
	if queryFailed(c, q.Select("s.id,s.user_id,u.email,s.provider,s.provider_subscription_id,s.plan,s.status,s.billing_interval,s.period_end,s.cancel_at_period_end,s.created_at,s.updated_at").Order("s.created_at DESC, s.id DESC").Offset((p.page-1)*p.limit).Limit(p.limit).Scan(&items).Error) {
		return
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
