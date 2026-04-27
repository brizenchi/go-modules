# auth

> Portable, provider-agnostic authentication: email-code passwordless + Google OAuth + JWT sessions + WebSocket tickets.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/auth.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/auth)

The `auth` module never imports project-specific models. Hosts integrate
via four interfaces and own their user table.

## Install

```bash
go get github.com/brizenchi/go-modules/auth
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
  memstore/    in-memory rate limit + exchange code (dev/tests)
  eventbus/    in-process synchronous bus
app/      use cases: SendCode, VerifyCode, StartOAuth, OAuthCallback,
          ExchangeToken, RefreshToken, IssueWSTicket
http/     Gin handlers, UserAuthMiddleware, Mount()
```

## Host responsibilities

The host project provides:

1. **`port.UserStore`** — read/write the host's user table (`FindByEmail`, `CreateFromIdentity`, ...)
2. **`port.RoleResolver`** — assign roles to identities (e.g. "admin" / "user")
3. **`port.ExchangeCodeStore`** — persist OAuth callback codes (memstore for dev, GORM for prod)
4. **Event listeners** — react to `UserSignedUp` / `UserLoggedIn` (welcome email, analytics, ...)

## Quick start

```go
import (
    "github.com/brizenchi/go-modules/auth"
    "github.com/brizenchi/go-modules/auth/adapter/emailcode"
    "github.com/brizenchi/go-modules/auth/adapter/eventbus"
    "github.com/brizenchi/go-modules/auth/adapter/google"
    authjwt "github.com/brizenchi/go-modules/auth/adapter/jwt"
    "github.com/brizenchi/go-modules/auth/adapter/memstore"
)

// 1. Token signers
sessionSigner := authjwt.NewSessionSigner("secret-32-bytes-or-more...", "auth-svc")
wsSigner      := authjwt.NewWSTicketSigner("ws-secret...", 60*time.Second)

// 2. Email-code issuer + verifier (uses pkg/email's SendService under the hood)
codeIssuer, codeVerifier := emailcode.NewPair(emailcode.Config{
    Mailer:        emailMod.Service,        // from pkg/email
    Sender:        domain.Address{Email: "no-reply@acme.com"},
    TemplateRef:   "3",                     // brevo template id
    TTL:           5 * time.Minute,
    RateLimit:     memstore.NewRateLimit(), // or redisstore for multi-instance
})

// 3. Google OAuth (optional)
googleProv := google.NewProvider(google.Config{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    RedirectURL:  "https://app.example.com/api/v1/auth/google/callback",
})

// 4. Wire the module
mod := auth.New(auth.Deps{
    UserStore:         myUserStore,           // your impl
    RoleResolver:      myRoleResolver,        // your impl
    ExchangeCodeStore: memstore.NewExchangeCodeStore(),
    TokenSigner:       sessionSigner,
    WSTicketSigner:    wsSigner,
    EmailCodeIssuer:   codeIssuer,
    EmailCodeVerifier: codeVerifier,
    IdentityProviders: map[string]port.IdentityProvider{"google": googleProv},
    Bus:               eventbus.New(),
    FrontendURL:       "https://app.example.com",
})

// 5. Mount routes
public := r.Group("/api/v1")
public.POST("/auth/send-code", mod.Handler.SendCode)
public.POST("/auth/verify-code", mod.Handler.VerifyCode)
public.GET("/auth/:provider/authorize", mod.Handler.StartOAuth)
public.GET("/auth/:provider/callback", mod.Handler.OAuthCallback)
public.POST("/auth/exchange-token", mod.Handler.ExchangeToken)

// 6. Protect routes with the middleware factory
private := r.Group("/api/v1", mod.Handler.RequireUser())
private.POST("/auth/refresh", mod.Handler.Refresh)
private.POST("/auth/logout", mod.Handler.Logout)
```

## Flows

### Email-code

```
POST /auth/send-code         { email }              → sends 6-digit code
POST /auth/verify-code       { email, code }        → returns access + refresh
POST /auth/refresh           { refresh_token }      → new access
POST /auth/logout            (Bearer)               → revokes refresh
```

### Google OAuth

```
GET  /auth/google/authorize  → 302 to Google
GET  /auth/google/callback   → 302 to FrontendURL?exchange_code=xxx
POST /auth/exchange-token    { exchange_code }      → returns access + refresh
```

The intermediate `exchange_code` keeps tokens out of browser history.

### WebSocket ticket

```
POST /websocket/ticket       (Bearer)               → returns short-lived ticket
```

Client passes ticket as `?ticket=<jwt>` on the WebSocket connection;
your WS handler verifies with the same `WSTicketSigner`.

## Testing

```bash
go test -race ./...
```

Coverage: domain 100%, memstore 97.7%, emailcode 87.1%, jwt 79.2%, google 40%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
