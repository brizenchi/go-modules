package http

import (
	"strconv"
	"testing"
)

func TestSanitizeMetadata_DropsReservedAndEmpty(t *testing.T) {
	in := map[string]string{
		"referral":   "rwf_abc",
		"campaign":   "spring",
		"user_id":    "spoof",   // reserved → dropped
		"plan":       "premium", // reserved → dropped
		"":           "v",       // empty key → dropped
		"k":          "",        // empty value → dropped
		"  spaced  ": "  v  ",   // trimmed
	}
	out := sanitizeMetadata(in)

	if got := out["referral"]; got != "rwf_abc" {
		t.Errorf("referral = %q", got)
	}
	if got := out["campaign"]; got != "spring" {
		t.Errorf("campaign = %q", got)
	}
	if _, ok := out["user_id"]; ok {
		t.Errorf("user_id should be dropped (reserved)")
	}
	if _, ok := out["plan"]; ok {
		t.Errorf("plan should be dropped (reserved)")
	}
	if got := out["spaced"]; got != "v" {
		t.Errorf("trim failed, got %q", got)
	}
	if len(out) != 3 {
		t.Errorf("expected 3 entries, got %d: %+v", len(out), out)
	}
}

func TestSanitizeMetadata_NilOnEmpty(t *testing.T) {
	if out := sanitizeMetadata(nil); out != nil {
		t.Errorf("nil input should yield nil, got %+v", out)
	}
	if out := sanitizeMetadata(map[string]string{"user_id": "x"}); out != nil {
		t.Errorf("only-reserved input should yield nil, got %+v", out)
	}
}

func TestSanitizeMetadata_EnforcesCap(t *testing.T) {
	in := make(map[string]string, 50)
	for i := range 50 {
		in["k"+strconv.Itoa(i)] = "v"
	}
	out := sanitizeMetadata(in)
	if len(out) != maxMetadataEntries {
		t.Errorf("len = %d, want %d", len(out), maxMetadataEntries)
	}
}
