# quickstart

Thin HTTP host shell for the shared `stacks/saascore` stack.

Use it when you want:

- a normal Go HTTP service
- shared auth / billing / referral from `go-modules`
- your own business logic to stay in your own handler / service / repository / model packages
- host-side writeback through hooks instead of modifying shared stack code

Read first:

1. [`docs/SAASCORE_GUIDE.md`](../../docs/SAASCORE_GUIDE.md)
2. [`docs/INTEGRATION.md`](../../docs/INTEGRATION.md)

## Ownership split

`go-modules` owns:

- `foundation/*`: config, logging, tracing, db, HTTP helpers
- `modules/*`: shared domain modules such as `user`
- `stacks/saascore`: standard auth, billing, referral flow and shared routes

`quickstart` owns:

- process boot
- config mapping
- DB / tracing / logger assembly
- mounting the shared stack
- host hooks for your business writeback
- host route registration for your own features

## Template tree

```text
quickstart/
├── cmd/quickstart/
│   ├── main.go
│   └── main_test.go
├── internal/
│   ├── bootstrap/
│   │   ├── app.go
│   │   ├── config.go
│   │   ├── host_hooks.go
│   │   ├── referral.go
│   │   └── saascore.go
│   └── http/
│       ├── host_routes.go
│       ├── router.go
│       └── middleware/router.go
├── deploy/config.yaml.example
├── .env.example
├── Dockerfile
├── go.mod
└── go.sum
```

Directory intent:

- `cmd/quickstart`: process lifecycle only
- `internal/bootstrap/app.go`: assemble logger, tracing, db, shared stack, HTTP server
- `internal/bootstrap/config.go`: app config schema and defaults
- `internal/bootstrap/saascore.go`: map host config into shared `saascore.Config`
- `internal/bootstrap/host_hooks.go`: fill in business callbacks after signup, login, payment, referral, subscription events
- `internal/bootstrap/referral.go`: default referral reward writeback to shared `users.credits`
- `internal/http/router.go`: compose shared routes and host routes into one router
- `internal/http/host_routes.go`: register your own business routes here
- `internal/http/middleware/router.go`: root Gin middleware setup

## How you develop on top of it

1. Copy the template and fill config.
2. Keep shared auth / billing / referral in `saascore`.
3. Put your own side effects in `internal/bootstrap/host_hooks.go`.
4. Register your own business routes in `internal/http/host_routes.go`.
5. Add your own feature folders only when needed.

Recommended host business layout:

- `internal/http/handler/<feature>/...`
- `internal/service/<feature>/...`
- `internal/repository/<feature>/...`
- `internal/model/entity/<feature>/...`
- `internal/integration/<provider>/...`

## Copy and run

```bash
cp -R templates/quickstart ~/code/your-new-service
cd ~/code/your-new-service

cp .env.example .env
cp deploy/config.yaml.example deploy/config.yaml

go test ./...
go build ./...
go run ./cmd/quickstart
```

Local `go run ./cmd/quickstart` auto-loads `.env` if it exists. Process env
variables still override `.env` and YAML.

## Minimum config

Required to boot:

- `db.host`
- `db.user`
- `db.password`
- `db.name`
- `auth.user_jwt_secret`

Common optional groups:

- `auth.google.*`
- `email.*`
- `billing.stripe.*`
- `referral.*`
- `tracing.*`
- `project`
- `env`

Stripe shared billing price slots:

- `billing.stripe.prices.starter_monthly`
- `billing.stripe.prices.starter_yearly`
- `billing.stripe.prices.pro_monthly`
- `billing.stripe.prices.pro_yearly`
- `billing.stripe.prices.premium_monthly`
- `billing.stripe.prices.premium_yearly`
- `billing.stripe.prices.lifetime`
- `billing.stripe.prices.credits[]`

## What you usually change

- `.env`
- `deploy/config.yaml`
- `internal/bootstrap/host_hooks.go`
- `internal/http/host_routes.go`
- your own feature folders under `internal/`

## What you should not rewrite

- JWT signing and verification
- email-code login flow
- Google OAuth callback exchange flow
- Stripe checkout session creation
- Stripe webhook parsing and idempotency
- shared referral activation flow

## Manual verification

Before production, confirm:

1. email-code login works
2. Google OAuth works when configured
3. protected routes reject missing bearer tokens
4. Stripe checkout and webhook flow work against a public backend URL
5. referral signup with `?ref=` activates after paid subscription

## When not to use this template

- your project already has a different `users` schema
- you only need one module, not the full shared stack
- you want a completely custom auth or billing model from day one
