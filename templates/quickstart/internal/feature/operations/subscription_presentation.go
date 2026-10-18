package operations

import (
	"encoding/json"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
)

// operatorSubscription exposes only presentation fields from the persisted
// provider snapshot. Reading user lists never calls Stripe or changes access.
func operatorSubscription(row *domain.BillingSubscription, now time.Time) *subscriptionView {
	if row == nil {
		return nil
	}
	v := &subscriptionView{
		ID: row.ID, UserID: row.UserID, Provider: row.Provider,
		ProviderSubscriptionID: row.ProviderSubscriptionID,
		Plan:                   row.Plan, Status: row.Status, BillingInterval: row.BillingInterval,
		PeriodEnd: row.PeriodEnd, CancelAtPeriodEnd: row.CancelAtPeriodEnd,
		CancelEffectiveAt: row.CancelEffectiveAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.Status != "active" && row.Status != "trialing" && row.Status != "past_due" && row.Status != "canceling" {
		return v
	}
	var snapshot domain.SubscriptionSnapshot
	// Legacy rows may not contain a normalized snapshot. Preserve their known
	// current state without fabricating an upcoming plan.
	if json.Unmarshal(row.RawSnapshotJSON, &snapshot) != nil {
		return v
	}
	v.PaymentPending = snapshot.PendingPayment != nil
	schedule := snapshot.ScheduledChange
	if v.PaymentPending || v.CancelAtPeriodEnd || v.CancelEffectiveAt != nil || row.Plan == "lifetime" || schedule == nil {
		return v
	}
	if schedule.Plan != domain.PlanStarter && schedule.Plan != domain.PlanPro && schedule.Plan != domain.PlanPremium {
		return v
	}
	if schedule.Interval != domain.IntervalMonthly && schedule.Interval != domain.IntervalYearly {
		return v
	}
	if schedule.EffectiveAt == nil || !schedule.EffectiveAt.After(now) {
		return v
	}
	stamp := schedule.EffectiveAt.UTC()
	v.ScheduledChange = &domain.ScheduledChange{Plan: schedule.Plan, Interval: schedule.Interval, EffectiveAt: &stamp}
	return v
}
