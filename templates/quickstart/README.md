# quickstart

Minimal Gin/GORM service that consumes every module in `go-modules/`.
Use this as the starting point for a new project — copy the directory,
rename the module path, fill in the project-specific glue.

## What you get

```
quickstart/
├── go.mod                    ← depends on all foundation/* + 4 business modules
├── cmd/main.go               ← boot wiring: slog + DB + Redis + Gin + 4 modules
├── internal/
│   ├── auth_glue/            ← UserStore + RoleResolver against your User model
│   ├── billing_glue/         ← CustomerStore + listeners
│   ├── email_glue/           ← viper → email.Module bridge
│   └── referral_glue/        ← code generator + GORM AutoMigrate
└── deploy/
    └── config.yaml.example   ← starting config
```

## Boot sequence (cmd/main.go)

```
1. slog.Setup
2. config.Load
3. pgx.Open  (DB)
4. rdx.Open  (Redis, optional)
5. email_glue.Init       → email.Module
6. auth_glue.Init        → auth.Module     (uses email_glue)
7. billing_glue.Init     → billing.Module  (uses your User model)
8. referral_glue.Init    → referral.Module (auto-migrates schema)
9. wire cross-module bridges:
     - auth UserSignedUp     → billing_glue.OnSignup (provision wallet)
     - billing SubActivated  → referral_glue.Activate
10. Gin engine + ginx middleware + httpresp helpers
11. Mount routes:
     - publicGroup:  auth.MountPublic + billing.MountPublic (webhook)
     - userGroup:    Bearer-auth middleware + auth.MountUser
                     + billing.MountUser + referral.MountUser
12. ListenAndServe + graceful shutdown
```

## Project-specific bits you MUST write

| File | What |
|---|---|
| `pkg/models/user.go` | Your User GORM model (whatever shape) |
| `internal/auth_glue/user_store.go` | Implement `auth.UserStore` against your User |
| `internal/billing_glue/customer_store.go` | Implement `billing.CustomerStore` |
| `internal/billing_glue/listeners.go` | What to do on subscription events (grant credits, send email, ...) |
| `internal/referral_glue/wire.go` | Pick deterministic vs random code generator |
| `deploy/config.yaml` | Real DB / Redis / Stripe / Brevo creds |

Everything else is wired by the modules — you don't write checkout
logic, OAuth flow, webhook idempotency, JWT signing, Brevo HTTP, etc.

## Time to first running service

About 1 hour: 30 min setting up DB + Stripe test creds, 30 min implementing
your User schema and the four `*_glue/` packages.

## Configuration

`deploy/config.yaml.example` ships with every key documented.

Required at minimum for the service to boot:
- `db.dsn` — Postgres
- `auth.user_jwt_secret` — random 32+ chars

Optional (each unlocks a feature):
- `auth.email.brevo_api_key` + `auth.email.sender_email` — real email
- `auth.google.client_id` + `secret` + `redirect_url` + `state_secret` — Google OAuth
- `billing.stripe.secret_key` + `webhook_secret` + price IDs — payments
- `redis.addr` — Redis (used by foundation/rdx if you opt in)

When optional features are not configured, the relevant adapter
gracefully falls back (email → log sender, OAuth → empty providers
map, billing → 503 on checkout). The service still boots.

## Versioning

When `go-modules` ships a new version, bump in your `go.mod`:

```bash
go get github.com/brizenchi/go-modules/auth@v0.2.0
go mod tidy
```

`replace` directives are NOT used by quickstart — it consumes published
versions. To work against a local checkout, add temporarily:

```go
// go.mod (local development only — DO NOT commit)
replace github.com/brizenchi/go-modules/auth => ../../go-modules/auth
```

## Make the new project yours

```bash
# 1. Copy
cp -R templates/quickstart ~/code/your-new-service
cd ~/code/your-new-service

# 2. Rename module path
NEW_MOD=github.com/yourname/yournewservice
find . -name '*.go' -o -name 'go.mod' | xargs sed -i '' \
  "s|github.com/brizenchi/go-modules/templates/quickstart|$NEW_MOD|g"
go mod edit -module "$NEW_MOD"

# 3. Tidy + build
go mod tidy
go build ./...

# 4. Iterate on internal/*_glue/
```
