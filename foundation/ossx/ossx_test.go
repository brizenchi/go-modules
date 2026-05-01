package ossx

import (
	"errors"
	"testing"
)

func TestValidateKey(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"ok-simple", "users/42/avatar.png", false},
		{"ok-deep", "a/b/c/d/e/f/g.txt", false},
		{"ok-unicode", "用户/头像.png", false},
		{"empty", "", true},
		{"leading-slash", "/foo", true},
		{"too-long", string(make([]byte, 1025)), true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateKey(tc.key)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateKey(%q) err=%v wantErr=%v", tc.key, err, tc.wantErr)
			}
			if tc.wantErr && !errors.Is(err, ErrInvalidKey) {
				t.Fatalf("expected ErrInvalidKey, got %v", err)
			}
		})
	}
}
