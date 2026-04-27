package stripe

import (
	"testing"

	"github.com/brizenchi/go-modules/billing/domain"
)

func TestBuildCheckoutMetadata_SystemFieldsOverrideCaller(t *testing.T) {
	in := domain.CheckoutInput{
		UserID:      "user-real",
		Email:       "real@example.com",
		Plan:        domain.PlanPro,
		Interval:    domain.IntervalMonthly,
		ProductType: domain.ProductSubscription,
		Metadata: map[string]string{
			"user_id":  "user-spoof", // attempt to override
			"email":    "spoof@example.com",
			"plan":     "premium",
			"referral": "rwf_abc",
			"campaign": "spring",
		},
	}

	m := buildCheckoutMetadata(in, "price_123", 1)

	if got := m["user_id"]; got != "user-real" {
		t.Errorf("system user_id should win, got %q", got)
	}
	if got := m["email"]; got != "real@example.com" {
		t.Errorf("system email should win, got %q", got)
	}
	if got := m["plan"]; got != "pro" {
		t.Errorf("system plan should win, got %q", got)
	}
	if got := m["referral"]; got != "rwf_abc" {
		t.Errorf("caller referral lost, got %q", got)
	}
	if got := m["campaign"]; got != "spring" {
		t.Errorf("caller campaign lost, got %q", got)
	}
	if got := m["price_id"]; got != "price_123" {
		t.Errorf("price_id = %q", got)
	}
	if got := m["quantity"]; got != "1" {
		t.Errorf("quantity = %q", got)
	}
}

// TestBuildCheckoutMetadata_MatchesReservedSet pins the stripe provider's
// system fields to domain.ReservedMetadataKeys. Adding a new system field
// without updating the reserved set (or vice-versa) breaks this test.
func TestBuildCheckoutMetadata_MatchesReservedSet(t *testing.T) {
	in := domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x.test",
		Plan:        domain.PlanPro,
		Interval:    domain.IntervalMonthly,
		ProductType: domain.ProductSubscription,
	}
	got := buildCheckoutMetadata(in, "p", 1)
	if len(got) != len(domain.ReservedMetadataKeys) {
		t.Fatalf("system field count drift: got %d (%+v), reserved %d (%+v)",
			len(got), got, len(domain.ReservedMetadataKeys), domain.ReservedMetadataKeys)
	}
	for _, k := range domain.ReservedMetadataKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("reserved key %q not written by provider", k)
		}
	}
}

func TestBuildCheckoutMetadata_NilCallerMetadata(t *testing.T) {
	in := domain.CheckoutInput{
		UserID:      "u",
		Email:       "e@x.test",
		Plan:        domain.PlanStarter,
		Interval:    domain.IntervalYearly,
		ProductType: domain.ProductSubscription,
	}
	m := buildCheckoutMetadata(in, "p", 2)
	if m["user_id"] != "u" || m["plan"] != "starter" || m["quantity"] != "2" {
		t.Errorf("unexpected metadata: %+v", m)
	}
}
