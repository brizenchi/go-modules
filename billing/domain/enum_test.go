package domain

import "testing"

func TestPlanType_Valid(t *testing.T) {
	cases := []struct {
		name string
		p    PlanType
		want bool
	}{
		{"free", PlanFree, true},
		{"starter", PlanStarter, true},
		{"pro", PlanPro, true},
		{"premium", PlanPremium, true},
		{"empty", PlanType(""), false},
		{"unknown", PlanType("enterprise"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.Valid(); got != c.want {
				t.Errorf("PlanType(%q).Valid() = %v, want %v", c.p, got, c.want)
			}
		})
	}
}

func TestCancelMode_Valid(t *testing.T) {
	cases := []struct {
		name string
		m    CancelMode
		want bool
	}{
		{"end_of_period", CancelAtPeriodEnd, true},
		{"3days", CancelIn3Days, true},
		{"empty", CancelMode(""), false},
		{"junk", CancelMode("immediately"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.m.Valid(); got != c.want {
				t.Errorf("CancelMode(%q).Valid() = %v, want %v", c.m, got, c.want)
			}
		})
	}
}
