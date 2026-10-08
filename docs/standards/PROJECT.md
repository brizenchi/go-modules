# go-modules: project conventions

> This file belongs to the project. The shared standards are maintained by
> [keel](https://github.com/brizenchi/keel); this file records how go-modules applies them.

## Architecture

- Layers and ownership: [ARCHITECTURE.md](../ARCHITECTURE.md); decision records: [adr/](../adr).
- `foundation/*` holds generic technical capabilities and never imports `modules` or the templates (checked by `make purity-check`).
- `modules/*` are reusable business modules, layered `http → app → domain / port` and `adapter → port / domain`.
  **Modules never import each other**; they cooperate through events, which the template subscribes to.
- `templates/quickstart` is the composition root and product code; `templates/quickstart-nextjs` is the frontend.
- Public APIs of `foundation/*` and `modules/*` change additively within a major version, with the package
  `CHANGELOG.md` updated; versioning rules: [VERSIONING.md](../../VERSIONING.md).

## Layout

| Directory | Contents |
| --- | --- |
| `foundation/` | logging, tracing, HTTP client, database, Redis, configuration and other shared capabilities |
| `modules/` | auth, billing, email, referral |
| `templates/quickstart/` | backend template: `internal/feature/*` (product features), `internal/platform` (provider selection), `internal/bootstrap` (startup and event subscriptions) |
| `templates/quickstart-nextjs/` | frontend template |
| `docs/` | architecture, configuration, observability, deployment, standards |
| `.claude/skills/` | AI procedures: adding an endpoint, a database migration, a third-party integration |

## Local development

```bash
make hooks                                     # install the pre-commit checks
make fmt && make test-race && make purity-check
cd templates/quickstart && cp .env.example .env && go run ./cmd/quickstart
cd templates/quickstart-nextjs && npm ci && npm run dev
```

Signing in locally: `.env.example` sets `APP_EMAIL_PROVIDER=log` and `APP_AUTH_EMAIL_DEBUG=true`, so the
verification code is shown directly in the sign-in form.

## Implementation conventions

### APIs ([API_STANDARD.md](./API_STANDARD.md))
- Responses go through `foundation/httpresp`: `OK`, `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`,
  `Conflict`, `TooManyRequests`, `InternalError`; use `Custom` for 503 and when a `reason` is needed:

  ```go
  httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, msg, gin.H{"reason": "AUTH_INVALID_CODE"})
  ```
- Routes are mounted on the `Public`, `User` and `Admin` groups of `hostapi.Groups` according to access.
- List endpoints return `{items, total, page, limit}`, with `limit` at most 100.
- Admin writes that must be idempotent require `Idempotency-Key` (see `internal/feature/operations/settings.go`).

### Error handling ([CODE_STYLE.md](./CODE_STYLE.md#error-handling))
- Each module defines its sentinel errors in `domain/errors.go`.
- The HTTP layer maps errors in a single `respondAppError` function, as in `modules/auth/http/handler.go`;
  its `default` branch logs once with `slog.ErrorContext` and returns a fixed message.

### Database ([DATABASE.md](./DATABASE.md))
- New tables, columns and indexes: change the GORM model; `AutoMigrate` runs at startup
  (`internal/platform/migrate.go`, `internal/bootstrap/host_migrate.go`).
- Changing existing data, changing types, dropping columns: add `templates/quickstart/migrations/YYYYMMDD_<description>.sql`
  and **run it by hand after a backup**.
- Main entities (`users`) use `varchar(36)` UUIDs; child records and ledgers (`notes`, credit transactions) may use auto-increment `bigint`.
- Tests use an in-memory SQLite database; SQL logs and traces omit parameter values by default (`foundation/pgx`).

### Outbound calls
- Every third-party HTTP call in the template uses the injected `platform.Config.HTTPClient` (with tracing, metrics and logging).
- Adapters accept an optional `HTTPClient` and pass `ctx` to every SDK call (`Context` in Stripe params).

### Frontend ([NODE.md](./NODE.md))
- Every request goes through `apiRequest` in `lib/api.ts`; the types live there too and change in the same pull request as the backend.
- Failures throw `ApiError` with `status`, `code`, `message` (never shown directly), `reason` and `requestId`.
- User-facing messages follow `ConsoleError` in `components/console-kit.tsx` and `describeRequestFailure` in
  `lib/request-state.ts`; 5xx errors show the request ID.
- Idempotent writes generate a key with `newIntentKey()` and reuse it on retry.
- Text uses `t({ en, zh })` (`lib/i18n.tsx`); environment variables are read only in `lib/env.ts`; session state only through `lib/auth.ts`.

### Logging and observability
- Log with `slog.*Context(ctx, …)`; `foundation/slog` adds `request_id` and `trace_id` and redacts sensitive fields.
- Fields, levels, traces, metrics and alerts: [OBSERVABILITY.md](../OBSERVABILITY.md).
- Logs in tests: `flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})`; traces in tests: `tracetest.NewSpanRecorder()`.

## Deployment

- **Deploys happen only after all CI checks pass**: the `deploy` job in `.github/workflows/ci.yml` calls the Dokploy API (`scripts/deploy-dokploy.sh`).
- Environments, one-time setup, rollback, post-release checks and the release acceptance checklist: [DEPLOYMENT.md](../DEPLOYMENT.md).

## Project-specific checks

In addition to keel's checks (`keel / …`), `.github/workflows/ci.yml` runs:

| Check | What it does |
| --- | --- |
| `template-quickstart` | Linux build of the template, plus a detached build outside the workspace (`scripts/verify-quickstart-release.sh`) |
| `go mod tidy` | `go.mod` and `go.sum` are tidy |
| `pkg purity` | shared packages do not import host code |
| `observability-config` | alert rule tests (promtool) and the Alloy configuration check |
| `deploy` | deploys after a push to `main` once all of the above pass |

These checks are also listed at the end of `.keel/required-checks.txt`, so `keel github` makes them required.

## Other

- `.gitleaksignore` lists fingerprints of fake test keys committed before gitleaks was introduced; new fake keys must use the `*_not-a-real-key` form.
- When `templates/quickstart` is copied into a new project with `make init-quickstart`, keel is installed too.
