package app

import (
	"context"
	"sync"

	"github.com/brizenchi/go-modules/referral/domain"
	"github.com/brizenchi/go-modules/referral/event"
	"github.com/brizenchi/go-modules/referral/port"
)

// --- mockCodeRepo -----------------------------------------------------

type mockCodeRepo struct {
	mu      sync.Mutex
	byUser  map[string]domain.Code
	byValue map[string]domain.Code
}

func newMockCodeRepo() *mockCodeRepo {
	return &mockCodeRepo{
		byUser:  make(map[string]domain.Code),
		byValue: make(map[string]domain.Code),
	}
}

func (r *mockCodeRepo) FindByUser(_ context.Context, userID string) (*domain.Code, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.byUser[userID]; ok {
		return &c, nil
	}
	return nil, domain.ErrNotFound
}

func (r *mockCodeRepo) FindByValue(_ context.Context, value string) (*domain.Code, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.byValue[value]; ok {
		return &c, nil
	}
	return nil, domain.ErrNotFound
}

func (r *mockCodeRepo) Create(_ context.Context, c domain.Code) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byValue[c.Value]; exists {
		return domain.ErrCodeCollision
	}
	r.byUser[c.UserID] = c
	r.byValue[c.Value] = c
	return nil
}

// --- mockReferralRepo -------------------------------------------------

type mockReferralRepo struct {
	mu       sync.Mutex
	byRefere map[string]domain.Referral
}

func newMockReferralRepo() *mockReferralRepo {
	return &mockReferralRepo{byRefere: make(map[string]domain.Referral)}
}

func (r *mockReferralRepo) FindByReferee(_ context.Context, refereeID string) (*domain.Referral, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.byRefere[refereeID]; ok {
		return &v, nil
	}
	return nil, domain.ErrNotFound
}

func (r *mockReferralRepo) Create(_ context.Context, ref domain.Referral) (*domain.Referral, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byRefere[ref.RefereeID]; exists {
		return nil, domain.ErrAlreadyAttributed
	}
	if ref.Status == "" {
		ref.Status = domain.StatusPending
	}
	r.byRefere[ref.RefereeID] = ref
	return &ref, nil
}

func (r *mockReferralRepo) Activate(_ context.Context, refereeID string, reward int) (*domain.Referral, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.byRefere[refereeID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if v.Status == domain.StatusActivated {
		return nil, domain.ErrAlreadyActivated
	}
	v.Status = domain.StatusActivated
	v.RewardCredits = reward
	r.byRefere[refereeID] = v
	return &v, nil
}

func (r *mockReferralRepo) ListByReferrer(_ context.Context, referrerID string, _, _ int) ([]domain.Referral, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Referral
	for _, v := range r.byRefere {
		if v.ReferrerID == referrerID {
			out = append(out, v)
		}
	}
	return out, len(out), nil
}

func (r *mockReferralRepo) StatsByReferrer(ctx context.Context, referrerID string) (*domain.Stats, error) {
	items, _, _ := r.ListByReferrer(ctx, referrerID, 0, 0)
	stats := &domain.Stats{}
	for _, v := range items {
		stats.TotalReferred++
		if v.Status == domain.StatusActivated {
			stats.Activated++
			stats.TotalRewardCredits += v.RewardCredits
		} else {
			stats.Pending++
		}
	}
	return stats, nil
}

// --- mockGenerator ----------------------------------------------------

type mockGenerator struct {
	values []string
	idx    int
}

func (g *mockGenerator) Generate(_ string) string {
	if g.idx >= len(g.values) {
		return "FALLBACK"
	}
	v := g.values[g.idx]
	g.idx++
	return v
}

// --- mockBus ----------------------------------------------------------

type mockBus struct {
	mu        sync.Mutex
	published []event.Envelope
}

func (b *mockBus) Subscribe(event.Kind, port.Listener) {}

func (b *mockBus) Publish(_ context.Context, env event.Envelope) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, env)
}

func (b *mockBus) GotKind(k event.Kind) []event.Envelope {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []event.Envelope
	for _, e := range b.published {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}
