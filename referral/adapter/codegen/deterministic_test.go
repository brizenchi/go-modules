package codegen

import (
	"strings"
	"testing"
)

func TestDeterministic_StableForSameUser(t *testing.T) {
	g := NewDeterministic("CLAW", 8)
	a := g.Generate("user-12345-abcdef")
	b := g.Generate("user-12345-abcdef")
	if a != b {
		t.Errorf("not stable: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, "CLAW") {
		t.Errorf("missing prefix: %q", a)
	}
}

func TestDeterministic_StripsNonAlphanum(t *testing.T) {
	g := NewDeterministic("R", 4)
	got := g.Generate("ab-cd_ef!@#gh")
	want := "RABCD"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeterministic_PadsShortInput(t *testing.T) {
	g := NewDeterministic("R", 6)
	got := g.Generate("ab")
	want := "RABXXXX"
	if got != want {
		t.Errorf("got %q, want %q (padded)", got, want)
	}
}

func TestRandom_RespectsLengthAndPrefix(t *testing.T) {
	g := NewRandom("R", 10)
	a := g.Generate("user")
	if !strings.HasPrefix(a, "R") {
		t.Errorf("missing prefix: %q", a)
	}
	if len(a)-1 != 10 {
		t.Errorf("length %d, want 10 after prefix", len(a)-1)
	}
}

func TestRandom_GeneratesDifferentValues(t *testing.T) {
	g := NewRandom("", 12)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		seen[g.Generate("u")] = true
	}
	if len(seen) < 40 {
		t.Errorf("low entropy: only %d distinct in 50", len(seen))
	}
}
