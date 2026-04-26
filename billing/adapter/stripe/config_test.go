package stripe

import (
	"testing"

	"github.com/brizenchi/go-modules/billing/domain"
)

func newTestConfig() Config {
	return Config{
		Enabled: true,
		SubscriptionPrices: map[domain.PlanType]map[domain.BillingInterval]string{
			domain.PlanStarter: {
				domain.IntervalMonthly: "price_starter_m",
				domain.IntervalYearly:  "price_starter_y",
			},
			domain.PlanPro: {
				domain.IntervalMonthly: "price_pro_m",
			},
			domain.PlanPremium: {
				domain.IntervalMonthly: "price_premium_m",
				domain.IntervalYearly:  "price_premium_y",
			},
		},
		CreditsPriceIDs: []string{"price_credits_a", "price_credits_b"},
		CreditsPerUnit:  40,
	}
}

func TestConfig_PriceFor(t *testing.T) {
	c := newTestConfig()
	cases := []struct {
		plan     domain.PlanType
		interval domain.BillingInterval
		want     string
	}{
		{domain.PlanStarter, domain.IntervalMonthly, "price_starter_m"},
		{domain.PlanStarter, domain.IntervalYearly, "price_starter_y"},
		{domain.PlanPro, domain.IntervalYearly, ""}, // not configured
		{domain.PlanFree, domain.IntervalMonthly, ""},
		{domain.PlanPremium, domain.IntervalMonthly, "price_premium_m"},
	}
	for _, tc := range cases {
		if got := c.PriceFor(tc.plan, tc.interval); got != tc.want {
			t.Errorf("PriceFor(%s,%s) = %q, want %q", tc.plan, tc.interval, got, tc.want)
		}
	}
}

func TestConfig_PlanForPrice(t *testing.T) {
	c := newTestConfig()
	cases := []struct {
		priceID  string
		wantPlan domain.PlanType
		wantInt  domain.BillingInterval
	}{
		{"price_starter_m", domain.PlanStarter, domain.IntervalMonthly},
		{"price_starter_y", domain.PlanStarter, domain.IntervalYearly},
		{"price_pro_m", domain.PlanPro, domain.IntervalMonthly},
		{"price_premium_y", domain.PlanPremium, domain.IntervalYearly},
		{"unknown", domain.PlanFree, ""},
		{"", domain.PlanFree, ""},
	}
	for _, tc := range cases {
		gotPlan, gotInt := c.PlanForPrice(tc.priceID)
		if gotPlan != tc.wantPlan || gotInt != tc.wantInt {
			t.Errorf("PlanForPrice(%q) = (%s,%s), want (%s,%s)",
				tc.priceID, gotPlan, gotInt, tc.wantPlan, tc.wantInt)
		}
	}
}

func TestConfig_IsCreditsPriceID(t *testing.T) {
	c := newTestConfig()
	if !c.IsCreditsPriceID("price_credits_a") {
		t.Error("expected price_credits_a to be credits")
	}
	if c.IsCreditsPriceID("price_starter_m") {
		t.Error("expected price_starter_m NOT to be credits")
	}
	if c.IsCreditsPriceID("") {
		t.Error("expected empty to not be credits")
	}
}
