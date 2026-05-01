package domain

import "testing"

func TestProvider_Valid(t *testing.T) {
	cases := []struct {
		p    Provider
		want bool
	}{
		{ProviderEmail, true},
		{ProviderGoogle, true},
		{ProviderAnthropic, true},
		{Provider(""), false},
		{Provider("github"), false},
	}
	for _, c := range cases {
		if got := c.p.Valid(); got != c.want {
			t.Errorf("Provider(%q).Valid() = %v, want %v", c.p, got, c.want)
		}
	}
}
