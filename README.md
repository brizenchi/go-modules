# go-modules

Multi-module monorepo of reusable Go packages used across our backend services.

```
foundation/   ← stable infrastructure (logger, db, http middleware, jwt, ...)
{auth,billing,email,referral}/   ← business modules with DDD layering
```

Each subdirectory is its own Go module (independent `go.mod` + version),
imported as `github.com/brizenchi/go-modules/<MODULE>`.

## Modules

### foundation/ — boring, stable, project-agnostic

| Module | Latest | What |
|---|---|---|
| [`foundation/slog`](./foundation/slog/) | v0.1.0 | `log/slog` Setup + Gin context helper |
| [`foundation/jwt`](./foundation/jwt/) | v0.1.0 | HS256/RS256 signer with leeway + alg-confusion guard |
| [`foundation/httpresp`](./foundation/httpresp/) | v0.1.0 | Uniform `{code, msg, data}` Gin response helpers |
| [`foundation/ginx`](./foundation/ginx/) | v0.1.0 | CORS, RequestID, Recover, AccessLog, Secure, NoCache middleware |
| [`foundation/config`](./foundation/config/) | v0.1.0 | viper-backed config loader (file + env) |
| [`foundation/pgx`](./foundation/pgx/) | v0.1.0 | GORM-postgres connection helper with slog-backed query logger |
| [`foundation/rdx`](./foundation/rdx/) | v0.1.0 | Redis client + simple distributed lock |

### Business modules — DDD-shaped, project-portable

| Module | Latest | What |
|---|---|---|
| [`auth`](./auth/) | v0.1.1 | Email-code + Google OAuth + JWT sessions + WS tickets. Pluggable UserStore. |
| [`billing`](./billing/) | v0.2.0 | Stripe subscriptions/checkout/credits + webhooks. Caller metadata pass-through (Rewardful). |
| [`email`](./email/) | v0.2.0 | Provider-agnostic transactional email (Brevo / Resend / SMTP / log / template). |
| [`referral`](./referral/) | v0.1.0 | C2C referral codes + attribution + activation events. |

Each business module follows the same layering — see its README:

```
domain/   pure types
event/    domain events
port/     interfaces (the package depends on these)
adapter/  default implementations (provider SDKs, GORM, etc.)
app/      use cases
http/     Gin handlers + Mount()
```

## Quick start (for a new project)

```bash
# In your project's go.mod:
go get github.com/brizenchi/go-modules/foundation/slog@latest
go get github.com/brizenchi/go-modules/foundation/ginx@latest
go get github.com/brizenchi/go-modules/auth@latest
go get github.com/brizenchi/go-modules/billing@latest
```

```go
// At boot:
import (
    fslog "github.com/brizenchi/go-modules/foundation/slog"
    "github.com/brizenchi/go-modules/foundation/ginx"
    "github.com/brizenchi/go-modules/foundation/pgx"
    "github.com/brizenchi/go-modules/auth"
    "github.com/brizenchi/go-modules/billing"
)

fslog.Setup(fslog.Config{Level: "info", Format: "json", Defaults: map[string]any{"service": "myapi"}})
db, _ := pgx.Open(pgx.Config{Host: ..., User: ..., Database: ...})

r := gin.New()
r.Use(ginx.Recover(), ginx.RequestID(), ginx.AccessLog(ginx.AccessLogConfig{}))
r.Use(ginx.CORS(ginx.CORSConfig{AllowedOrigins: []string{"https://app"}}))

// Wire your project glue (UserStore, CustomerStore, listeners) and Mount().
// See each module's README.
```

## Versioning

Tags use a `<module>/vX.Y.Z` prefix — required for multi-module repos:

```bash
git tag foundation/slog/v1.0.0
git tag auth/v1.0.0
```

See [VERSIONING.md](VERSIONING.md) for the full policy.

## Status

🟢 **Verified by `clawmesh-backend` in production.** All 26+ test packages
pass with `-race`; full E2E (18/18) passes against a live Postgres + Brevo
integration. See each module's CHANGELOG.md for version history.

Per-module test coverage (latest):

| Module | Coverage highlights |
|---|---|
| `auth/domain` | 100% |
| `auth/adapter/{memstore, emailcode}` | 97.7% / 87.1% |
| `billing/adapter/{eventbus, stripe}` | 100% / 76.1% |
| `billing/domain` | 75% |
| `email/adapter/{log, resend, brevo, smtp}` | 100% / 85% / 68.2% / 68.9% |
| `email/app` | 94.1% |
| `referral/{app, adapter/codegen}` | 78% / 84% |

## Local development

This repo uses Go workspaces. `go.work` declares all 11 modules so cross-module
edits resolve to local paths (no `replace` directives needed):

```bash
make test         # run tests in every module
make tidy         # go mod tidy in every module
make fmt          # gofmt -s
make purity-check # ensure pkg/* doesn't leak project-specific imports
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full guide. TL;DR:

- One module per PR when possible — keeps blast radius small.
- Foundation modules require a deprecation cycle for breaking changes
  (one minor version).
- Business modules can break majors freely (semver).
- Every module ships its own README + CHANGELOG + tests; CI enforces.
- New module? Drop a `go.mod`, add to `go.work`, add to the Makefile
  loops, and add to the CI matrix in `.github/workflows/ci.yml`.
