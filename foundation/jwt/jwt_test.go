package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHS256_RoundTrip(t *testing.T) {
	s, err := NewHS256("test-secret", Options{Issuer: "auth-svc"})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.Sign(Claims{
		Subject: "u-1",
		TTL:     time.Hour,
		Extra:   map[string]any{"role": "admin"},
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	parsed, err := s.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Subject != "u-1" {
		t.Errorf("sub = %q", parsed.Subject)
	}
	if parsed.Issuer != "auth-svc" {
		t.Errorf("iss = %q", parsed.Issuer)
	}
	if parsed.Extra["role"] != "admin" {
		t.Errorf("role = %v", parsed.Extra["role"])
	}
	if parsed.ExpiresAt.IsZero() || parsed.IssuedAt.IsZero() {
		t.Error("missing iat/exp")
	}
}

func TestHS256_RejectsEmptySecret(t *testing.T) {
	_, err := NewHS256("", Options{})
	if !errors.Is(err, ErrSecretRequired) {
		t.Errorf("expected ErrSecretRequired, got %v", err)
	}
}

func TestSign_RequiresTTL(t *testing.T) {
	s, _ := NewHS256("x", Options{})
	_, err := s.Sign(Claims{Subject: "u"})
	if !errors.Is(err, ErrTTLRequired) {
		t.Errorf("expected ErrTTLRequired, got %v", err)
	}
}

func TestParse_RejectsTampered(t *testing.T) {
	s, _ := NewHS256("x", Options{})
	tok, _ := s.Sign(Claims{Subject: "u", TTL: time.Hour})
	// Tamper a char IN the payload segment (between the two dots).
	// Mutating the last sig char is unreliable: base64url's last char
	// encodes only 4 bits, so swapping characters that share the high
	// 2 bits leaves the decoded signature bytes unchanged.
	parts := strings.SplitN(tok, ".", 3)
	if len(parts) != 3 {
		t.Fatalf("malformed jwt: %q", tok)
	}
	mid := []byte(parts[1])
	// Flip one byte in the middle of the payload to a different valid
	// base64url char. The payload is JSON, so any unequal char produces
	// a different signed content and will fail HMAC verification.
	idx := len(mid) / 2
	if mid[idx] == 'A' {
		mid[idx] = 'B'
	} else {
		mid[idx] = 'A'
	}
	bad := parts[0] + "." + string(mid) + "." + parts[2]
	_, err := s.Parse(bad)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParse_RejectsWrongSecret(t *testing.T) {
	s1, _ := NewHS256("secret-1", Options{})
	s2, _ := NewHS256("secret-2", Options{})
	tok, _ := s1.Sign(Claims{Subject: "u", TTL: time.Hour})
	_, err := s2.Parse(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParse_RejectsExpired(t *testing.T) {
	// JWT exp is second-precision; sub-second TTLs don't reliably push
	// the expiry past now until we cross a full second boundary, which
	// made earlier 2ms-sleep variants flaky under -race. Sleep 1.1s to
	// guarantee we cross.
	s, _ := NewHS256("x", Options{})
	tok, _ := s.Sign(Claims{Subject: "u", TTL: 100 * time.Millisecond})
	time.Sleep(1100 * time.Millisecond)
	_, err := s.Parse(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected to also wrap ErrExpired, got %v", err)
	}
}

func TestParse_LeewayTolerates(t *testing.T) {
	// We need to assert: token expired by wall clock, but still accepted
	// thanks to leeway. Both exp and now are second-precision, so use
	// a real-second TTL + sleep past it + leeway clearly larger than the gap.
	s, _ := NewHS256("x", Options{Leeway: 5 * time.Second})
	tok, _ := s.Sign(Claims{Subject: "u", TTL: 500 * time.Millisecond})
	time.Sleep(1500 * time.Millisecond) // past TTL by ~1s, well within 5s leeway
	if _, err := s.Parse(tok); err != nil {
		t.Errorf("expected leeway to tolerate, got %v", err)
	}
}

func TestParse_RejectsWrongIssuer(t *testing.T) {
	signer, _ := NewHS256("x", Options{Issuer: "good"})
	verifier, _ := NewHS256("x", Options{Issuer: "expected-other"})
	tok, _ := signer.Sign(Claims{Subject: "u", TTL: time.Hour})
	if _, err := verifier.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken for wrong issuer, got %v", err)
	}
}

func TestRS256_RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewRS256(priv, &priv.PublicKey, Options{})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.Sign(Claims{Subject: "u", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := s.Parse(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Subject != "u" {
		t.Errorf("sub = %q", parsed.Subject)
	}
}

func TestRS256_VerifyOnly(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := NewRS256(priv, &priv.PublicKey, Options{})
	tok, _ := signer.Sign(Claims{Subject: "u", TTL: time.Hour})

	verifyOnly, _ := NewRS256(nil, &priv.PublicKey, Options{})
	if _, err := verifyOnly.Parse(tok); err != nil {
		t.Errorf("verify-only parse failed: %v", err)
	}
	if _, err := verifyOnly.Sign(Claims{Subject: "u", TTL: time.Hour}); err == nil {
		t.Error("verify-only signer should fail to Sign")
	}
}

func TestParse_RejectsAlgConfusion(t *testing.T) {
	// Token signed with HS256 must NOT pass verification by an RS256 signer
	// even if the secret happens to match — the WithValidMethods guard is
	// the well-known JWT alg-confusion mitigation.
	hsSigner, _ := NewHS256("x", Options{})
	tok, _ := hsSigner.Sign(Claims{Subject: "u", TTL: time.Hour})

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	rsSigner, _ := NewRS256(priv, &priv.PublicKey, Options{})
	if _, err := rsSigner.Parse(tok); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected alg-confusion rejection, got %v", err)
	}
}
