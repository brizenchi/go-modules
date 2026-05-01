# Config Standard

This document defines the recommended configuration and bootstrap split
for projects built on top of `go-modules`.

Goal:

- keep shared SaaS capabilities standardized
- keep business projects flexible
- avoid repeating the same config wiring across many services
- avoid hiding config behavior inside reusable modules

## Core rule

Only the outer application bootstrap reads config files or environment
variables.

That means:

- `foundation/*` does not read `.env`
- `modules/*` does not read `.env`
- `stacks/*` does not read `.env`
- only the host app startup layer loads `.env`, YAML, or deployment env

Reusable packages should only receive typed config values and concrete
dependencies.

## Recommended ownership split

### `go-modules` should own

- shared config structs for reusable modules and stacks
- validation rules for those shared config structs
- mapping examples in templates
- standard startup templates such as `templates/quickstart`
- integration docs and configuration rules

### business projects should own

- actual `.env` files
- actual `config.yaml` files
- secrets manager / CI / container env injection
- host-only config fields
- final bootstrap and dependency assembly in `main.go`
- project-specific listeners, policies, routes, and background jobs

## Recommended config flow

Use this order:

1. code defaults
2. `config.yaml`
3. local `.env`
4. real process environment variables
5. validation
6. typed dependency assembly

Practical meaning:

- YAML is the structured baseline
- `.env` is a local development convenience layer
- deployment env is the final authority

## Recommended config layering

Use two config layers.

### layer 1: host app config

This belongs to the business project.

Example:

```go
type AppConfig struct {
    Server   ServerConfig
    Log      LogConfig
    Tracing  TracingConfig
    DB       DBConfig
    Auth     AuthConfig
    Email    EmailConfig
    Billing  BillingConfig
    Referral ReferralConfig
}
```

This layer may include:

- server name and port
- log format and level
- tracing exporter settings
- DB connectivity
- host-only feature flags
- host-only domains and product options

### layer 2: shared stack config

This belongs to `go-modules`.

Example:

- `stacks/saascore.Config`
- `modules/auth` typed config inputs
- `modules/billing` typed config inputs

This layer should only contain fields that the shared package actually
needs.

## Assembly rule

The host app loads a full `AppConfig`, then passes smaller typed
sub-configs into each dependency.

Example:

```go
cfg := loadAppConfig()

db := mustOpenDB(cfg.DB)
traceShutdown := mustSetupTracing(cfg.Tracing)

stack := mustNewSaaSCore(
    db,
    cfg.SaaSCoreConfig(),
    hostHooks,
    policyHooks,
)
```

Important:

- `main.go` may hold the full app config
- reusable modules should only receive the config slice they need
- do not expose a package-global mutable config object to all modules

## Validation rule

Validation should happen after config is fully loaded and before any
real dependency is opened.

### validate at the host layer

Host should validate:

- server port
- DB connection shape
- host-only domain settings
- project-specific required fields

### validate at the shared layer

Shared stack should validate:

- JWT secrets
- Google OAuth required pairs
- email provider requirements
- Stripe required fields when enabled
- referral link assumptions when needed

Recommended direction:

- keep host validation in the host app bootstrap
- keep shared business validation near the shared stack or module

## Logging rule

At startup, print a sanitized config summary.

Print:

- service name
- server port
- DB host, DB name, DB user
- email provider
- Google enabled or disabled
- Stripe enabled or disabled
- tracing enabled or disabled

Do not print:

- passwords
- JWT secrets
- OAuth client secrets
- Stripe secrets
- webhook secrets

## `.env` rule

`.env` is a local development convenience, not a reusable module
contract.

Recommended behavior:

- host bootstrap may auto-load `.env` in local development
- `.env` must never override already-set deployment environment
- reusable modules must not know whether config came from `.env`, YAML,
  or a secrets manager

This is why `.env` loading belongs in the template or business project,
not in shared modules.

## Recommended `go-modules` structure

Keep these responsibilities:

```text
go-modules/
  foundation/
    config/          config loader helper only
    ginx/
    httpresp/
    pgx/
    slog/
    tracing/
  modules/
    auth/
    billing/
    email/
    referral/
    user/
  stacks/
    saascore/        shared SaaS composition and its typed config
  templates/
    quickstart/      standard backend bootstrap shell
    quickstart-nextjs/
  docs/
    INTEGRATION.md
    SAASCORE_GUIDE.md
    CONFIG_STANDARD.md
```

### what should stay out of shared modules

- direct reads from `.env`
- direct reads from host YAML paths
- host deployment layout assumptions
- host-only project models
- host-only middleware and policies

## Recommended business project structure

Recommended backend layout:

```text
your-backend/
  cmd/
    server/
      main.go
  internal/
    bootstrap/       config load, validate, logger, tracing, DB, wiring
    hooks/           host business hooks for shared lifecycle events
    listener/        extra host listeners
    route/           host-only routes
  deploy/
    config.yaml
  .env
```

### bootstrap package should own

- load `.env` if present
- load YAML
- apply environment overrides
- validate final app config
- print sanitized config summary
- create DB / logger / tracing / stack dependencies

### bootstrap package should not own

- module business logic
- domain models unrelated to startup
- product features

## `quickstart` position

`templates/quickstart` should be treated as:

- the reference implementation of the standard bootstrap path
- the copyable starting point for greenfield services
- the example of how `AppConfig` maps into `saascore.Config`

It should not become a dumping ground for host-specific business code.

## Recommendation for `clawmesh-backend`

For a mature business project:

- keep config loading and validation in the backend repo
- keep shared auth, billing, email, referral, and user capability in
  `go-modules`
- minimize custom glue where `stacks/saascore` already defines the
  shared path

If a project fully matches the shared SaaS assumptions, it should move
toward:

- `stacks/saascore`
- template-aligned bootstrap
- host hooks instead of large custom glue packages

If a project does not match the shared assumptions, it should still keep
the bootstrap split, but compose `modules/*` directly instead of forcing
`saascore`.

## Decision summary

Use this rule of thumb:

- standard capability belongs in `go-modules`
- configuration source handling belongs in the host app
- standard bootstrap pattern belongs in templates and docs
- final app assembly belongs in the business project

That gives the best tradeoff between standardization and flexibility
across many SaaS projects.
