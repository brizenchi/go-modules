package app

import (
	"context"
	"time"

	"github.com/brizenchi/go-modules/auth/domain"
	"github.com/brizenchi/go-modules/auth/port"
)

// SessionService handles refreshing tokens and issuing WebSocket tickets.
type SessionService struct {
	users        port.UserStore
	roles        port.RoleResolver
	signer       port.TokenSigner
	ticketSigner port.WSTicketSigner
	tokenTTL     time.Duration
	defaultWSTTL time.Duration
}

type SessionDeps struct {
	Users        port.UserStore
	Roles        port.RoleResolver
	Signer       port.TokenSigner
	TicketSigner port.WSTicketSigner
	TokenTTL     time.Duration
	WSTicketTTL  time.Duration
}

func NewSessionService(d SessionDeps) *SessionService {
	if d.TokenTTL == 0 {
		d.TokenTTL = 7 * 24 * time.Hour
	}
	if d.WSTicketTTL == 0 {
		d.WSTicketTTL = 5 * time.Minute
	}
	return &SessionService{
		users:        d.Users,
		roles:        d.Roles,
		signer:       d.Signer,
		ticketSigner: d.TicketSigner,
		tokenTTL:     d.TokenTTL,
		defaultWSTTL: d.WSTicketTTL,
	}
}

// Refresh issues a fresh token for the user identified by the existing one.
//
// The caller (HTTP middleware) has already parsed and validated the
// existing token before calling Refresh. This method re-loads the user
// from UserStore so role changes take effect on refresh.
func (s *SessionService) Refresh(ctx context.Context, userID string) (*VerifyResult, error) {
	id, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.roles != nil {
		if role, err := s.roles.Resolve(ctx, *id); err == nil && role != "" {
			id.Role = role
		}
	}
	tok, err := s.signer.Issue(*id, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Token: tok, Identity: *id}, nil
}

// IssueWSTicket mints a short-lived WebSocket ticket for the given user
// + scope (e.g. {"bot_id": "..."}).
func (s *SessionService) IssueWSTicket(ctx context.Context, userID string, scope map[string]string) (*domain.WSTicket, error) {
	if s.ticketSigner == nil {
		return nil, domain.ErrProviderUnavailable
	}
	return s.ticketSigner.Issue(userID, scope, s.defaultWSTTL)
}

// VerifyToken parses and validates a session token.
func (s *SessionService) VerifyToken(value string) (*domain.Identity, error) {
	if s.signer == nil {
		return nil, domain.ErrProviderUnavailable
	}
	return s.signer.Parse(value)
}

// VerifyWSTicket parses a WebSocket ticket.
func (s *SessionService) VerifyWSTicket(value string) (*domain.WSTicket, error) {
	if s.ticketSigner == nil {
		return nil, domain.ErrProviderUnavailable
	}
	return s.ticketSigner.Parse(value)
}
