package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/event"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

// OAuthService coordinates an OAuth login round-trip.
//
//  1. StartOAuth(name) → stores a one-time browser-bound flow and returns the
//     provider URL plus an opaque binding for an HttpOnly cookie
//  2. provider redirects back to the callback URL with ?code=...&state=...
//  3. OAuthCallback(name, query, binding) → verifies and atomically consumes
//     the browser-bound state, fetches the profile,
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
	flowStore     port.OAuthFlowStore
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
	FlowStore     port.OAuthFlowStore
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
		flowStore:     d.FlowStore,
		bus:           d.Bus,
		tokenTTL:      d.TokenTTL,
		exchangeTTL:   d.ExchangeTTL,
	}
}

const defaultOAuthFlowTTL = 20 * time.Minute

// OAuthStartResult contains the provider redirect plus the browser binding
// that the HTTP adapter must place in an HttpOnly cookie. BrowserBinding must
// never be returned in JSON or embedded in the provider redirect URL.
type OAuthStartResult struct {
	AuthorizeURL   string
	FlowID         string
	BrowserBinding string
	ExpiresAt      time.Time
}

// StartOAuth creates a fresh, single-use browser-bound OAuth flow.
func (s *OAuthService) StartOAuth(ctx context.Context, providerName, verifierChallenge string) (*OAuthStartResult, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return nil, domain.ErrProviderUnavailable
	}
	if s.flowStore == nil {
		return nil, domain.ErrOAuthFlowUnavailable
	}
	verifierChallenge = strings.TrimSpace(verifierChallenge)
	if !validOAuthChallenge(verifierChallenge) {
		return nil, domain.ErrInvalidState
	}
	state, err := p.IssueState()
	if err != nil {
		return nil, err
	}
	authorizeURL, err := p.AuthorizeURL(state, nil)
	if err != nil {
		return nil, err
	}
	binding, err := RandomToken(32)
	if err != nil {
		return nil, err
	}
	ttl := defaultOAuthFlowTTL
	if providerWithTTL, ok := p.(port.OAuthStateLifetime); ok && providerWithTTL.OAuthStateTTL() > 0 {
		ttl = providerWithTTL.OAuthStateTTL()
	}
	expiresAt := time.Now().UTC().Add(ttl)
	flowID := OAuthFlowID(state)
	if err := s.flowStore.SaveOAuthFlow(ctx, domain.OAuthFlow{
		StateHash:         flowID,
		Provider:          p.Name(),
		BindingHash:       oauthBindingHash(binding),
		VerifierChallenge: verifierChallenge,
		ExpiresAt:         expiresAt,
	}); err != nil {
		return nil, err
	}
	return &OAuthStartResult{
		AuthorizeURL:   authorizeURL,
		FlowID:         flowID,
		BrowserBinding: binding,
		ExpiresAt:      expiresAt,
	}, nil
}

// OAuthFlowID is a stable, non-secret identifier used for the flow row and
// its uniquely named browser cookie. The raw signed state is not persisted.
func OAuthFlowID(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

func oauthBindingHash(binding string) string {
	sum := sha256.Sum256([]byte(binding))
	return hex.EncodeToString(sum[:])
}

func validOAuthChallenge(challenge string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	return err == nil && len(decoded) == sha256.Size && base64.RawURLEncoding.EncodeToString(decoded) == challenge
}

func oauthVerifierChallenge(verifier string) (string, error) {
	verifier = strings.TrimSpace(verifier)
	decoded, err := base64.RawURLEncoding.DecodeString(verifier)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != verifier {
		return "", domain.ErrInvalidExchange
	}
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// OAuthCallbackResult is returned to the callback handler.
type OAuthCallbackResult struct {
	ExchangeCode string
	FlowID       string
	Identity     domain.Identity
}

// OAuthCallback verifies and atomically consumes the browser-bound provider
// state before exchanging the authorization code and staging an exchange code.
func (s *OAuthService) OAuthCallback(ctx context.Context, providerName string, query url.Values, browserBinding string) (*OAuthCallbackResult, error) {
	p, ok := s.providers[providerName]
	if !ok {
		return nil, domain.ErrProviderUnavailable
	}
	if s.flowStore == nil {
		return nil, domain.ErrOAuthFlowUnavailable
	}
	state := query.Get("state")
	if err := p.VerifyState(state); err != nil {
		return nil, err
	}
	if strings.TrimSpace(browserBinding) == "" {
		return nil, domain.ErrInvalidState
	}
	flow, err := s.flowStore.ConsumeOAuthFlow(
		ctx,
		p.Name(),
		OAuthFlowID(state),
		oauthBindingHash(browserBinding),
	)
	if err != nil {
		return nil, err
	}
	profile, err := p.Exchange(ctx, query)
	if err != nil {
		// Preserve only the browser-held verifier challenge so the HTTP adapter
		// can tell the initiating frontend which parallel flow to discard. The
		// provider state and HttpOnly browser binding remain secret and consumed.
		return &OAuthCallbackResult{FlowID: flow.VerifierChallenge}, err
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
		Code:        code,
		UserID:      id.UserID,
		Provider:    profile.Provider,
		IsNew:       id.IsNew,
		BindingHash: flow.VerifierChallenge,
		ExpiresAt:   time.Now().UTC().Add(s.exchangeTTL),
	}
	if err := s.exchangeStore.Save(ctx, exch); err != nil {
		return nil, err
	}
	return &OAuthCallbackResult{
		ExchangeCode: code,
		FlowID:       flow.VerifierChallenge,
		Identity:     *id,
	}, nil
}

// ExchangeToken consumes an exchange code and finalizes the login.
func (s *OAuthService) ExchangeToken(ctx context.Context, code, browserVerifier string) (*VerifyResult, error) {
	challenge, err := oauthVerifierChallenge(browserVerifier)
	if err != nil {
		return nil, err
	}
	rec, err := s.exchangeStore.Consume(ctx, strings.TrimSpace(code), challenge)
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
