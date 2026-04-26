package app

import (
	"context"
	"errors"
	"testing"

	"github.com/brizenchi/go-modules/referral/domain"
)

func TestCodeService_GetOrCreate_GeneratesOnce(t *testing.T) {
	repo := newMockCodeRepo()
	gen := &mockGenerator{values: []string{"V1"}}
	svc := NewCodeService(repo, gen)

	a, err := svc.GetOrCreate(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if a.Value != "V1" {
		t.Errorf("first call value = %q", a.Value)
	}

	// Second call returns the persisted one — generator is NOT called again.
	b, err := svc.GetOrCreate(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if b.Value != "V1" {
		t.Errorf("second call value = %q", b.Value)
	}
	if gen.idx != 1 {
		t.Errorf("generator called %d times, want 1", gen.idx)
	}
}

func TestCodeService_GetOrCreate_RetriesOnCollision(t *testing.T) {
	repo := newMockCodeRepo()
	// Pre-fill so the first generated value collides.
	_ = repo.Create(context.Background(), domain.Code{UserID: "other", Value: "V1"})
	gen := &mockGenerator{values: []string{"V1", "V2"}}
	svc := NewCodeService(repo, gen)

	c, err := svc.GetOrCreate(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Value != "V2" {
		t.Errorf("got %q, want V2 after collision", c.Value)
	}
}

func TestCodeService_GetOrCreate_RejectsEmptyUser(t *testing.T) {
	svc := NewCodeService(newMockCodeRepo(), &mockGenerator{values: []string{"X"}})
	_, err := svc.GetOrCreate(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidUser) {
		t.Errorf("expected ErrInvalidUser, got %v", err)
	}
}

func TestCodeService_Resolve_NotFound(t *testing.T) {
	svc := NewCodeService(newMockCodeRepo(), &mockGenerator{values: []string{"X"}})
	_, err := svc.Resolve(context.Background(), "missing")
	if !errors.Is(err, domain.ErrInvalidCode) {
		t.Errorf("expected ErrInvalidCode, got %v", err)
	}
}

func TestCodeService_Resolve_Found(t *testing.T) {
	repo := newMockCodeRepo()
	_ = repo.Create(context.Background(), domain.Code{UserID: "u1", Value: "ABC"})
	svc := NewCodeService(repo, &mockGenerator{values: []string{"X"}})
	c, err := svc.Resolve(context.Background(), "ABC")
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != "u1" {
		t.Errorf("user_id = %q", c.UserID)
	}
}
