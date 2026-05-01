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
├── .env.example
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
