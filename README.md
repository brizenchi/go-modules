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

| Module | Path | What |
|---|---|---|
| `slog` | `foundation/slog` | `slog.SetDefault` setup + Gin context helper |
| `jwt` | `foundation/jwt` | HS256/RS256 signer with leeway + alg-confusion guard |
| `httpresp` | `foundation/httpresp` | Uniform `{code, msg, data}` Gin response helpers |
| `ginx` | `foundation/ginx` | CORS, RequestID, Recover, AccessLog, Secure, NoCache middleware |
| `config` | `foundation/config` | viper-backed config loader (file + env) |
| `pgx` | `foundation/pgx` | GORM-postgres connection helper with slog-backed query logger |
| `rdx` | `foundation/rdx` | Redis client + simple distributed lock |

### Business modules — DDD-shaped, project-portable

| Module | What |
|---|---|
| `auth` | Email-code + Google OAuth + JWT sessions + WS tickets. Pluggable UserStore. |
| `billing` | Stripe subscriptions/checkout/credits + webhooks. Pluggable CustomerStore + listeners. |
| `email` | Provider-agnostic transactional email (Brevo/SMTP/log/template). |
| `referral` | Referrer→referee codes + activation events. |

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

- One module per PR when possible — keeps blast radius small.
- Foundation modules require deprecation cycle for breaking changes (one minor version).
- Business modules can break majors freely (semver).
- Every module ships its own README + tests; CI enforces.
- New module? Drop a `go.mod` + add to `go.work` + add to Makefile loops.
