package app

import (
	"context"
	"errors"
	"testing"
	"time"

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
	svc, _, bus := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	r, err := svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	if !errors.Is(err, domain.ErrAlreadyAttributed) {
		t.Errorf("expected ErrAlreadyAttributed, got %v", err)
	}
	if r == nil || r.ReferrerID != "referrer-1" {
		t.Fatalf("replayed referral=%+v, want persisted attribution", r)
	}
	if got := len(bus.GotKind(event.KindReferralRegistered)); got != 2 {
		t.Fatalf("ReferralRegistered count=%d, want initial delivery plus replay", got)
	}
}

func TestAttribute_ListenerFailureCanBeRetried(t *testing.T) {
	svc, refs, bus := setupAttribute()
	bus.failKind = event.KindReferralRegistered
	bus.failCount = 1

	r, err := svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	if err == nil || r == nil {
		t.Fatalf("first attribution=(%+v, %v), want persisted referral and delivery error", r, err)
	}
	stored, findErr := refs.FindByReferee(context.Background(), "referee-1")
	if findErr != nil || stored.ReferrerID != "referrer-1" {
		t.Fatalf("persisted referral=(%+v, %v), want original attribution", stored, findErr)
	}

	r, err = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	if !errors.Is(err, domain.ErrAlreadyAttributed) {
		t.Fatalf("retry attribution error=%v, want ErrAlreadyAttributed", err)
	}
	if r.ReferrerID != "referrer-1" {
		t.Fatalf("retry referral=%+v, want original attribution", r)
	}
	if got := len(bus.GotKind(event.KindReferralRegistered)); got != 2 {
		t.Fatalf("ReferralRegistered count=%d, want one failed attempt plus retry", got)
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

func TestActivate_AlreadyActivatedReplaysEvent(t *testing.T) {
	svc, _, bus := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	_, _ = svc.ActivateReferral(context.Background(), "referee-1", 100)
	r, err := svc.ActivateReferral(context.Background(), "referee-1", 999)
	if !errors.Is(err, domain.ErrAlreadyActivated) {
		t.Fatalf("replay activation error=%v, want ErrAlreadyActivated", err)
	}
	if r.RewardCredits != 100 {
		t.Fatalf("replayed reward=%d, want stored reward 100", r.RewardCredits)
	}
	if got := len(bus.GotKind(event.KindReferralActivated)); got != 2 {
		t.Fatalf("ReferralActivated count=%d, want 2", got)
	}
}

func TestActivate_ListenerFailureCanBeRetried(t *testing.T) {
	svc, refs, bus := setupAttribute()
	_, _ = svc.AttributeReferral(context.Background(), "referee-1", "CODE-A")
	bus.failKind = event.KindReferralActivated
	bus.failCount = 1

	r, err := svc.ActivateReferral(context.Background(), "referee-1", 100)
	if err == nil || r == nil {
		t.Fatalf("first activation=(%+v, %v), want persisted referral and delivery error", r, err)
	}
	stored, findErr := refs.FindByReferee(context.Background(), "referee-1")
	if findErr != nil || stored.Status != domain.StatusActivated {
		t.Fatalf("persisted referral=(%+v, %v), want activated", stored, findErr)
	}

	r, err = svc.ActivateReferral(context.Background(), "referee-1", 999)
	if !errors.Is(err, domain.ErrAlreadyActivated) {
		t.Fatalf("retry activation error=%v, want ErrAlreadyActivated", err)
	}
	if r.RewardCredits != 100 {
		t.Fatalf("retry reward=%d, want original 100", r.RewardCredits)
	}
	if got := len(bus.GotKind(event.KindReferralActivated)); got != 2 {
		t.Fatalf("ReferralActivated count=%d, want one failed attempt plus retry", got)
	}
}

func TestActivate_NotFound(t *testing.T) {
	svc, _, _ := setupAttribute()
	_, err := svc.ActivateReferral(context.Background(), "no-such-user", 100)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestActivate_ExpiredDoesNotPublishReward(t *testing.T) {
	svc, refs, bus := setupAttribute()
	past := time.Now().UTC().Add(-time.Minute)
	refs.byRefere["referee-1"] = domain.Referral{
		ReferrerID: "referrer-1",
		RefereeID:  "referee-1",
		Status:     domain.StatusPending,
		ExpiresAt:  &past,
	}

	r, err := svc.ActivateReferral(context.Background(), "referee-1", 100)
	if !errors.Is(err, domain.ErrExpired) {
		t.Fatalf("activation error=%v, want ErrExpired", err)
	}
	if r != nil {
		t.Fatalf("service result=%+v, want nil for expired referral", r)
	}
	stored, findErr := refs.FindByReferee(context.Background(), "referee-1")
	if findErr != nil || stored.Status != domain.StatusExpired || stored.RewardCredits != 0 {
		t.Fatalf("stored referral=(%+v, %v), want expired without reward", stored, findErr)
	}
	if got := len(bus.GotKind(event.KindReferralActivated)); got != 0 {
		t.Fatalf("ReferralActivated count=%d, want 0", got)
	}
}
