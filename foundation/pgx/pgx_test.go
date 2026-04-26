package pgx

import (
	"strings"
	"testing"
	"time"
)

func TestEffectiveDSN_DiscreteFields(t *testing.T) {
	cfg := Config{
		Host: "h", User: "u", Password: "p",
		Database: "d", Port: 5432, SSLMode: "disable",
	}
	got := cfg.effectiveDSN()
	for _, want := range []string{"host=h", "user=u", "password=p", "dbname=d", "port=5432", "sslmode=disable", "TimeZone=UTC"} {
		if !strings.Contains(got, want) {
			t.Errorf("dsn missing %q in %q", want, got)
		}
	}
}

func TestEffectiveDSN_DSNTakesPrecedence(t *testing.T) {
	cfg := Config{
		DSN:  "postgres://x/y",
		Host: "ignored",
	}
	if cfg.effectiveDSN() != "postgres://x/y" {
		t.Errorf("dsn = %q", cfg.effectiveDSN())
	}
}

func TestEffectiveDSN_DefaultsTimeZoneAndSSL(t *testing.T) {
	cfg := Config{Host: "h", User: "u", Database: "d", Port: 5432}
	got := cfg.effectiveDSN()
	if !strings.Contains(got, "TimeZone=UTC") || !strings.Contains(got, "sslmode=disable") {
		t.Errorf("missing defaults: %q", got)
	}
}

func TestNonZeroHelpers(t *testing.T) {
	if nonZeroInt(0, 5) != 5 {
		t.Error("default int")
	}
	if nonZeroInt(7, 5) != 7 {
		t.Error("explicit int")
	}
	if nonZeroDur(0, time.Second) != time.Second {
		t.Error("default dur")
	}
	if nonZeroDur(2*time.Second, time.Second) != 2*time.Second {
		t.Error("explicit dur")
	}
}
