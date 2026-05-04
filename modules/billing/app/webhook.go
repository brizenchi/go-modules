package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/brizenchi/go-modules/modules/billing/domain"
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
// On any error after a successful CreateIfAbsent insert we return the
// error to the caller without marking processed. The next delivery will
// be detected as a duplicate row but will NOT be reprocessed (Processed
// flag is false). For now we rely on Stripe's retry; for stricter
// at-least-once semantics, swap in a worker that reads unprocessed rows.
type WebhookService struct {
	provider port.Provider
	repo     port.BillingEventRepository
	resolver port.UserResolver
	bus      port.EventBus
}

func NewWebhookService(p port.Provider, r port.BillingEventRepository, ur port.UserResolver, b port.EventBus) *WebhookService {
	return &WebhookService{provider: p, repo: r, resolver: ur, bus: b}
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
		s.bus.Publish(ctx, env)
	}

	if err := s.repo.MarkProcessed(ctx, s.provider.Name(), parsed.ProviderEventID); err != nil {
		return nil, err
	}

	return &ProcessResult{
		ProviderEventID: parsed.ProviderEventID,
		Type:            parsed.Type,
	}, nil
}

// IsSignatureError reports whether err originates from webhook signature verification.
// Useful for HTTP handlers to return 400 vs 500.
func IsSignatureError(err error) bool {
	return errors.Is(err, domain.ErrSignatureInvalid)
}
