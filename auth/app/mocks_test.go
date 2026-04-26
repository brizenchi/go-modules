package app

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/event"
	"github.com/brizenchi/go-modules/auth/port"
)

// --- mockUserStore ----------------------------------------------------

type mockUserStore struct {
	mu       sync.Mutex
	byEmail  map[string]*domain.Identity
	byID     map[string]*domain.Identity
	logins   map[string]int
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		byEmail: make(map[string]*domain.Identity),
		byID:    make(map[string]*domain.Identity),
		logins:  make(map[string]int),
	}
}

func (m *mockUserStore) seed(id domain.Identity) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := id
	m.byEmail[id.Email] = &cp
	m.byID[id.UserID] = &cp
}

func (m *mockUserStore) FindByEmail(_ context.Context, email string) (*domain.Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byEmail[email]; ok {
		out := *id
		out.IsNew = false
		return &out, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserStore) FindOrCreateByEmail(_ context.Context, email string) (*domain.Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byEmail[email]; ok {
		out := *id
		out.IsNew = false
		return &out, nil
	}
	id := &domain.Identity{UserID: "u-" + email, Email: email, IsNew: true}
	m.byEmail[email] = id
	m.byID[id.UserID] = id
	return id, nil
}

func (m *mockUserStore) FindOrCreateFromOAuth(_ context.Context, p domain.OAuthProfile) (*domain.Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byEmail[p.Email]; ok {
		out := *id
		out.IsNew = false
		return &out, nil
	}
	id := &domain.Identity{UserID: "u-" + p.Email, Email: p.Email, Provider: p.Provider, Subject: p.Subject, IsNew: true}
	m.byEmail[p.Email] = id
	m.byID[id.UserID] = id
	return id, nil
}

func (m *mockUserStore) FindByID(_ context.Context, userID string) (*domain.Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byID[userID]; ok {
		out := *id
		out.IsNew = false
		return &out, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserStore) MarkLogin(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logins[userID]++
	return nil
}

// --- mockSigner -------------------------------------------------------

type mockSigner struct{}

func (mockSigner) Issue(id domain.Identity, ttl time.Duration) (*domain.Token, error) {
	return &domain.Token{Value: "tok-" + id.UserID, ExpiresAt: time.Now().Add(ttl)}, nil
}
func (mockSigner) Parse(value string) (*domain.Identity, error) {
	if value == "" {
		return nil, domain.ErrInvalidToken
	}
	return &domain.Identity{UserID: value[len("tok-"):]}, nil
}

// --- mockBus ----------------------------------------------------------

type mockBus struct {
	mu        sync.Mutex
	published []event.Envelope
}

func (b *mockBus) Subscribe(kind event.Kind, fn port.Listener) {}

func (b *mockBus) Publish(_ context.Context, env event.Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, env)
}

func (b *mockBus) Got(kind event.Kind) []event.Envelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []event.Envelope
	for _, e := range b.published {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// --- mockEmailCode ----------------------------------------------------

type mockIssuer struct {
	calls int
	err   error
}

func (m *mockIssuer) Issue(_ context.Context, email string) (*domain.CodeIssueResult, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	return &domain.CodeIssueResult{Email: email, ExpiresAt: time.Now().Add(time.Minute)}, nil
}

type mockVerifier struct{ err error }

func (m *mockVerifier) Verify(_ context.Context, email, code string) error {
	if m.err != nil {
		return m.err
	}
	if code == "good" {
		return nil
	}
	return domain.ErrInvalidCode
}

// --- mockProvider (OAuth) ---------------------------------------------

type mockProvider struct {
	name        domain.Provider
	exchangeErr error
	profile     *domain.OAuthProfile
	stateOK     bool
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		name:    domain.ProviderGoogle,
		stateOK: true,
		profile: &domain.OAuthProfile{
			Provider: domain.ProviderGoogle,
			Subject:  "sub-1",
			Email:    "u@example.com",
		},
	}
}

func (p *mockProvider) Name() domain.Provider                     { return p.name }
func (p *mockProvider) AuthorizeURL(state string, _ url.Values) (string, error) {
	return "https://provider/authz?state=" + state, nil
}
func (p *mockProvider) Exchange(_ context.Context, _ url.Values) (*domain.OAuthProfile, error) {
	if p.exchangeErr != nil {
		return nil, p.exchangeErr
	}
	return p.profile, nil
}
func (p *mockProvider) IssueState() (string, error) { return "state-1", nil }
func (p *mockProvider) VerifyState(s string) error {
	if !p.stateOK {
		return domain.ErrInvalidState
	}
	return nil
}

// --- mockExchangeStore ------------------------------------------------

type mockExchange struct {
	mu sync.Mutex
	m  map[string]domain.ExchangeCode
}

func newMockExchange() *mockExchange { return &mockExchange{m: make(map[string]domain.ExchangeCode)} }

func (s *mockExchange) Save(_ context.Context, c domain.ExchangeCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[c.Code] = c
	return nil
}
func (s *mockExchange) Consume(_ context.Context, code string) (*domain.ExchangeCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.m[code]
	if !ok {
		return nil, domain.ErrInvalidExchange
	}
	delete(s.m, code)
	if time.Now().After(c.ExpiresAt) {
		return nil, domain.ErrInvalidExchange
	}
	return &c, nil
}

// avoid unused import if helpers change
var _ = errors.New
