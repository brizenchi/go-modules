package app

import (
	"context"

	"github.com/brizenchi/go-modules/modules/referral/domain"
	"github.com/brizenchi/go-modules/modules/referral/port"
)

// QueryService exposes read-only queries for the referral dashboard.
type QueryService struct {
	referrals port.ReferralRepository
}

func NewQueryService(referrals port.ReferralRepository) *QueryService {
	return &QueryService{referrals: referrals}
}

func (s *QueryService) ListByReferrer(ctx context.Context, referrerID string, page, limit int) ([]domain.Referral, int, error) {
	return s.referrals.ListByReferrer(ctx, referrerID, page, limit)
}

func (s *QueryService) Stats(ctx context.Context, referrerID string) (*domain.Stats, error) {
	return s.referrals.StatsByReferrer(ctx, referrerID)
}
