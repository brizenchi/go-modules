// Package app contains the referral module's use cases.
package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/go-modules/modules/referral/port"
)

// CodeService manages referral code lookup + lazy creation.
type CodeService struct {
	codes     port.CodeRepository
	generator port.CodeGenerator
	maxRetry  int
}

func NewCodeService(codes port.CodeRepository, generator port.CodeGenerator) *CodeService {
	return &CodeService{codes: codes, generator: generator, maxRetry: 5}
}

// GetOrCreate returns the user's existing code, or creates one and stores it.
//
// For deterministic generators, retries are not needed (user_id is unique).
// For random generators, the loop catches collisions and regenerates.
func (s *CodeService) GetOrCreate(ctx context.Context, userID string) (*domain.Code, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrInvalidUser
	}
	existing, err := s.codes.FindByUser(ctx, userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	for attempt := 0; attempt < s.maxRetry; attempt++ {
		value := s.generator.Generate(userID)
		c := domain.Code{UserID: userID, Value: value, CreatedAt: time.Now().UTC()}
		if err := s.codes.Create(ctx, c); err == nil {
			return &c, nil
		} else if !errors.Is(err, domain.ErrCodeCollision) {
			return nil, err
		}
		// collision: try again with a new value (only useful for random generators).
	}
	return nil, domain.ErrCodeCollision
}

// Resolve looks up the owner of a code value. Returns ErrInvalidCode
// when the code does not exist.
func (s *CodeService) Resolve(ctx context.Context, value string) (*domain.Code, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, domain.ErrInvalidCode
	}
	c, err := s.codes.FindByValue(ctx, value)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCode
		}
		return nil, err
	}
	return c, nil
}
