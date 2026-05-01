// Package app contains the auth module's use cases.
//
// Layering rules:
//   - app/ depends on domain + port + event only.
//   - app/ never depends on adapter/ (adapters depend on port).
//   - app/ never imports HTTP types.
package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

// LoginService coordinates email-code login: SendCode + VerifyCode.
//
// VerifyCode finalizes the login by resolving (or creating) the user
// in UserStore, assigning a Role, issuing a Token, and publishing
// UserSignedUp / UserLoggedIn.
type LoginService struct {
	issuer   port.EmailCodeIssuer
	verifier port.EmailCodeVerifier
	users    port.UserStore
	roles    port.RoleResolver
	signer   port.TokenSigner
	bus      port.EventBus
	tokenTTL time.Duration
}

// LoginDeps gathers the dependencies LoginService needs.
type LoginDeps struct {
	Issuer   port.EmailCodeIssuer
	Verifier port.EmailCodeVerifier
	Users    port.UserStore
	Roles    port.RoleResolver
	Signer   port.TokenSigner
	Bus      port.EventBus
	TokenTTL time.Duration
}

func NewLoginService(d LoginDeps) *LoginService {
	if d.TokenTTL == 0 {
		d.TokenTTL = 7 * 24 * time.Hour
	}
	return &LoginService{
		issuer:   d.Issuer,
		verifier: d.Verifier,
		users:    d.Users,
		roles:    d.Roles,
		signer:   d.Signer,
		bus:      d.Bus,
		tokenTTL: d.TokenTTL,
	}
}

// SendCode issues and dispatches a verification code.
func (s *LoginService) SendCode(ctx context.Context, email string) (*domain.CodeIssueResult, error) {
	if s.issuer == nil {
		return nil, domain.ErrProviderUnavailable
	}
	return s.issuer.Issue(ctx, email)
}

// VerifyResult is what VerifyCode (and OAuthService.Finalize) return.
type VerifyResult struct {
	Token    *domain.Token
	Identity domain.Identity
}

// VerifyCode finishes the email-code flow.
//
// On success: invalidates the code, find/creates the user, assigns role,
// mints a token, marks login, publishes events.
func (s *LoginService) VerifyCode(ctx context.Context, email, code string) (*VerifyResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if err := s.verifier.Verify(ctx, email, code); err != nil {
		return nil, err
	}

	id, err := s.users.FindOrCreateByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if id.Email == "" {
		id.Email = email
	}
	if s.roles != nil {
		role, err := s.roles.Resolve(ctx, *id)
		if err == nil && role != "" {
			id.Role = role
		}
	}
	tok, err := s.signer.Issue(*id, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	if err := s.users.MarkLogin(ctx, id.UserID); err != nil {
		// non-fatal — token is already minted.
		_ = err
	}

	now := time.Now().UTC()
	if id.IsNew && s.bus != nil {
		s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindUserSignedUp,
			UserID:     id.UserID,
			OccurredAt: now,
			Payload:    event.UserSignedUp{Identity: *id},
		})
	}
	if s.bus != nil {
		s.bus.Publish(ctx, event.Envelope{
			Kind:       event.KindUserLoggedIn,
			UserID:     id.UserID,
			OccurredAt: now,
			Payload:    event.UserLoggedIn{Identity: *id, Provider: domain.ProviderEmail},
		})
	}
	return &VerifyResult{Token: tok, Identity: *id}, nil
}

// RandomToken is a tiny helper (used by OAuthService for exchange codes too).
func RandomToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = 32
	}
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
