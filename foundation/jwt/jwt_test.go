package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
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
	bad := tok[:len(tok)-1] + "A"
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
	s, _ := NewHS256("x", Options{})
	tok, _ := s.Sign(Claims{Subject: "u", TTL: time.Nanosecond})
	time.Sleep(2 * time.Millisecond)
	_, err := s.Parse(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected to also wrap ErrExpired, got %v", err)
	}
}

func TestParse_LeewayTolerates(t *testing.T) {
	// Leeway is enforced in seconds resolution by jwt/v5; use a 1s window.
	s, _ := NewHS256("x", Options{Leeway: time.Second})
	tok, _ := s.Sign(Claims{Subject: "u", TTL: time.Nanosecond})
	time.Sleep(50 * time.Millisecond)
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
