// Package jwt is a thin, opinionated JWT helper for service-to-service
// and user-session use cases.
//
// Design goals:
//   - Generic Claims struct + custom payload via map (no codegen needed)
//   - HS256 (symmetric) for trusted-party tokens; RS256 for cross-org
//   - Sane defaults: short TTL, mandatory exp/iat, leeway-aware verify
//   - Stdlib + golang-jwt/v5 only — no host-app coupling
//
// Recipes:
//
//	signer := jwt.NewHS256("super-secret", jwt.Options{Issuer: "auth-svc"})
//	token, _ := signer.Sign(jwt.Claims{
//	    Subject: "user-42",
//	    TTL:     time.Hour,
//	    Extra:   map[string]any{"role": "admin"},
//	})
//
//	parsed, _ := signer.Parse(token)
//	uid := parsed.Subject
//	role, _ := parsed.Extra["role"].(string)
package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Errors returned by Sign / Parse.
var (
	ErrSecretRequired = errors.New("jwt: secret required")
	ErrInvalidToken   = errors.New("jwt: invalid token")
	ErrExpired        = errors.New("jwt: token expired")
	ErrTTLRequired    = errors.New("jwt: ttl must be > 0")
)

// Options tweaks all signers built by this package.
type Options struct {
	// Issuer is embedded as the standard "iss" claim. Optional.
	Issuer string

	// Audience is embedded as "aud". Optional.
	Audience []string

	// Leeway is the clock-skew window allowed when validating exp/nbf/iat.
	// Default: 0 (strict).
	Leeway time.Duration
}

// Claims is the per-token payload.
type Claims struct {
	// Subject is "sub" — typically the user id.
	Subject string
	// TTL is how long from now the token is valid. Required.
	TTL time.Duration
	// Extra holds custom claim names → values. Encoded into the JWT
	// payload alongside the registered claims; round-trips through
	// Parse intact.
	Extra map[string]any
}

// Parsed is the verified outcome of Parse.
type Parsed struct {
	Subject   string
	Issuer    string
	Audience  []string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Extra     map[string]any
}

// Signer mints and verifies tokens. Use NewHS256 or NewRS256 to construct.
type Signer struct {
	method    jwtv5.SigningMethod
	signKey   any
	verifyKey any
	opts      Options
}

// NewHS256 returns a Signer using HMAC-SHA256.
func NewHS256(secret string, opts Options) (*Signer, error) {
	if secret == "" {
		return nil, ErrSecretRequired
	}
	return &Signer{
		method:    jwtv5.SigningMethodHS256,
		signKey:   []byte(secret),
		verifyKey: []byte(secret),
		opts:      opts,
	}, nil
}

// NewRS256 returns a Signer using RSA-SHA256 with separate sign/verify keys.
//
// Pass nil signKey to build a verify-only signer (e.g. for downstream
// services that only need to validate tokens, not mint them).
func NewRS256(signKey *rsa.PrivateKey, verifyKey *rsa.PublicKey, opts Options) (*Signer, error) {
	if verifyKey == nil {
		return nil, fmt.Errorf("jwt: verify key required")
	}
	s := &Signer{
		method:    jwtv5.SigningMethodRS256,
		verifyKey: verifyKey,
		opts:      opts,
	}
	if signKey != nil {
		s.signKey = signKey
	}
	return s, nil
}

// Sign produces a JWT carrying the given claims.
func (s *Signer) Sign(c Claims) (string, error) {
	if s.signKey == nil {
		return "", fmt.Errorf("jwt: signer is verify-only")
	}
	if c.TTL <= 0 {
		return "", ErrTTLRequired
	}
	now := time.Now().UTC()

	mapped := jwtv5.MapClaims{
		"sub": c.Subject,
		"iat": now.Unix(),
		"exp": now.Add(c.TTL).Unix(),
	}
	if s.opts.Issuer != "" {
		mapped["iss"] = s.opts.Issuer
	}
	if len(s.opts.Audience) > 0 {
		mapped["aud"] = s.opts.Audience
	}
	for k, v := range c.Extra {
		mapped[k] = v
	}

	tok := jwtv5.NewWithClaims(s.method, mapped)
	str, err := tok.SignedString(s.signKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return str, nil
}

// Parse validates a token's signature + standard claims and returns the
// embedded payload. Returns ErrInvalidToken for any signature/format
// failure and ErrExpired for an expired token (also wrapped in ErrInvalidToken
// via errors.Is).
func (s *Signer) Parse(value string) (*Parsed, error) {
	parserOpts := []jwtv5.ParserOption{
		jwtv5.WithLeeway(s.opts.Leeway),
		jwtv5.WithValidMethods([]string{s.method.Alg()}),
	}
	if s.opts.Issuer != "" {
		parserOpts = append(parserOpts, jwtv5.WithIssuer(s.opts.Issuer))
	}
	parser := jwtv5.NewParser(parserOpts...)

	claims := jwtv5.MapClaims{}
	tok, err := parser.ParseWithClaims(value, claims, func(*jwtv5.Token) (any, error) {
		return s.verifyKey, nil
	})
	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidToken, ErrExpired)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !tok.Valid {
		return nil, ErrInvalidToken
	}

	out := &Parsed{Extra: map[string]any{}}
	if v, ok := claims["sub"].(string); ok {
		out.Subject = v
	}
	if v, ok := claims["iss"].(string); ok {
		out.Issuer = v
	}
	switch v := claims["aud"].(type) {
	case string:
		out.Audience = []string{v}
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok {
				out.Audience = append(out.Audience, s)
			}
		}
	}
	if iat, ok := claims["iat"].(float64); ok {
		out.IssuedAt = time.Unix(int64(iat), 0).UTC()
	}
	if exp, ok := claims["exp"].(float64); ok {
		out.ExpiresAt = time.Unix(int64(exp), 0).UTC()
	}
	for k, v := range claims {
		switch k {
		case "sub", "iss", "aud", "iat", "exp", "nbf":
		default:
			out.Extra[k] = v
		}
	}
	return out, nil
}
