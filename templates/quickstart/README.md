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

## Template files

```text
quickstart/
├── .dockerignore
├── .env.example
├── Dockerfile
├── Dockerfile.demo
├── go.mod
├── cmd/quickstart/
│   ├── main.go
│   ├── main_test.go
│   └── reward.go
├── deploy/
│   └── config.yaml.example
└── scripts/
    └── use-remote-go-modules.sh
```

## Copy and run

```bash
cp -R templates/quickstart ~/code/your-new-service
cd ~/code/your-new-service

NEW_MOD=github.com/yourname/yournewservice
find . -name '*.go' -o -name 'go.mod' | xargs sed -i '' \
  "s|github.com/brizenchi/go-modules/templates/quickstart|$NEW_MOD|g"
go mod edit -module "$NEW_MOD"

cp .env.example .env
cp deploy/config.yaml.example deploy/config.yaml

go test ./...
go build ./...
go run ./cmd/quickstart
```

For local development, `go run ./cmd/quickstart` will auto-load `.env`
from the current directory if the file exists. Existing process
environment variables still win over `.env`.

## Dependency mode

Default copied state:

- `go.mod` keeps local `replace` directives for repo-local iteration

If the new repo should switch to published GitHub module tags:

```bash
bash scripts/use-remote-go-modules.sh
```

That only works after the required `foundation/*`, `modules/*`, and
`stacks/*` tags have been published.

## Docker

This template now includes two Docker build modes.

For current `go-modules` repo demo usage:

```bash
docker build -f templates/quickstart/Dockerfile.demo -t quickstart-demo .
docker run --rm -p 8080:8080 --env-file templates/quickstart/.env quickstart-demo
```

Notes:

- build context is the repo root
- uses the current workspace code and local `replace` directives
- use this when you want a quick online demo from the monorepo snapshot

Build after copying the template into a new backend repo:

```bash
docker build -t your-new-service .
docker run --rm -p 8080:8080 --env-file .env your-new-service
```

Notes:

- the image build automatically drops the template's local `replace`
  directives and resolves published `go-modules` tags instead
- the runtime image bundles `deploy/config.yaml.example` as
  `/app/deploy/config.yaml`
- set real deployment values through environment variables; env still
  overrides YAML at boot
- default container port is `8080`

Recommended split:

- local debug: `go run ./cmd/quickstart`
- monorepo demo image: `Dockerfile.demo`
- copied project image: `Dockerfile`

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

Email provider defaults:

- `email.provider=log` for local/dev
- `email.provider=brevo` when using Brevo template-based delivery
- `email.provider=resend` when using Resend API delivery

## What you usually change

- `.env` and `deploy/config.yaml`
- `cmd/quickstart/reward.go`
- host-specific routes or hooks around the shared stack

Main extension surface:

- `saascore.HostHooks`
- `saascore.PolicyHooks`

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
5. referral signup with `?ref=` activates after paid subscription

## When not to use this template

- your project already has a different `users` schema
- you only need one module, not the full shared stack
- you want a custom auth or billing model from day one
