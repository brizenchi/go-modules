# AGENTS.md — templates/quickstart (backend)

The composition root and product code of a SaaS. Root `AGENTS.md` still applies.

## Where code goes

| Change | Location |
| --- | --- |
| New product feature | `internal/feature/<name>/` — copy `internal/feature/note` (`note.go` wiring, `repository.go` DB only, `service.go` rules only, `handler.go` HTTP only) |
| Register its routes / tables | `internal/http/host_routes.go`, `internal/bootstrap/host_migrate.go` |
| React to module events (signup, payment, referral) | `internal/bootstrap/subscriptions.go`, `internal/bootstrap/host_hooks.go` |
| Choose or replace a provider (OAuth, email, payment) | `internal/platform/*_provider.go` |
| Product configuration | `internal/hostcfg` (not `platform.Config`) |
| User fields | `internal/user` + a new SQL file in `migrations/` when existing data changes |
| Business metrics / spans | event subscriptions in `internal/bootstrap`, using `otel.Meter("quickstart")` |

## Rules

- Mount routes on the right group: `Public`, `User` (auth required) or `Admin`. Never put a
  user-scoped route on `Public`.
- Every query filters by owner for user data (`WHERE id = ? AND user_id = ?`).
- Third-party HTTP calls use `platform.Config.HTTPClient` (traced and logged).
- Do not edit files under `migrations/` that may have run anywhere; add a new `YYYYMMDD_<desc>.sql`.
- Configuration keys are added to `deploy/config.yaml.example` (the schema test reads it) and
  documented in `docs/CONFIG_STANDARD.md` when user-facing.
- Do not edit files marked `TEMPLATE-OWNED` unless the change is meant for every SaaS.
- Run `go test ./...` in this directory before finishing.
