package rdx

import (
	"testing"
	"time"
)

func TestPrefix(t *testing.T) {
	if got := Prefix("", "k"); got != "k" {
		t.Errorf("empty prefix: %q", got)
	}
	if got := Prefix("svc", "k"); got != "svc:k" {
		t.Errorf("prefixed: %q", got)
	}
}

func TestNonZeroHelpers(t *testing.T) {
	if nonZeroInt(0, 5) != 5 || nonZeroInt(7, 5) != 7 {
		t.Error("nonZeroInt")
	}
	if nonZeroDur(0, time.Second) != time.Second || nonZeroDur(2*time.Second, time.Second) != 2*time.Second {
		t.Error("nonZeroDur")
	}
}

func TestRandomTokenLengthAndCharset(t *testing.T) {
	tok, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 32 {
		t.Errorf("len = %d, want 32", len(tok))
	}
	for _, r := range tok {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			t.Errorf("non-alphanum char: %q", r)
		}
	}
	tok2, _ := randomToken()
	if tok == tok2 {
		t.Error("two random tokens collided — entropy broken")
	}
}
