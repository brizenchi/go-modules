package operations

import (
	"context"
	"strconv"
	"time"

	"github.com/brizenchi/go-modules/foundation/httpresp"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/quickstart-template/internal/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dailyMetrics struct {
	Date           string `json:"date"`
	Registrations  int64  `json:"registrations"`
	Referrals      int64  `json:"referrals"`
	Payments       int64  `json:"payments"`
	NewPayingUsers int64  `json:"new_paying_users"`
}

func (m *Module) overview(c *gin.Context) {
	db := m.db(c)
	if db == nil {
		return
	}
	timezone := c.DefaultQuery("timezone", "Asia/Shanghai")
	location := time.FixedZone("UTC+8", 8*60*60)
	switch timezone {
	case "Asia/Shanghai":
	case "UTC":
		location = time.UTC
	default:
		httpresp.BadRequest(c, "unsupported reporting timezone")
		return
	}
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || (days != 7 && days != 30) {
		httpresp.BadRequest(c, "days must be 7 or 30")
		return
	}
	mode := c.DefaultQuery("mode", "live")
	if mode != "live" && mode != "test" {
		httpresp.BadRequest(c, "mode must be live or test")
		return
	}
	billingEnabled := m.deps.Modules == nil || m.deps.Modules.Billing != nil
	referralEnabled := m.deps.Modules == nil || m.deps.Modules.Referral != nil
	result, err := loadOverview(c.Request.Context(), db, time.Now().UTC(), location, days, mode, billingEnabled, referralEnabled)
	if queryFailed(c, err) {
		return
	}
	result["timezone"] = timezone
	httpresp.OK(c, result)
}

func loadOverview(ctx context.Context, db *gorm.DB, now time.Time, location *time.Location, days int, mode string, billingEnabled, referralEnabled bool) (gin.H, error) {
	db = db.WithContext(ctx)
	local := now.In(location)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	start := today.AddDate(0, 0, -days+1)
	series := make([]dailyMetrics, days)
	indexes := map[string]int{}
	for i := range series {
		series[i].Date = start.AddDate(0, 0, i).Format("2006-01-02")
		indexes[series[i].Date] = i
	}
	result := gin.H{"subscriptions": int64(0), "active_subscriptions": int64(0), "billing_events": int64(0), "referrals": int64(0), "pending_referrals": int64(0), "activated_referrals": int64(0)}
	queries := map[string]*gorm.DB{"users": db.Model(&user.User{})}
	if billingEnabled {
		queries["subscriptions"] = db.Model(&billingdomain.BillingSubscription{})
		queries["active_subscriptions"] = db.Model(&billingdomain.BillingSubscription{}).Where("status IN ?", []string{"active", "trialing", "canceling"})
		queries["billing_events"] = db.Model(&billingdomain.BillingEvent{})
	}
	if referralEnabled {
		queries["referrals"] = db.Table("referrals")
		queries["pending_referrals"] = db.Table("referrals").Where("status = ? AND (expires_at IS NULL OR expires_at > ?)", "pending", now)
		queries["activated_referrals"] = db.Table("referrals").Where("status = ?", "activated")
	}
	for key, q := range queries {
		var count int64
		if err := q.Count(&count).Error; err != nil {
			return nil, err
		}
		result[key] = count
	}
	// Stream bounded rows; calendar bucketing is identical in SQLite and PostgreSQL.
	if err := countDaily(ctx, db, "users", start.UTC(), now, location, indexes, func(i int) { series[i].Registrations++ }); err != nil {
		return nil, err
	}
	if referralEnabled {
		if err := countDaily(ctx, db, "referrals", start.UTC(), now, location, indexes, func(i int) { series[i].Referrals++ }); err != nil {
			return nil, err
		}
	}
	firstPaid := map[string]time.Time{}
	if billingEnabled {
		orders, err := loadOrderSummaries(ctx, db)
		if err != nil {
			return nil, err
		}
		for _, order := range orders {
			if order.Livemode == nil || *order.Livemode != (mode == "live") || order.Status != "paid" || order.RecordType == "payment_event" || order.Amount == nil || *order.Amount <= 0 || order.paidAt.After(now) {
				continue
			}
			if i, ok := indexes[order.paidAt.In(location).Format("2006-01-02")]; ok {
				series[i].Payments++
			}
			if order.UserID != "" {
				if previous, ok := firstPaid[order.UserID]; !ok || order.paidAt.Before(previous) {
					firstPaid[order.UserID] = order.paidAt
				}
			}
		}
	}
	// Restrict paying users to current accounts; stale/orphan provider metadata is not a customer.
	ids := make([]string, 0, len(firstPaid))
	for id := range firstPaid {
		ids = append(ids, id)
	}
	var paidUsers int64
	for offset := 0; offset < len(ids); offset += 500 {
		end := min(offset+500, len(ids))
		var known []string
		if err := db.Model(&user.User{}).Where("id IN ?", ids[offset:end]).Order("id ASC").Limit(500).Pluck("id", &known).Error; err != nil {
			return nil, err
		}
		paidUsers += int64(len(known))
		for _, id := range known {
			if i, ok := indexes[firstPaid[id].In(location).Format("2006-01-02")]; ok {
				series[i].NewPayingUsers++
			}
		}
	}
	result["today_registrations"] = series[days-1].Registrations
	result["today_referrals"] = series[days-1].Referrals
	result["today_payments"] = series[days-1].Payments
	result["paid_users"] = paidUsers
	result["series"] = series
	result["mode"] = mode
	result["as_of"] = now
	return result, nil
}

func countDaily(ctx context.Context, db *gorm.DB, table string, start, end time.Time, location *time.Location, indexes map[string]int, add func(int)) error {
	type stamp struct {
		ID        string
		CreatedAt time.Time
	}
	// ID is used only as a stable tiebreaker; CAST supports numeric referral IDs and UUID users alike.
	cursor := ""
	for {
		rows := []stamp{}
		dateRange := "created_at >= ? AND created_at <= ?"
		if db.Dialector.Name() == "sqlite" {
			dateRange = "datetime(created_at) >= datetime(?) AND datetime(created_at) <= datetime(?)"
		}
		q := db.WithContext(ctx).Table(table).Select("CAST(id AS TEXT) AS id, created_at").Where(dateRange, start, end).Where("CAST(id AS TEXT) > ?", cursor).Order("CAST(id AS TEXT) ASC").Limit(500)
		if err := q.Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			if row.CreatedAt.Before(start) || row.CreatedAt.After(end) {
				continue
			}
			if i, ok := indexes[row.CreatedAt.In(location).Format("2006-01-02")]; ok {
				add(i)
			}
		}
		cursor = rows[len(rows)-1].ID
	}
}
