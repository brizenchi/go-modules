package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/brizenchi/go-modules/modules/billing/domain"
	"github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/billing/port"
)

// WebhookService is the entry point for provider webhook deliveries.
//
// Flow:
//  1. provider.VerifyAndParseWebhook  → verify signature, parse, derive events
//  2. repo.CreateIfAbsent             → idempotency: insert event row by unique
//     (provider, provider_event_id). On duplicate,
//     skip processing.
//  3. resolver.Resolve                → fill UserID for each envelope when missing
//  4. bus.Publish                     → dispatch domain events to listeners
//  5. repo.MarkProcessed              → mark the event row as handled
//
// On any error after a successful CreateIfAbsent insert we return the error
// without marking processed. A later provider retry finds the unprocessed row
// and runs it again. Every listener must therefore use an idempotency key for
// externally visible side effects.
type WebhookService struct {
	provider port.Provider
	repo     port.BillingEventRepository
	subs     port.SubscriptionRepository
	resolver port.UserResolver
	bus      port.EventBus
}

func NewWebhookService(p port.Provider, r port.BillingEventRepository, sr port.SubscriptionRepository, ur port.UserResolver, b port.EventBus) *WebhookService {
	return &WebhookService{provider: p, repo: r, subs: sr, resolver: ur, bus: b}
}

// ProcessResult summarizes what Process did. It's primarily for logging/responses.
type ProcessResult struct {
	ProviderEventID string
	Type            string
	Duplicate       bool
}

// Process verifies a webhook payload and dispatches the resulting events.
func (s *WebhookService) Process(ctx context.Context, payload []byte, signature string) (*ProcessResult, error) {
	parsed, err := s.provider.VerifyAndParseWebhook(payload, signature)
	if err != nil {
		return nil, err
	}

	resolvedUserID := parsed.UserHint.UserID
	if resolvedUserID == "" && s.resolver != nil {
		if uid, err := s.resolver.Resolve(ctx, parsed.UserHint); err == nil {
			resolvedUserID = uid
		}
	}

	row := &domain.BillingEvent{
		UserID:          resolvedUserID,
		Provider:        s.provider.Name(),
		ProviderEventID: parsed.ProviderEventID,
		EventType:       parsed.Type,
		Payload:         json.RawMessage(parsed.RawPayload),
	}
	stored, inserted, err := s.repo.CreateIfAbsent(ctx, row)
	if err != nil {
		return nil, err
	}
	if !inserted && stored.Processed {
		slog.InfoContext(ctx, "billing: skip duplicate webhook",
			"provider", s.provider.Name(),
			"event_id", parsed.ProviderEventID,
			"type", parsed.Type,
		)
		return &ProcessResult{
			ProviderEventID: parsed.ProviderEventID,
			Type:            parsed.Type,
			Duplicate:       true,
		}, nil
	}

	for _, env := range parsed.Envelopes {
		if env.UserID == "" {
			env.UserID = resolvedUserID
		}
		if env.Provider == "" {
			env.Provider = s.provider.Name()
		}
		if env.ProviderEventID == "" {
			env.ProviderEventID = parsed.ProviderEventID
		}
		if err := s.persistSubscriptionSnapshot(ctx, env); err != nil {
			return nil, err
		}
		if err := s.bus.Publish(ctx, env); err != nil {
			return nil, fmt.Errorf("billing: publish %s: %w", env.Kind, err)
		}
	}

	if err := s.repo.MarkProcessed(ctx, s.provider.Name(), parsed.ProviderEventID); err != nil {
		return nil, err
	}

	return &ProcessResult{
		ProviderEventID: parsed.ProviderEventID,
		Type:            parsed.Type,
	}, nil
}

func (s *WebhookService) persistSubscriptionSnapshot(ctx context.Context, env event.Envelope) error {
	if s.subs == nil {
		return nil
	}
	var snapshot *domain.SubscriptionSnapshot
	switch payload := env.Payload.(type) {
	case event.SubscriptionActivated:
		snapshot = &payload.Snapshot
	case event.SubscriptionRenewed:
		snapshot = &payload.Snapshot
	case event.SubscriptionUpdated:
		snapshot = &payload.Snapshot
	case event.SubscriptionCanceling:
		snapshot = &payload.Snapshot
	case event.SubscriptionReactivated:
		snapshot = &payload.Snapshot
	case event.TrialConverted:
		snapshot = &payload.Snapshot
	case event.SubscriptionCanceled:
		if payload.Snapshot.ProviderSubscriptionID != "" {
			snapshot = &payload.Snapshot
		}
	}
	if snapshot == nil {
		return nil
	}
	if env.UserID == "" {
		return fmt.Errorf("billing: user_id required to persist %s snapshot", env.Kind)
	}
	if err := s.subs.UpsertSnapshot(ctx, env.UserID, env.Provider, *snapshot); err != nil {
		return fmt.Errorf("billing: persist %s snapshot: %w", env.Kind, err)
	}
	return nil
}

// IsSignatureError reports whether err originates from webhook signature verification.
// Useful for HTTP handlers to return 400 vs 500.
func IsSignatureError(err error) bool {
	return errors.Is(err, domain.ErrSignatureInvalid)
}
