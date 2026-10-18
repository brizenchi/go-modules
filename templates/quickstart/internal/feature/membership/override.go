package membership

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/gorm"
)

// Override is a site-owned access decision, independent of provider billing.
// Expiry reverts to provider-derived access; it never alters recurring charges.
type Override struct {
	UserID    string     `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Plan      string     `gorm:"type:varchar(20);not null" json:"plan"`
	ExpiresAt *time.Time `json:"expires_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Override) TableName() string { return "membership_overrides" }
func Models() []any                { return []any{&Override{}} }

type State struct {
	Membership View      `json:"membership"`
	Billing    View      `json:"billing"`
	Override   *Override `json:"override"`
	// Retained for administrative presentation; never serialize the raw snapshot.
	Subscription *domain.BillingSubscription `json:"-"`
}

func Resolve(billing View, override *Override, now time.Time) View {
	if override == nil || (override.ExpiresAt != nil && !now.Before(*override.ExpiresAt)) {
		return billing
	}
	switch override.Plan {
	case "free":
		return View{Plan: "free", Source: "manual", Reason: "membership_required", ValidUntil: override.ExpiresAt}
	case "lifetime":
		return View{Plan: "lifetime", Active: true, Source: "manual", Reason: "active", ValidUntil: override.ExpiresAt}
	case "starter", "pro", "premium":
		if override.ExpiresAt != nil {
			return View{Plan: override.Plan, Active: true, Source: "manual", Reason: "active", ValidUntil: override.ExpiresAt}
		}
	}
	return billing
}

// LoadStates batches membership reads for one bounded admin page.
func LoadStates(ctx context.Context, db *gorm.DB, ids []string, now time.Time, billingEnabled bool) (map[string]State, error) {
	states := map[string]State{}
	for _, id := range ids {
		v := View{Plan: "free", Reason: "membership_required", Source: "billing"}
		states[id] = State{Membership: v, Billing: v}
	}
	if db == nil || len(ids) == 0 {
		return states, nil
	}
	db = db.WithContext(ctx)
	if billingEnabled {
		rows := []domain.BillingSubscription{}
		if err := db.Where("user_id IN ? AND provider = ?", ids, "stripe").Order("id ASC").Limit(len(ids)).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			v := Evaluate(row, now)
			v.Source = "billing"
			states[row.UserID] = State{Membership: v, Billing: v, Subscription: &row}
		}
	}
	overrides := []Override{}
	if err := db.Where("user_id IN ?", ids).Order("user_id ASC").Limit(len(ids)).Find(&overrides).Error; err != nil {
		return nil, err
	}
	for i := range overrides {
		o := &overrides[i]
		s := states[o.UserID]
		s.Override = o
		s.Membership = Resolve(s.Billing, o, now)
		states[o.UserID] = s
	}
	return states, nil
}
