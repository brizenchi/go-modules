package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"gorm.io/gorm"
)

// LegacyBillingSyncOptions controls which legacy users rows are scanned.
type LegacyBillingSyncOptions struct {
	UserID string
	Limit  int
}

// LegacyBillingBackfillReport summarizes a legacy -> billing-table sync.
type LegacyBillingBackfillReport struct {
	Scanned                int      `json:"scanned"`
	CustomerCandidates     int      `json:"customer_candidates"`
	SubscriptionCandidates int      `json:"subscription_candidates"`
	CustomersSynced        int      `json:"customers_synced"`
	SubscriptionsSynced    int      `json:"subscriptions_synced"`
	Warnings               []string `json:"warnings,omitempty"`
}

// LegacyBillingIssue is a single consistency failure between legacy users
// billing columns and the new billing-owned tables.
type LegacyBillingIssue struct {
	UserID  string `json:"user_id"`
	Kind    string `json:"kind"`
	Details string `json:"details"`
}

// LegacyBillingCheckReport summarizes consistency between legacy users
// billing fields and the new billing-owned tables.
type LegacyBillingCheckReport struct {
	Scanned                int                  `json:"scanned"`
	CustomerCandidates     int                  `json:"customer_candidates"`
	SubscriptionCandidates int                  `json:"subscription_candidates"`
	Issues                 []LegacyBillingIssue `json:"issues,omitempty"`
}

func (r LegacyBillingCheckReport) OK() bool { return len(r.Issues) == 0 }

type legacyUserBillingRow struct {
	ID                   string
	Email                string
	Plan                 string
	StripeCustomerID     string
	StripeSubscriptionID string
	StripePriceID        string
	StripeProductID      string
	BillingStatus        string
	BillingPeriodStart   *time.Time
	BillingPeriodEnd     *time.Time
	CancelEffectiveAt    *time.Time
}

// BackfillLegacyStripeState copies billing data out of the legacy users
// table into billing-owned tables.
func BackfillLegacyStripeState(ctx context.Context, db *gorm.DB, opts LegacyBillingSyncOptions) (*LegacyBillingBackfillReport, error) {
	if db == nil {
		return nil, fmt.Errorf("billing: db required")
	}
	rows, err := loadLegacyUserBillingRows(ctx, db, opts)
	if err != nil {
		return nil, err
	}

	customers := NewCustomerStore(db)
	subscriptions := NewSubscriptionRepo(db)
	report := &LegacyBillingBackfillReport{Scanned: len(rows)}

	for _, row := range rows {
		if customerID := strings.TrimSpace(row.StripeCustomerID); customerID != "" {
			report.CustomerCandidates++
			if err := customers.SaveCustomerID(ctx, row.ID, "stripe", customerID); err != nil {
				return nil, err
			}
			report.CustomersSynced++
		}

		snapshot, ok := legacySubscriptionSnapshot(row)
		if !ok {
			continue
		}
		report.SubscriptionCandidates++
		if err := subscriptions.UpsertSnapshot(ctx, row.ID, "stripe", snapshot); err != nil {
			return nil, err
		}
		report.SubscriptionsSynced++
	}

	return report, nil
}

// CheckLegacyStripeState compares the legacy users billing fields against
// the new billing-owned persistence tables.
func CheckLegacyStripeState(ctx context.Context, db *gorm.DB, opts LegacyBillingSyncOptions) (*LegacyBillingCheckReport, error) {
	if db == nil {
		return nil, fmt.Errorf("billing: db required")
	}
	rows, err := loadLegacyUserBillingRows(ctx, db, opts)
	if err != nil {
		return nil, err
	}

	report := &LegacyBillingCheckReport{Scanned: len(rows)}
	for _, row := range rows {
		if customerID := strings.TrimSpace(row.StripeCustomerID); customerID != "" {
			report.CustomerCandidates++
			var customer domain.BillingCustomer
			err := db.WithContext(ctx).
				Where("user_id = ? AND provider = ?", row.ID, "stripe").
				Take(&customer).Error
			switch {
			case err == nil:
				if customer.ProviderCustomerID != customerID {
					report.Issues = append(report.Issues, LegacyBillingIssue{
						UserID:  row.ID,
						Kind:    "customer_mismatch",
						Details: fmt.Sprintf("billing_customers.provider_customer_id=%q want %q", customer.ProviderCustomerID, customerID),
					})
				}
			case errors.Is(err, gorm.ErrRecordNotFound):
				report.Issues = append(report.Issues, LegacyBillingIssue{
					UserID:  row.ID,
					Kind:    "missing_customer",
					Details: fmt.Sprintf("expected billing_customers row for stripe customer %q", customerID),
				})
			default:
				return nil, err
			}
		}

		expected, ok := legacySubscriptionSnapshot(row)
		if !ok {
			continue
		}
		report.SubscriptionCandidates++
		actual, err := NewSubscriptionRepo(db).FindByUser(ctx, row.ID)
		switch {
		case err == nil:
			report.Issues = append(report.Issues, compareLegacySnapshot(row.ID, expected, actual)...)
		case errors.Is(err, gorm.ErrRecordNotFound):
			report.Issues = append(report.Issues, LegacyBillingIssue{
				UserID:  row.ID,
				Kind:    "missing_subscription",
				Details: "expected billing_subscriptions row",
			})
		default:
			return nil, err
		}
	}

	return report, nil
}

func loadLegacyUserBillingRows(ctx context.Context, db *gorm.DB, opts LegacyBillingSyncOptions) ([]legacyUserBillingRow, error) {
	query := db.WithContext(ctx).
		Table("users").
		Select(
			"id",
			"email",
			"plan",
			"stripe_customer_id",
			"stripe_subscription_id",
			"stripe_price_id",
			"stripe_product_id",
			"billing_status",
			"billing_period_start",
			"billing_period_end",
			"cancel_effective_at",
		)

	if strings.TrimSpace(opts.UserID) != "" {
		query = query.Where("id = ?", strings.TrimSpace(opts.UserID))
	}
	query = query.Where(
		`COALESCE(stripe_customer_id, '') <> '' OR
         COALESCE(stripe_subscription_id, '') <> '' OR
         COALESCE(stripe_price_id, '') <> '' OR
         COALESCE(stripe_product_id, '') <> '' OR
         COALESCE(billing_status, '') <> '' OR
         billing_period_start IS NOT NULL OR
         billing_period_end IS NOT NULL OR
         cancel_effective_at IS NOT NULL`,
	)
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	query = query.Order("created_at ASC")

	var rows []legacyUserBillingRow
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func legacySubscriptionSnapshot(row legacyUserBillingRow) (domain.SubscriptionSnapshot, bool) {
	if !hasLegacySubscriptionData(row) {
		return domain.SubscriptionSnapshot{}, false
	}

	plan := normalizeLegacyPlan(row.Plan)
	status := normalizeLegacyStatus(row.BillingStatus)
	productType := domain.ProductSubscription
	if plan == domain.PlanLifetime && strings.TrimSpace(row.StripeSubscriptionID) == "" {
		productType = domain.ProductLifetime
	}
	if status == "" {
		status = domain.StatusActive
	}

	return domain.SubscriptionSnapshot{
		ProviderSubscriptionID: strings.TrimSpace(row.StripeSubscriptionID),
		ProviderCustomerID:     strings.TrimSpace(row.StripeCustomerID),
		ProviderPriceID:        strings.TrimSpace(row.StripePriceID),
		ProviderProductID:      strings.TrimSpace(row.StripeProductID),
		ProductType:            productType,
		Plan:                   plan,
		Status:                 status,
		PeriodStart:            row.BillingPeriodStart,
		PeriodEnd:              row.BillingPeriodEnd,
		CancelEffectiveAt:      row.CancelEffectiveAt,
	}, true
}

func hasLegacySubscriptionData(row legacyUserBillingRow) bool {
	return strings.TrimSpace(row.StripeSubscriptionID) != "" ||
		strings.TrimSpace(row.StripeCustomerID) != "" ||
		strings.TrimSpace(row.StripePriceID) != "" ||
		strings.TrimSpace(row.StripeProductID) != "" ||
		strings.TrimSpace(row.BillingStatus) != "" ||
		row.BillingPeriodStart != nil ||
		row.BillingPeriodEnd != nil ||
		row.CancelEffectiveAt != nil
}

func normalizeLegacyPlan(raw string) domain.PlanType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.PlanStarter):
		return domain.PlanStarter
	case string(domain.PlanPro):
		return domain.PlanPro
	case string(domain.PlanPremium):
		return domain.PlanPremium
	case string(domain.PlanLifetime):
		return domain.PlanLifetime
	default:
		return domain.PlanFree
	}
}

func normalizeLegacyStatus(raw string) domain.SubscriptionStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.StatusTrialing):
		return domain.StatusTrialing
	case string(domain.StatusActive):
		return domain.StatusActive
	case string(domain.StatusPastDue):
		return domain.StatusPastDue
	case string(domain.StatusCanceling):
		return domain.StatusCanceling
	case string(domain.StatusCanceled):
		return domain.StatusCanceled
	case string(domain.StatusIncomplete):
		return domain.StatusIncomplete
	case string(domain.StatusPaymentFailed):
		return domain.StatusPaymentFailed
	default:
		return ""
	}
}

func compareLegacySnapshot(userID string, expected domain.SubscriptionSnapshot, actual *domain.BillingSubscription) []LegacyBillingIssue {
	if actual == nil {
		return []LegacyBillingIssue{{
			UserID:  userID,
			Kind:    "missing_subscription",
			Details: "expected billing_subscriptions row",
		}}
	}

	var issues []LegacyBillingIssue
	check := func(kind, got, want string) {
		if strings.TrimSpace(got) == strings.TrimSpace(want) {
			return
		}
		issues = append(issues, LegacyBillingIssue{
			UserID:  userID,
			Kind:    kind,
			Details: fmt.Sprintf("got %q want %q", got, want),
		})
	}

	check("subscription_customer_mismatch", actual.ProviderCustomerID, expected.ProviderCustomerID)
	check("subscription_id_mismatch", actual.ProviderSubscriptionID, expected.ProviderSubscriptionID)
	check("subscription_price_mismatch", actual.ProviderPriceID, expected.ProviderPriceID)
	check("subscription_product_mismatch", actual.ProviderProductID, expected.ProviderProductID)
	check("subscription_plan_mismatch", actual.Plan, string(expected.Plan))
	check("subscription_status_mismatch", actual.Status, string(expected.Status))
	check("subscription_product_type_mismatch", actual.ProductType, string(expected.ProductType))

	if !sameTimePtr(actual.PeriodStart, expected.PeriodStart) {
		issues = append(issues, LegacyBillingIssue{
			UserID:  userID,
			Kind:    "subscription_period_start_mismatch",
			Details: fmt.Sprintf("got %s want %s", formatTimePtr(actual.PeriodStart), formatTimePtr(expected.PeriodStart)),
		})
	}
	if !sameTimePtr(actual.PeriodEnd, expected.PeriodEnd) {
		issues = append(issues, LegacyBillingIssue{
			UserID:  userID,
			Kind:    "subscription_period_end_mismatch",
			Details: fmt.Sprintf("got %s want %s", formatTimePtr(actual.PeriodEnd), formatTimePtr(expected.PeriodEnd)),
		})
	}
	if !sameTimePtr(actual.CancelEffectiveAt, expected.CancelEffectiveAt) {
		issues = append(issues, LegacyBillingIssue{
			UserID:  userID,
			Kind:    "subscription_cancel_effective_at_mismatch",
			Details: fmt.Sprintf("got %s want %s", formatTimePtr(actual.CancelEffectiveAt), formatTimePtr(expected.CancelEffectiveAt)),
		})
	}

	return issues
}

func sameTimePtr(a, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.UTC().Equal(b.UTC())
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.UTC().Format(time.RFC3339)
}
