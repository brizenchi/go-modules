# auth

> Portable, provider-agnostic authentication: email-code passwordless + Google OAuth + JWT sessions + WebSocket tickets.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/modules/auth.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/modules/auth)

The `auth` module never imports project-specific models. Hosts integrate
via ports and keep ownership of their user table, role rules, and route tree.

## Install

```bash
go get github.com/brizenchi/go-modules/modules/auth
```

## Layering

```
domain/   pure types: Identity, Token, OAuthProfile, errors
event/    domain events (UserSignedUp, UserLoggedIn)
port/     interfaces the module depends on
adapter/  concrete implementations
  jwt/         HS256 token + WS ticket signer
  google/      Google OAuth IdentityProvider
  emailcode/   passwordless email-code provider
  memstore/    in-memory code + exchange stores for dev/tests
  eventbus/    in-process synchronous bus
app/      use cases: SendCode, VerifyCode, StartOAuth, OAuthCallback,
          ExchangeToken, Refresh, IssueWSTicket
http/     Gin handlers, middleware, Mount()
```

## Host responsibilities

1. `port.UserStore` — map the module to your user table.
2. `port.RoleResolver` — assign `user` / `admin` roles.
3. `port.ExchangeCodeStore` — persist OAuth exchange codes.
4. Event listeners — react to `UserSignedUp` / `UserLoggedIn`.

## Quick start

```go
import (
	"time"

	"github.com/brizenchi/go-modules/modules/auth"
	"github.com/brizenchi/go-modules/modules/auth/adapter/emailcode"
	autheventbus "github.com/brizenchi/go-modules/modules/auth/adapter/eventbus"
	"github.com/brizenchi/go-modules/modules/auth/adapter/google"
	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	"github.com/brizenchi/go-modules/modules/auth/adapter/memstore"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	"github.com/brizenchi/go-modules/modules/auth/port"
)

sessionSigner, err := authjwt.NewSigner(authjwt.Config{
	Secret:  os.Getenv("AUTH_JWT_SECRET"),
	Issuer:  "app-auth",
	UserTTL: 7 * 24 * time.Hour,
})
if err != nil {
	log.Fatal(err)
}

ticketSigner, err := authjwt.NewTicketSigner(authjwt.Config{
	Secret:    os.Getenv("AUTH_JWT_SECRET"),
	Issuer:    "app-auth-ws",
	TicketTTL: 5 * time.Minute,
})
if err != nil {
	log.Fatal(err)
}

codeStore := memstore.NewCodeStore()
exchangeStore := memstore.NewExchangeStore()

issuer := emailcode.NewIssuer(emailcode.Config{
	TTL:         10 * time.Minute,
	MaxAttempts: 5,
	TemplateRef: "3",
}, codeStore, myMailerAdapter)

verifier := emailcode.NewVerifier(emailcode.Config{
	MaxAttempts: 5,
}, codeStore)

providers := map[string]port.IdentityProvider{}
googleProvider, err := google.New(google.Config{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	RedirectURL:  "https://api.example.com/api/v1/auth/google/callback",
	StateSecret:  os.Getenv("GOOGLE_STATE_SECRET"),
	StateTTL:     20 * time.Minute,
})
if err == nil {
	providers["google"] = googleProvider
}

mod := auth.New(auth.Deps{
	UserStore:         myUserStore,
	RoleResolver:      myRoleResolver,
	TokenSigner:       sessionSigner,
	WSTicketSigner:    ticketSigner,
	ExchangeCodeStore: exchangeStore,
	EmailCodeIssuer:   issuer,
	EmailCodeVerifier: verifier,
	IdentityProviders: providers,
	Bus:               autheventbus.NewInProc(),
	FrontendURL:       "https://app.example.com/login",
})

public := r.Group("/api/v1")
user := r.Group("/api/v1")
user.Use(authhttp.RequireUser(mod.Session))

authhttp.Mount(mod.Handler, public, user)
```

`modules/email` can be used for delivery, but `adapter/emailcode` only
depends on a tiny `Mailer` interface. See
[`templates/quickstart/internal/auth_glue`](../../templates/quickstart/internal/auth_glue/)
for a concrete host-side wrapper.

## HTTP flows

### Email-code login

```
POST /auth/send-code    { "email": "user@example.com" }
POST /auth/verify-code  { "email": "user@example.com", "code": "123456" }
POST /auth/refresh      Authorization: Bearer <jwt>
POST /auth/logout       Authorization: Bearer <jwt>
```

`SendCode` returns `debug_code` only when debug mode is enabled on the issuer.

### Google OAuth

```
GET  /auth/google/authorize
GET  /auth/google/callback
POST /auth/exchange-token  { "code": "<exchange_code>" }
```

`GET /auth/:provider/authorize` returns JSON containing `redirect_url`.
If `FrontendURL` is configured, `OAuthCallback` redirects the browser to
`FrontendURL?code=<exchange_code>`. Otherwise it returns JSON containing
`exchange_code`.

### WebSocket ticket

```
POST /websocket/ticket  Authorization: Bearer <jwt>
```

The response contains a short-lived JWT ticket for host-specific WS entry points.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
