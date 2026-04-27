package app

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/event"
	"github.com/brizenchi/go-modules/auth/port"
)

// OAuthService coordinates an OAuth login round-trip.
//
//  1. StartOAuth(name) → returns a redirect URL the browser follows
//  2. provider redirects back to the callback URL with ?code=...&state=...
//  3. OAuthCallback(name, query) → verifies state, fetches profile,
//     creates/links the user, mints an exchange code, returns it
//  4. ExchangeToken(code) → consumes the exchange code, mints a token,
//     publishes events. This is what the SPA calls after the callback
//     redirects the browser back to the frontend.
type OAuthService struct {
	providers     map[string]port.IdentityProvider
	users         port.UserStore
	roles         port.RoleResolver
	signer        port.TokenSigner
	exchangeStore port.ExchangeCodeStore
	bus           port.EventBus
	tokenTTL      time.Duration
	exchangeTTL   time.Duration
}

type OAuthDeps struct {
	Providers     map[string]port.IdentityProvider
	Users         port.UserStore
	Roles         port.RoleResolver
	Signer        port.TokenSigner
	ExchangeStore port.ExchangeCodeStore
	Bus           port.EventBus
	TokenTTL      time.Duration
	ExchangeTTL   time.Duration
}

func NewOAuthService(d OAuthDeps) *OAuthService {
	if d.TokenTTL == 0 {
		d.TokenTTL = 7 * 24 * time.Hour
	}
	if d.ExchangeTTL == 0 {
		d.ExchangeTTL = 2 * time.Minute
	}
	return &OAuthService{
		providers:     d.Providers,
		users:         d.Users,
		roles:         d.Roles,
		signer:        d.Signer,
		exchangeStore: d.ExchangeStore,
		bus:           d.Bus,
		tokenTTL:      d.TokenTTL,
		exchangeTTL:   d.ExchangeTTL,
	}
}

// StartOAuth returns the provider's authorize URL.
func (s *OAuthService) StartOAuth(ctx context.Context, providerName string) (string, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return "", domain.ErrProviderUnavailable
	}
	state, err := p.IssueState()
	if err != nil {
		return "", err
	}
	return p.AuthorizeURL(state, nil)
}

// OAuthCallbackResult is returned to the callback handler.
type OAuthCallbackResult struct {
	ExchangeCode string
	Identity     domain.Identity
}

// OAuthCallback verifies the provider callback and stages an exchange code.
func (s *OAuthService) OAuthCallback(ctx context.Context, providerName string, query url.Values) (*OAuthCallbackResult, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return nil, domain.ErrProviderUnavailable
	}
	profile, err := p.Exchange(ctx, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(profile.Email) == "" {
		return nil, domain.ErrInvalidEmail
	}
	id, err := s.users.FindOrCreateFromOAuth(ctx, *profile)
	if err != nil {
		return nil, err
	}
	if id.Email == "" {
		id.Email = profile.Email
	}

	code, err := RandomToken(32)
	if err != nil {
		return nil, err
	}
	exch := domain.ExchangeCode{
		Code:      code,
		UserID:    id.UserID,
		Provider:  profile.Provider,
		IsNew:     id.IsNew,
		ExpiresAt: time.Now().UTC().Add(s.exchangeTTL),
	}
	if err := s.exchangeStore.Save(ctx, exch); err != nil {
		return nil, err
	}
	return &OAuthCallbackResult{ExchangeCode: code, Identity: *id}, nil
}

// ExchangeToken consumes an exchange code and finalizes the login.
func (s *OAuthService) ExchangeToken(ctx context.Context, code string) (*VerifyResult, error) {
	rec, err := s.exchangeStore.Consume(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	id, err := s.users.FindByID(ctx, rec.UserID)
	if err != nil {
		return nil, err
	}
	id.IsNew = rec.IsNew
	if s.roles != nil {
		if role, err := s.roles.Resolve(ctx, *id); err == nil && role != "" {
			id.Role = role
		}
	}
	tok, err := s.signer.Issue(*id, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	if err := s.users.MarkLogin(ctx, id.UserID); err != nil {
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
			Payload:    event.UserLoggedIn{Identity: *id, Provider: rec.Provider},
		})
	}
	return &VerifyResult{Token: tok, Identity: *id}, nil
}
