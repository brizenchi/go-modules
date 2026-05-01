// Package randx provides cryptographically secure random helpers used
// across services: numeric verification codes, URL-safe tokens, and
// short alphanumeric IDs. All sources read from crypto/rand — never use
// math/rand for anything user-facing.
package randx

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
)

// Charset is a builder for custom alphabets. Use one of the predefined
// charsets unless you have a reason not to.
type Charset string

const (
	// Numeric: "0123456789". Use for SMS / email verification codes.
	Numeric Charset = "0123456789"
	// LowerAlphaNum: digits + lowercase letters. URL-safe and human-typeable.
	LowerAlphaNum Charset = "0123456789abcdefghijklmnopqrstuvwxyz"
	// AlphaNum: digits + both cases.
	AlphaNum Charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// Unambiguous strips look-alikes (0/O, 1/l/I) — best for short codes
	// users have to retype.
	Unambiguous Charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz"
)

// Code returns a length-n random string drawn uniformly from charset.
//
//	randx.Code(6, randx.Numeric)        // "473829"
//	randx.Code(8, randx.Unambiguous)    // "kP7Ham3X"
//
// Returns an error if length <= 0 or charset is empty.
func Code(length int, charset Charset) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("randx: length must be > 0")
	}
	if len(charset) == 0 {
		return "", fmt.Errorf("randx: charset must not be empty")
	}
	max := big.NewInt(int64(len(charset)))
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("randx: read entropy: %w", err)
		}
		out[i] = charset[n.Int64()]
	}
	return string(out), nil
}

// MustCode is Code without the error return. Panics on entropy failure.
// Safe for app-startup constants and tests.
func MustCode(length int, charset Charset) string {
	s, err := Code(length, charset)
	if err != nil {
		panic(err)
	}
	return s
}

// NumericCode returns a length-n digit string. Convenience wrapper for
// SMS / email verification codes; default length is 6.
func NumericCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	return Code(length, Numeric)
}

// Bytes reads n cryptographically random bytes.
func Bytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("randx: n must be > 0")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("randx: read entropy: %w", err)
	}
	return b, nil
}

// HexToken returns a hex-encoded token sourced from n random bytes.
// The resulting string has length 2*n. Use for non-user-facing tokens
// (idempotency keys, internal handles).
func HexToken(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// URLToken returns a URL-safe base64 (no padding) token from n random
// bytes. Use for password-reset / email-verify links and webhook secrets.
func URLToken(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Base32Token returns a Crockford-style base32 token (no padding). More
// human-readable than base64; useful for backup codes.
func Base32Token(n int) (string, error) {
	b, err := Bytes(n)
	if err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
