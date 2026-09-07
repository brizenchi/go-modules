// Package jwt provides HS256 JWT-backed implementations of port.TokenSigner
// and port.WSTicketSigner.
//
// Symmetric secrets are intentional: this is a server-issued, server-verified
// session token. Switch to RS256 if you need third parties to verify tokens
// without sharing your secret.
package jwt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Config holds the static configuration for HS256 signing.
type Config struct {
	Revocations port.TokenRevocationStore
	Secret      string
	Issuer      string        // optional, embedded as "iss" claim
	UserTTL     time.Duration // default token TTL when caller passes 0
	TicketTTL   time.Duration // default ws ticket TTL when caller passes 0
}

const (
	tokenTypeAccess   = "access"
	tokenTypeWSTicket = "ws_ticket"
)

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
	Type   string `json:"token_type"`
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
			ID:        rand.Text(),
			Issuer:    s.cfg.Issuer,
			Subject:   id.UserID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
		UserID: id.UserID,
		Email:  id.Email,
		Role:   string(id.Role),
		Type:   tokenTypeAccess,
	}
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	str, err := tok.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("jwt: sign: %w", err)
	}
	return &domain.Token{Value: str, ExpiresAt: now.Add(ttl)}, nil
}

func (s *Signer) Parse(value string) (*domain.Identity, error) {
	return s.ParseContext(context.Background(), value)
}

func (s *Signer) ParseContext(ctx context.Context, value string) (*domain.Identity, error) {
	claims, err := s.parseClaims(value)
	if err != nil {
		return nil, err
	}
	if s.cfg.Revocations != nil {
		revoked, err := s.cfg.Revocations.IsRevoked(ctx, tokenHash(value))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", domain.ErrSessionUnavailable, err)
		}
		if revoked {
			return nil, domain.ErrInvalidToken
		}
	}
	return &domain.Identity{UserID: claims.Subject, Email: claims.Email, Role: domain.Role(claims.Role)}, nil
}

func (s *Signer) Revoke(ctx context.Context, value string) error {
	claims, err := s.parseClaims(value)
	if err != nil {
		return err
	}
	if s.cfg.Revocations == nil {
		return domain.ErrSessionUnavailable
	}
	if err := s.cfg.Revocations.RevokeToken(ctx, tokenHash(value), claims.ExpiresAt.Time); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrSessionUnavailable, err)
	}
	return nil
}

func tokenHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func (s *Signer) parseClaims(value string) (*userClaims, error) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, domain.ErrInvalidToken
	}
	claims := &userClaims{}
	options := []jwtv5.ParserOption{
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
		jwtv5.WithExpirationRequired(),
		jwtv5.WithStrictDecoding(),
	}
	if s.cfg.Issuer != "" {
		options = append(options, jwtv5.WithIssuer(s.cfg.Issuer))
	}
	parser := jwtv5.NewParser(options...)
	tok, err := parser.ParseWithClaims(value, claims, func(*jwtv5.Token) (any, error) {
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidToken, err)
	}
	if claims.Type != tokenTypeAccess {
		return nil, fmt.Errorf("%w: unexpected token_type %q", domain.ErrInvalidToken, claims.Type)
	}
	uid := claims.Subject
	if uid == "" {
		uid = claims.UserID
	}
	if uid == "" {
		return nil, fmt.Errorf("%w: user id required", domain.ErrInvalidToken)
	}
	claims.Subject = uid
	return claims, nil
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
	Type  string `json:"token_type"`
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
		Type:  tokenTypeWSTicket,
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
	options := []jwtv5.ParserOption{
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
		jwtv5.WithExpirationRequired(),
	}
	if s.cfg.Issuer != "" {
		options = append(options, jwtv5.WithIssuer(s.cfg.Issuer))
	}
	parser := jwtv5.NewParser(options...)
	tok, err := parser.ParseWithClaims(value, claims, func(*jwtv5.Token) (any, error) {
		return []byte(s.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidWSTicket, err)
	}
	if claims.Type != tokenTypeWSTicket {
		return nil, fmt.Errorf("%w: unexpected token_type %q", domain.ErrInvalidWSTicket, claims.Type)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: user id required", domain.ErrInvalidWSTicket)
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
