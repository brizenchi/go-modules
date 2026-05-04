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
├── Dockerfile.standalone.example
├── cmd/quickstart/
│   ├── main.go
│   ├── main_test.go
│   └── reward.go
└── deploy/
    └── config.yaml.example
```

## Copy and run

```bash
cp -R templates/quickstart ~/code/your-new-service
cd ~/code/your-new-service

go mod init github.com/yourname/yournewservice
go get github.com/brizenchi/go-modules@latest
go mod tidy

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

- the copied service depends on the published
  `github.com/brizenchi/go-modules` root module
- no `replace` directives are required

If you want to pin a specific shared repo release:

```bash
go get github.com/brizenchi/go-modules@v0.3.0
```

If you need to iterate against local unpublished changes in this repo,
add a temporary replace in the copied service:

```bash
go mod edit -replace github.com/brizenchi/go-modules=/absolute/path/to/go-modules
go mod tidy
```

## Docker

### In This Monorepo

This Dockerfile is designed for monorepo-root build context.

From this repo root:

```bash
docker build -f templates/quickstart/Dockerfile -t quickstart .
docker run --rm -p 8080:8080 --env-file templates/quickstart/.env quickstart
```

Notes:

- build context must be the repo root `.`
- `templates/quickstart/Dockerfile.dockerignore` trims the monorepo
  context down to the files this image actually needs
- the image bakes in `templates/quickstart/deploy/config.yaml.example`
  as `/app/deploy/config.yaml` via the builder stage, so runtime stage
  does not need to re-read that file from the build context
- runtime env vars still override YAML at boot

### After Copying The Template Out

If you copied `templates/quickstart` into its own backend repo and ran
`go mod init` + `go mod tidy`, use this Dockerfile shape instead:

```bash
cp Dockerfile.standalone.example Dockerfile
docker build -t your-new-service .
docker run --rm -p 8080:8080 --env-file .env your-new-service
```

Notes:

- the runtime image bundles `deploy/config.yaml.example` as
  `/app/deploy/config.yaml`
- set real deployment values through environment variables; env still
  overrides YAML at boot
- default container port is `8080`

Recommended split:

- local debug: `go run ./cmd/quickstart`
- monorepo image: `templates/quickstart/Dockerfile`
- copied project image: `Dockerfile.standalone.example`

## Dokploy

For Dokploy monorepo deployment, configure:

- `Dockerfile Path`: `templates/quickstart/Dockerfile`
- `Docker Context Path`: `.`
- `Port`: `8080`

Recommended environment setup:

- set `CONFIG=/app/deploy/config.yaml`
- provide production values with Dokploy env vars such as
  `APP_DB_HOST`, `APP_DB_USER`, `APP_DB_PASSWORD`,
  `APP_DB_NAME`, `APP_AUTH_USER_JWT_SECRET`

Do not set Docker context to `templates/quickstart`, because this
template imports packages from the repo root module and needs
`foundation/`, `modules/`, `stacks/`, and the root `go.mod` during the
build.

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
