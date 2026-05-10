# quickstart

Runnable backend starter for the standard shared `stacks/saascore`
contract.

Read first:

1. [`docs/SAASCORE_GUIDE.md`](../../docs/SAASCORE_GUIDE.md)
2. [`docs/INTEGRATION.md`](../../docs/INTEGRATION.md)

## Use this template when

- the new backend can adopt the shared `users` schema
- auth, billing, and referral should follow the standard SaaS flow
- you want to start from the maintained stack, not rebuild glue code

## Already included

- `stacks/saascore`
- shared `modules/user` schema and migrations
- JWT auth, email-code login, Google OAuth, WS ticket issuance
- Stripe checkout routes, webhook parsing, and idempotency handling
- referral attribution, activation, and user-facing referral routes
- structured logging, request id middleware, trace-aware HTTP setup
- standard observability fields: `service`, `project`, `env`, `request_id`, `trace_id`, `span_id`

## Template files

```text
quickstart/
├── .dockerignore
├── .env.example
├── Dockerfile
├── go.mod
├── go.sum
├── cmd/quickstart/
│   ├── main.go
│   └── main_test.go
├── internal/
│   ├── bootstrap/
│   │   ├── app.go
│   │   ├── billing.go
│   │   ├── config.go
│   │   ├── referral.go
│   │   └── saascore.go
│   ├── http/
│   │   ├── router.go
│   │   ├── handler/
│   │   │   └── billing/
│   │   │       └── topup.go
│   │   ├── routes/
│   │   │   └── billing.go
│   │   └── middleware/
│   │       └── router.go
│   ├── integration/
│   │   └── stripe/
│   │       └── topup_client.go
│   ├── model/
│   │   └── entity/
│   │       └── billing/
│   │           └── topup_event.go
│   ├── repository/
│   │   └── billing/
│   │       └── topup_event.go
│   └── service/
│       └── billing/
│           └── topup.go
└── deploy/
    └── config.yaml.example
```

Structure rules:

- `cmd/quickstart` only owns process lifecycle
- `internal/bootstrap` owns config loading and dependency assembly
- `internal/http` owns route assembly and HTTP transport concerns
- `internal/http/handler` owns HTTP handlers and transport-only request parsing
- `internal/http/middleware` owns Gin middleware wiring
- `internal/http/routes` owns feature route registration
- `internal/service` owns business logic
- `internal/repository` owns persistence details
- `internal/model/entity` owns project-local table models
- `internal/integration` owns third-party SDK interaction
- feature code can be nested by domain inside those layers, for example `billing/*`
- shared SaaS routes still come from `stacks/saascore`

This template intentionally does not create a global `dto` package by
default. Small request/response structs stay close to the handler first.
Only split them into feature-local transport packages when a business
module grows enough to justify that extra layer.

Current example:

- the custom Stripe credits top-up flow is grouped under the `billing` domain across handler, service, repository, and entity layers

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

For local development, `go run ./cmd/quickstart` will auto-load `.env`
from the current directory if the file exists. Existing process
environment variables still win over `.env`.

Config intent:

- `.env.example` is the fast local-dev profile: text logs, `email.provider=log`, `auth.email.debug=true`
- `deploy/config.yaml.example` is the safer deploy baseline: structured logs, `auth.email.debug=false`
- Stripe becomes active automatically when both `billing.stripe.secret_key` and `billing.stripe.webhook_secret` are set; this template does not use a separate `billing.stripe.enabled` flag

## Dependency mode

Default copied state:

- this template is already a standalone Go module
- it depends on published `github.com/brizenchi/go-modules` versions
- no extra `go mod init` step is required

## Docker

### In This Directory

This Dockerfile is designed for `templates/quickstart` as its own build
context.

From this directory:

```bash
docker build -t quickstart .
docker run --rm -p 8080:8080 --env-file .env quickstart
```

Notes:

- build context is this template directory
- the image bakes in `deploy/config.yaml.example` as
  `/app/deploy/config.yaml`
- runtime env vars still override YAML at boot

Recommended split:

- local debug: `go run ./cmd/quickstart`
- deploy image: `Dockerfile`

## Dokploy

For Dokploy deployment, use Dockerfile build type with:

- `Dockerfile Path`: `Dockerfile`
- `Docker Context Path`: `templates/quickstart`
- `Docker Build Stage`: leave empty
- `Port`: `8080`

Recommended environment setup:

- set `CONFIG=/app/deploy/config.yaml`
- provide production values with Dokploy env vars such as
  `APP_DB_HOST`, `APP_DB_USER`, `APP_DB_PASSWORD`,
  `APP_DB_NAME`, `APP_AUTH_USER_JWT_SECRET`

This template is intended to build and deploy directly from
`templates/quickstart`.

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
- `project` / `env`

Stripe quickstart price slots:

- `billing.stripe.prices.starter_monthly`
- `billing.stripe.prices.starter_yearly`
- `billing.stripe.prices.pro_monthly`
- `billing.stripe.prices.pro_yearly`
- `billing.stripe.prices.premium_monthly`
- `billing.stripe.prices.premium_yearly`
- `billing.stripe.prices.lifetime`
- `billing.stripe.prices.credits[]`
- `billing.stripe.topup.min_amount_usd`
- `billing.stripe.topup.max_amount_usd`
- `billing.stripe.topup.credits_per_usd`

Observability-focused env keys:

- `APP_PROJECT`
- `APP_ENV`
- `APP_TRACING_AUTHORIZATION`
- `APP_DB_LOG_LEVEL`
- `APP_DB_SLOW_QUERY_MS`

Email provider defaults:

- `email.provider=log` for local/dev
- `email.provider=brevo` when using Brevo template-based delivery
- `email.provider=resend` when using Resend API delivery

Local auth defaults:

- `.env.example` uses `auth.email.debug=true` so local frontend work does not require a real mailbox
- `deploy/config.yaml.example` keeps `auth.email.debug=false` to avoid leaking OTPs in non-dev environments

Google OAuth defaults:

- `auth.google.state_ttl_minutes=20`
- if you see `auth: invalid oauth state ... token is expired`, first check callback delay, stale authorize links, and backend server time

## What you usually change

- `.env` and `deploy/config.yaml`
- `internal/http/handler/*`
- `internal/service/*`
- `internal/repository/*`
- `internal/model/entity/*`
- host-specific routes or hooks around the shared stack

Main extension surface:

- `saascore.HostHooks`
- `saascore.PolicyHooks`
- `internal/bootstrap/*`
- `internal/http/*`
- `internal/service/*`

## What you should not rewrite

- JWT signing and verification
- email-code login flow
- Google OAuth callback exchange flow
- Stripe checkout session creation
- Stripe webhook parsing and idempotency
- referral repositories and HTTP handlers

## Manual verification

Before calling this template production-ready, confirm:

1. email-code login works
2. Google OAuth works when configured
3. protected routes reject missing bearer tokens
4. Stripe checkout and webhook flow work against a public backend URL
5. custom-amount top-up can create a PaymentIntent and add credits after webhook confirmation
6. referral signup with `?ref=` activates after paid subscription

## When not to use this template

- your project already has a different `users` schema
- you only need one module, not the full shared stack
- you want a custom auth or billing model from day one
