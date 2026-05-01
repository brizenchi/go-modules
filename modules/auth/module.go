package auth

import (
	"github.com/brizenchi/go-modules/modules/auth/app"
	"github.com/brizenchi/go-modules/modules/auth/event"
	httpapi "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

// Module bundles the wired use cases + handlers for one auth setup.
type Module struct {
	Deps Deps

	Login   *app.LoginService
	OAuth   *app.OAuthService
	Session *app.SessionService
	Handler *httpapi.Handler
}

// Deps gathers the host-supplied collaborators.
//
// IdentityProviders is a map keyed by provider name (matching the
// :provider URL parameter on /auth/:provider/authorize).
// Pass an empty map to disable OAuth — the email-code flow still works.
type Deps struct {
	UserStore         port.UserStore
	RoleResolver      port.RoleResolver
	TokenSigner       port.TokenSigner
	WSTicketSigner    port.WSTicketSigner
	ExchangeCodeStore port.ExchangeCodeStore
	EmailCodeIssuer   port.EmailCodeIssuer
	EmailCodeVerifier port.EmailCodeVerifier
	IdentityProviders map[string]port.IdentityProvider
	Bus               port.EventBus

	// FrontendURL is the SPA URL for OAuth callback browser redirects.
	// When empty, the callback returns JSON instead of redirecting.
	FrontendURL string
}

// New wires the module from its dependencies.
func New(d Deps) *Module {
	login := app.NewLoginService(app.LoginDeps{
		Issuer:   d.EmailCodeIssuer,
		Verifier: d.EmailCodeVerifier,
		Users:    d.UserStore,
		Roles:    d.RoleResolver,
		Signer:   d.TokenSigner,
		Bus:      d.Bus,
	})
	oauth := app.NewOAuthService(app.OAuthDeps{
		Providers:     d.IdentityProviders,
		Users:         d.UserStore,
		Roles:         d.RoleResolver,
		Signer:        d.TokenSigner,
		ExchangeStore: d.ExchangeCodeStore,
		Bus:           d.Bus,
	})
	session := app.NewSessionService(app.SessionDeps{
		Users:        d.UserStore,
		Roles:        d.RoleResolver,
		Signer:       d.TokenSigner,
		TicketSigner: d.WSTicketSigner,
	})
	handler := httpapi.NewHandler(httpapi.Deps{
		Login:       login,
		OAuth:       oauth,
		Session:     session,
		FrontendURL: d.FrontendURL,
	})
	return &Module{
		Deps:    d,
		Login:   login,
		OAuth:   oauth,
		Session: session,
		Handler: handler,
	}
}

// Subscribe is a thin pass-through to the bus.
func (m *Module) Subscribe(kind event.Kind, fn port.Listener) {
	if m.Deps.Bus != nil {
		m.Deps.Bus.Subscribe(kind, fn)
	}
}
