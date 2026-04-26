// Package codegen provides reference CodeGenerator implementations.
package codegen

import (
	"crypto/rand"
	"strings"

	"github.com/brizenchi/go-modules/referral/port"
)

// Deterministic derives a code from the user_id by stripping non-alphanum
// characters, uppercasing, truncating/padding to Length, and prefixing.
//
// Stable across restarts; no DB collision possible (user_id is unique).
type Deterministic struct {
	Prefix string
	Length int
	Pad    rune // padding char if user id shorter than Length, default 'X'
}

func NewDeterministic(prefix string, length int) *Deterministic {
	if length <= 0 {
		length = 8
	}
	return &Deterministic{Prefix: prefix, Length: length, Pad: 'X'}
}

func (d *Deterministic) Generate(userID string) string {
	src := strings.ToUpper(strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return r
		}
		return -1
	}, userID))
	if len(src) > d.Length {
		src = src[:d.Length]
	}
	pad := d.Pad
	if pad == 0 {
		pad = 'X'
	}
	for len(src) < d.Length {
		src += string(pad)
	}
	return d.Prefix + src
}

// Random produces a random alphanumeric code (0-9A-Z).
//
// Use this if you want unguessable codes; pair with a CodeRepository
// to enforce uniqueness across users (caller retries on ErrCodeCollision).
type Random struct {
	Prefix string
	Length int
}

func NewRandom(prefix string, length int) *Random {
	if length <= 0 {
		length = 8
	}
	return &Random{Prefix: prefix, Length: length}
}

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (r *Random) Generate(_ string) string {
	buf := make([]byte, r.Length)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand.Read on a working OS shouldn't fail; on the off
		// chance it does we fall back to a hardcoded seed (not great
		// but predictable).
		copy(buf, []byte("XXXXXXXX"))
	}
	out := make([]byte, r.Length)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return r.Prefix + string(out)
}

var (
	_ port.CodeGenerator = (*Deterministic)(nil)
	_ port.CodeGenerator = (*Random)(nil)
)
