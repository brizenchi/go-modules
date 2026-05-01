// Package emailcode implements the passwordless email-code flow:
// EmailCodeIssuer + EmailCodeVerifier.
//
// Issuer: generates a 6-digit numeric code, persists it via a
// CodeRateLimitStore, and sends it through modules/email's SendService
// (or any compatible Mailer).
//
// Verifier: looks up the persisted code, increments attempts on
// mismatch, deletes on success or after MaxAttempts.
package emailcode

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

// Mailer is the contract this adapter needs from the email module —
// just send a templated message keyed by a provider template ref.
//
// modules/email's *email.Module satisfies this implicitly because its
// SendProviderTemplate signature matches.
type Mailer interface {
	SendProviderTemplate(ctx context.Context, templateRef string, to []EmailAddress, vars map[string]any) error
}

// EmailAddress is a tiny re-statement of the email module's domain.Address
// type, decoupling this adapter from modules/email's import path.
type EmailAddress struct {
	Name  string
	Email string
}

// Config bundles the rate-limit + delivery options.
type Config struct {
	CodeLength   int           // default 6
	TTL          time.Duration // default 10 minutes
	MinResendGap time.Duration // default 60 seconds (per-email throttle)
	DailyCap     int           // default 10 codes per email per day
	MaxAttempts  int           // default 5 failed verifications before invalidating
	TemplateRef  string        // provider-side template id (e.g. Brevo template id)
	Debug        bool          // when true, returned CodeIssueResult.DebugCode contains the code
}

func (c Config) withDefaults() Config {
	if c.CodeLength == 0 {
		c.CodeLength = 6
	}
	if c.TTL == 0 {
		c.TTL = 10 * time.Minute
	}
	if c.MinResendGap == 0 {
		c.MinResendGap = 60 * time.Second
	}
	if c.DailyCap == 0 {
		c.DailyCap = 10
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 5
	}
	return c
}

// Issuer implements port.EmailCodeIssuer.
type Issuer struct {
	cfg    Config
	store  port.CodeRateLimitStore
	mailer Mailer
}

func NewIssuer(cfg Config, store port.CodeRateLimitStore, mailer Mailer) *Issuer {
	return &Issuer{cfg: cfg.withDefaults(), store: store, mailer: mailer}
}

// Issue produces and sends a fresh code.
//
// Returns ErrInvalidEmail when email is empty, ErrCodeRateLimited when
// the per-minute or per-day cap is hit.
func (i *Issuer) Issue(ctx context.Context, email string) (*domain.CodeIssueResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, domain.ErrInvalidEmail
	}

	now := time.Now().UTC()
	dayBucket := now.Format("2006-01-02")

	// Per-day cap (incremented optimistically; we accept that a failed
	// send still consumes one slot — keeps the implementation atomic).
	count, err := i.store.IncrDailyCount(ctx, email, dayBucket)
	if err != nil {
		return nil, err
	}
	if count > i.cfg.DailyCap {
		return nil, domain.ErrCodeRateLimited
	}

	// Per-minute throttle.
	_, lastSent, err := i.store.LoadCode(ctx, email)
	if err != nil {
		return nil, err
	}
	if !lastSent.IsZero() && now.Sub(lastSent) < i.cfg.MinResendGap {
		return nil, domain.ErrCodeRateLimited
	}

	code, err := generateNumericCode(i.cfg.CodeLength)
	if err != nil {
		return nil, err
	}
	expiresAt := now.Add(i.cfg.TTL)
	if err := i.store.SaveCode(ctx, email, code, expiresAt, now); err != nil {
		return nil, err
	}

	if i.cfg.TemplateRef != "" && i.mailer != nil {
		if err := i.mailer.SendProviderTemplate(ctx, i.cfg.TemplateRef,
			[]EmailAddress{{Email: email}},
			map[string]any{
				"code": code,
				"year": fmt.Sprintf("%d", now.Year()),
			},
		); err != nil {
			// In debug mode the code is returned in the response anyway,
			// so a delivery failure (provider down, IP not allow-listed,
			// missing credentials, ...) shouldn't fail the call. We log
			// the underlying error and continue.
			if !i.cfg.Debug {
				return nil, fmt.Errorf("emailcode: deliver: %w", err)
			}
			slog.WarnContext(ctx, "emailcode: delivery failed in debug mode, returning code anyway",
				"email", email,
				"error", err,
			)
		}
	}

	res := &domain.CodeIssueResult{Email: email, ExpiresAt: expiresAt}
	if i.cfg.Debug {
		res.DebugCode = code
	}
	return res, nil
}

func generateNumericCode(length int) (string, error) {
	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0') + byte(n.Int64())
	}
	return string(digits), nil
}

var _ port.EmailCodeIssuer = (*Issuer)(nil)
