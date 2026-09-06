package stripe

import (
	"testing"

	"github.com/brizenchi/go-modules/modules/billing/domain"
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
		LifetimePriceID: "price_lifetime",
		CreditsPriceIDs: []string{"price_1CreditsAlpha123456789", "price_1CreditsBravo123456789"},
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
		{"price_lifetime", domain.PlanLifetime, ""},
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
	if !c.IsCreditsPriceID("price_1CreditsAlpha123456789") {
		t.Error("expected configured valid price to be credits")
	}
	if c.IsCreditsPriceID("price_starter_m") {
		t.Error("expected price_starter_m NOT to be credits")
	}
	if c.IsCreditsPriceID("") {
		t.Error("expected empty to not be credits")
	}
}

func TestValidPriceID(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "price_1TBFpjQ4kdQzysE6eeAA5D38", want: true},
		{value: ""},
		{value: "price_..."},
		{value: "prod_1TBFpjQ4kdQzysE6eeAA5D38"},
		{value: "price_tooShort"},
		{value: "price_placeholder123456789"},
		{value: "price_1TBFpjQ4kdQzysE6_bad"},
		{value: " price_1TBFpjQ4kdQzysE6eeAA5D38"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := ValidPriceID(tt.value); got != tt.want {
				t.Fatalf("ValidPriceID(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
