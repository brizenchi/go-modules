package auth_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brizenchi/go-modules/modules/auth"
	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

type lifetimeUsers struct{ port.UserStore }

func (lifetimeUsers) FindByID(context.Context, string) (*domain.Identity, error) {
	return &domain.Identity{UserID: "lifetime-user", Email: "lifetime@example.test"}, nil
}

func (users lifetimeUsers) FindOrCreateByEmail(ctx context.Context, email string) (*domain.Identity, error) {
	return users.FindByID(ctx, email)
}

func (lifetimeUsers) MarkLogin(context.Context, string) error { return nil }

type lifetimeVerifier struct{}

func (lifetimeVerifier) Verify(context.Context, string, string) error { return nil }

type lifetimeExchange struct{ port.ExchangeCodeStore }

func (lifetimeExchange) Consume(context.Context, string, string) (*domain.ExchangeCode, error) {
	return &domain.ExchangeCode{UserID: "lifetime-user"}, nil
}

func TestModulePropagatesLifetimes(t *testing.T) {
	for _, configured := range []bool{false, true} {
		name := "defaults"
		if configured {
			name = "configured"
		}
		t.Run(name, func(t *testing.T) {
			signer, err := authjwt.NewSigner(authjwt.Config{Secret: "module-lifetime-test"})
			if err != nil {
				t.Fatal(err)
			}
			tickets, err := authjwt.NewTicketSigner(authjwt.Config{Secret: "module-ticket-test"})
			if err != nil {
				t.Fatal(err)
			}
			deps := auth.Deps{UserStore: lifetimeUsers{}, TokenSigner: signer, WSTicketSigner: tickets,
				EmailCodeVerifier: lifetimeVerifier{}, ExchangeCodeStore: lifetimeExchange{}}
			wantToken, wantTicket := 7*24*time.Hour, 5*time.Minute
			if configured {
				wantToken, wantTicket = time.Hour, 30*time.Second
				deps.TokenTTL, deps.WSTicketTTL = wantToken, wantTicket
			}
			module := auth.New(deps)
			for _, login := range []struct {
				name string
				run  func() (*app.VerifyResult, error)
			}{
				{"email", func() (*app.VerifyResult, error) {
					return module.Login.VerifyCode(t.Context(), "lifetime@example.test", "123456")
				}},
				{"oauth", func() (*app.VerifyResult, error) {
					return module.OAuth.ExchangeToken(t.Context(), "exchange", strings.Repeat("A", 43))
				}},
				{"refresh", func() (*app.VerifyResult, error) { return module.Session.Refresh(t.Context(), "lifetime-user") }},
			} {
				result, err := login.run()
				if err != nil {
					t.Fatalf("%s: %v", login.name, err)
				}
				if remaining := time.Until(result.Token.ExpiresAt); remaining > wantToken || remaining < wantToken-time.Second {
					t.Errorf("%s TTL = %s, want %s", login.name, remaining, wantToken)
				}
			}
			ticket, err := module.Session.IssueWSTicket(t.Context(), "lifetime-user", nil)
			if err != nil {
				t.Fatal(err)
			}
			if remaining := time.Until(ticket.ExpiresAt); remaining > wantTicket || remaining < wantTicket-time.Second {
				t.Errorf("ticket TTL = %s, want %s", remaining, wantTicket)
			}
		})
	}
}
