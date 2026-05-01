package randx

import (
	"strings"
	"testing"
)

func TestCode_LengthAndCharset(t *testing.T) {
	cases := []struct {
		name string
		n    int
		set  Charset
	}{
		{"numeric-6", 6, Numeric},
		{"numeric-1", 1, Numeric},
		{"alnum-32", 32, AlphaNum}, // also stresses the inner loop
		{"unambig-8", 8, Unambiguous},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := Code(tc.n, tc.set)
			if err != nil {
				t.Fatalf("Code error: %v", err)
			}
			if len(got) != tc.n {
				t.Fatalf("len = %d, want %d", len(got), tc.n)
			}
			for _, r := range got {
				if !strings.ContainsRune(string(tc.set), r) {
					t.Fatalf("rune %q not in charset %q", r, tc.set)
				}
			}
		})
	}
}

func TestCode_Errors(t *testing.T) {
	if _, err := Code(0, Numeric); err == nil {
		t.Fatal("expected error for length=0")
	}
	if _, err := Code(-1, Numeric); err == nil {
		t.Fatal("expected error for negative length")
	}
	if _, err := Code(6, ""); err == nil {
		t.Fatal("expected error for empty charset")
	}
}

func TestNumericCode_DefaultLength(t *testing.T) {
	got, err := NumericCode(0)
	if err != nil {
		t.Fatalf("NumericCode error: %v", err)
	}
	if len(got) != 6 {
		t.Fatalf("default length = %d, want 6", len(got))
	}
}

func TestMustCode_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = MustCode(-1, Numeric)
}

func TestTokens(t *testing.T) {
	hex, err := HexToken(16)
	if err != nil {
		t.Fatalf("HexToken: %v", err)
	}
	if len(hex) != 32 {
		t.Fatalf("hex len = %d, want 32", len(hex))
	}

	url, err := URLToken(16)
	if err != nil {
		t.Fatalf("URLToken: %v", err)
	}
	if strings.ContainsAny(url, "+/=") {
		t.Fatalf("URLToken should be RawURL-safe: %q", url)
	}

	b32, err := Base32Token(10)
	if err != nil {
		t.Fatalf("Base32Token: %v", err)
	}
	if strings.ContainsRune(b32, '=') {
		t.Fatalf("Base32Token should have no padding: %q", b32)
	}
}

func TestBytes_Distinct(t *testing.T) {
	a, err := Bytes(32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Bytes(32)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) == string(b) {
		t.Fatal("two 32-byte reads should be distinct with overwhelming probability")
	}
}
