package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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
//  4. subs.UpsertSnapshot             → update the canonical read model only
//     when this snapshot is not stale
//  5. bus.Publish                     → dispatch every unprocessed domain event
//     to listeners, even when its snapshot was stale
//  6. repo.MarkProcessed              → mark the event row as handled
//
// On any error after a successful CreateIfAbsent insert we return the error
// without marking processed. A later provider retry finds the unprocessed row
// and runs it again. Every listener must therefore use an idempotency key for
// externally visible side effects. Stateful listeners must also use OccurredAt
// or query the canonical subscription because provider delivery order is not
// guaranteed.
type WebhookService struct {
	provider     port.Provider
	repo         port.BillingEventRepository
	subs         port.SubscriptionRepository
	resolver     port.UserResolver
	bus          port.EventBus
	reservations port.CheckoutReservationRepository
}

func NewWebhookService(p port.Provider, r port.BillingEventRepository, sr port.SubscriptionRepository, ur port.UserResolver, b port.EventBus, reservationRepos ...port.CheckoutReservationRepository) *WebhookService {
	var reservations port.CheckoutReservationRepository
	if len(reservationRepos) > 0 {
		reservations = reservationRepos[0]
	}
	return &WebhookService{provider: p, repo: r, subs: sr, resolver: ur, bus: b, reservations: reservations}
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

	if parsed.ReleaseCheckoutReservation {
		if s.reservations == nil {
			return nil, fmt.Errorf("billing: checkout reservation repository required to release terminal session")
		}
		reservationID := strings.TrimSpace(parsed.CheckoutReservationID)
		sessionID := strings.TrimSpace(parsed.CheckoutSessionID)
		if reservationID == "" && sessionID == "" {
			return nil, fmt.Errorf("billing: terminal checkout event missing reservation and session ids")
		}
		var releaseErr error
		if reservationID != "" {
			_, releaseErr = s.reservations.ReleaseCheckoutReservationByReservationID(ctx, s.provider.Name(), reservationID)
		} else {
			_, releaseErr = s.reservations.ReleaseCheckoutReservation(ctx, s.provider.Name(), sessionID)
		}
		if releaseErr != nil {
			return nil, fmt.Errorf("billing: release terminal checkout reservation: %w", releaseErr)
		}
	} else if parsed.CheckoutSessionID != "" && parsed.CheckoutSubscriptionID != "" && s.reservations != nil {
		if err := s.reservations.LinkCheckoutSubscription(ctx, s.provider.Name(), parsed.CheckoutSessionID, parsed.CheckoutSubscriptionID); err != nil {
			return nil, fmt.Errorf("billing: link checkout reservation subscription: %w", err)
		}
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
		snapshotApplied, err := s.persistSubscriptionSnapshot(ctx, env)
		if err != nil {
			return nil, err
		}
		if !snapshotApplied {
			slog.InfoContext(ctx, "billing: ignore stale subscription snapshot",
				"provider", env.Provider,
				"event_id", env.ProviderEventID,
				"kind", env.Kind,
				"user_id", env.UserID,
			)
		}
		if snapshotApplied && s.reservations != nil {
			if subID := terminalSubscriptionID(env); subID != "" {
				if _, err := s.reservations.ReleaseCheckoutReservationBySubscription(ctx, env.Provider, subID); err != nil {
					return nil, fmt.Errorf("billing: release terminal checkout reservation: %w", err)
				}
			}
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

func terminalSubscriptionID(env event.Envelope) string {
	var snapshot domain.SubscriptionSnapshot
	switch payload := env.Payload.(type) {
	case event.SubscriptionCanceled:
		snapshot = payload.Snapshot
		if snapshot.ProviderSubscriptionID == "" {
			snapshot.ProviderSubscriptionID = payload.ProviderSubscriptionID
		}
	case event.SubscriptionUpdated:
		snapshot = payload.Snapshot
	default:
		return ""
	}
	if snapshot.Status != domain.StatusCanceled && snapshot.Status != domain.StatusIncompleteExpired {
		return ""
	}
	return strings.TrimSpace(snapshot.ProviderSubscriptionID)
}

func (s *WebhookService) persistSubscriptionSnapshot(ctx context.Context, env event.Envelope) (bool, error) {
	if s.subs == nil {
		return true, nil
	}
	var snapshot *domain.SubscriptionSnapshot
	switch payload := env.Payload.(type) {
	case event.SubscriptionActivated:
		snapshot = &payload.Snapshot
	case event.SubscriptionRenewed:
		// Invoice payloads are payment facts, not authoritative subscription
		// snapshots (they do not carry cancellation flags and can omit fields).
		// customer.subscription.* events own the canonical read model.
		return true, nil
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
		return true, nil
	}
	if env.UserID == "" {
		return false, fmt.Errorf("billing: user_id required to persist %s snapshot", env.Kind)
	}
	if env.OccurredAt.IsZero() {
		return false, fmt.Errorf("billing: occurred_at required to persist %s snapshot", env.Kind)
	}
	applied, err := s.subs.UpsertSnapshot(ctx, env.UserID, env.Provider, *snapshot, env.OccurredAt, env.ProviderEventID)
	if err != nil {
		return false, fmt.Errorf("billing: persist %s snapshot: %w", env.Kind, err)
	}
	return applied, nil
}

// IsSignatureError reports whether err originates from webhook signature verification.
// Useful for HTTP handlers to return 400 vs 500.
func IsSignatureError(err error) bool {
	return errors.Is(err, domain.ErrSignatureInvalid)
}
