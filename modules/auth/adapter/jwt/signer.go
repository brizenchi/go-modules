// Package jwt provides HS256 JWT-backed implementations of port.TokenSigner
// and port.WSTicketSigner.
//
// Symmetric secrets are intentional: this is a server-issued, server-verified
// session token. Switch to RS256 if you need third parties to verify tokens
// without sharing your secret.
package jwt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Config holds the static configuration for HS256 signing.
type Config struct {
	Secret    string
	Issuer    string        // optional, embedded as "iss" claim
	UserTTL   time.Duration // default token TTL when caller passes 0
	TicketTTL time.Duration // default ws ticket TTL when caller passes 0
}

// Signer implements port.TokenSigner.
type Signer struct {
	cfg Config
}

func NewSigner(cfg Config) (*Signer, error) {
	if cfg.Secret == "" {
		return nil, fmt.Errorf("jwt: secret required")
	}
	return &Signer{cfg: cfg}, nil
}

type userClaims struct {
	jwtv5.RegisteredClaims
	// UserID duplicates Subject. Some legacy middleware reads "user_id"
	// instead of the standard "sub" claim; this keeps tokens compatible
	// with both. Carries no security cost — the JWT signature still
	// covers both claims.
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func (s *Signer) Issue(id domain.Identity, ttl time.Duration) (*domain.Token, error) {
	if ttl <= 0 {
		ttl = s.cfg.UserTTL
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	now := time.Now().UTC()
	claims := userClaims{
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   id.UserID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
		UserID: id.UserID,
		Email:  id.Email,
		Role:   string(id.Role),
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("jwt: sign: %w", err)
	}
	return &domain.Token{Value: str, ExpiresAt: now.Add(ttl)}, nil
}

func (s *Signer) Parse(value string) (*domain.Identity, error) {
	claims := &userClaims{}
	tok, err := jwtv5.ParseWithClaims(value, claims, func(*jwtv5.Token) (any, error) {
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidToken, err)
	}
	uid := claims.Subject
	if uid == "" {
		uid = claims.UserID
	}
	return &domain.Identity{
		UserID: uid,
		Email:  claims.Email,
		Role:   domain.Role(claims.Role),
	}, nil
}

// TicketSigner implements port.WSTicketSigner.
//
// The Scope map is encoded as a JSON-marshaled "scp" claim — this lets
// arbitrary string→string maps round-trip cleanly without nesting issues.
type TicketSigner struct {
	cfg Config
}

func NewTicketSigner(cfg Config) (*TicketSigner, error) {
	if cfg.Secret == "" {
		return nil, fmt.Errorf("jwt: secret required")
	}
	return &TicketSigner{cfg: cfg}, nil
}

type ticketClaims struct {
	jwtv5.RegisteredClaims
	Scope string `json:"scp"`
}

func (s *TicketSigner) Issue(userID string, scope map[string]string, ttl time.Duration) (*domain.WSTicket, error) {
	if ttl <= 0 {
		ttl = s.cfg.TicketTTL
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, ticketClaims{
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    s.cfg.Issuer,
			Subject:   userID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
		Scope: string(scopeJSON),
	})
	str, err := tok.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("jwt: sign ticket: %w", err)
	}
	return &domain.WSTicket{
		Value:     str,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: now.Add(ttl),
	}, nil
}

func (s *TicketSigner) Parse(value string) (*domain.WSTicket, error) {
	claims := &ticketClaims{}
	tok, err := jwtv5.ParseWithClaims(value, claims, func(*jwtv5.Token) (any, error) {
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidWSTicket, err)
	}
	scope := map[string]string{}
	if claims.Scope != "" {
		if err := json.Unmarshal([]byte(claims.Scope), &scope); err != nil {
			return nil, fmt.Errorf("%w: scope decode: %v", domain.ErrInvalidWSTicket, err)
		}
	}
	exp, _ := claims.GetExpirationTime()
	expAt := time.Time{}
	if exp != nil {
		expAt = exp.Time
	}
	return &domain.WSTicket{
		Value:     value,
		UserID:    claims.Subject,
		Scope:     scope,
		ExpiresAt: expAt,
	}, nil
}

var (
	_ port.TokenSigner    = (*Signer)(nil)
	_ port.WSTicketSigner = (*TicketSigner)(nil)
)
