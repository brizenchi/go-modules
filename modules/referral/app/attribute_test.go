package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/go-modules/modules/referral/event"
)

func setupAttribute() (*AttributeService, *mockReferralRepo, *mockBus) {
	codes := newMockCodeRepo()
	_ = codes.Create(context.Background(), domain.Code{UserID: "referrer-1", Value: "CODE-A"})
	codeSvc := NewCodeService(codes, &mockGenerator{})
	refs := newMockReferralRepo()
	bus := &mockBus{}
	return NewAttributeService(AttributeDeps{Codes: codeSvc, Referrals: refs, Bus: bus}), refs, bus
}

func TestAttribute_Success(t *testing.T) {
	svc, refs, bus := setupAttribute()
	r, err := svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	if err != nil {
		t.Fatal(err)
	}
	if r.ReferrerID != "referrer-1" || r.RefereeID != "referee-1" {
		t.Errorf("got %+v", r)
	}
	if r.Status != domain.StatusPending {
		t.Errorf("status = %s, want pending", r.Status)
	}
	if got := len(refs.byRefere); got != 1 {
		t.Errorf("repo size = %d, want 1", got)
	}
	if got := len(bus.GotKind(event.KindReferralRegistered)); got != 1 {
		t.Errorf("ReferralRegistered count = %d, want 1", got)
	}
}

func TestAttribute_RejectsSelfReferral(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, err := svc.AttributeReferral(context.Background(), "referrer-1", "CODE-A")
	if !errors.Is(err, domain.ErrSelfReferral) {
		t.Errorf("expected ErrSelfReferral, got %v", err)
	}
}

func TestAttribute_RejectsInvalidCode(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, err := svc.AttributeReferral(context.Background(), "referee-1", "BOGUS")
	if !errors.Is(err, domain.ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode, got %v", err)
	}
}

func TestAttribute_RejectsAlreadyAttributed(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	_, err := svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	if !errors.Is(err, domain.ErrAlreadyAttributed) {
		t.Errorf("expected ErrAlreadyAttributed, got %v", err)
	}
}

func TestActivate_TransitionsAndPublishes(t *testing.T) {
	svc, _, bus := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")

	r, err := svc.ActivateReferral(context.Background(), "referee-1", 100)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != domain.StatusActivated {
		t.Errorf("status = %s", r.Status)
	}
	if r.RewardCredits != 100 {
		t.Errorf("reward = %d", r.RewardCredits)
	}
	if got := len(bus.GotKind(event.KindReferralActivated)); got != 1 {
		t.Errorf("ReferralActivated count = %d, want 1", got)
	}
}

func TestActivate_AlreadyActivated(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	_, _ = svc.ActivateReferral(context.Background(), "referee-1", 100)
	_, err := svc.ActivateReferral(context.Background(), "referee-1", 100)
	if !errors.Is(err, domain.ErrAlreadyActivated) {
		t.Errorf("expected ErrAlreadyActivated, got %v", err)
	}
}

func TestActivate_NotFound(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, err := svc.ActivateReferral(context.Background(), "no-such-user", 100)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
