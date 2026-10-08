# Project conventions

> This file belongs to the project. The shared standards are maintained by
> [keel](https://github.com/brizenchi/keel); this file records how this project applies them.
> The project was created from the go-modules quickstart template.

## Architecture

- `backend/`: the Go backend, composition root and product code. Shared capabilities come from
  `github.com/brizenchi/go-modules` (`foundation/*` for technical capabilities, `modules/*` for sign-in,
  payments, email and referrals) — **change them upstream, do not copy them in**.
- `frontend/`: the Next.js frontend.
- Layers and ownership: [ARCHITECTURE.md](../ARCHITECTURE.md).

## Layout

| Directory | Contents |
| --- | --- |
| `backend/internal/feature/*` | product features (see `note`: `note.go`, `repository.go`, `service.go`, `handler.go`) |
| `backend/internal/platform` | provider selection, module wiring, route mounting |
| `backend/internal/bootstrap` | startup, event subscriptions, business callbacks, background jobs |
| `backend/migrations` | SQL migrations run by hand |
| `backend/deploy` | example configuration, Alloy log shipping, alert rules |
| `frontend/lib` | API client (`api.ts`), state, environment variables |

## Local development

```bash
.keel/bin/keel hooks
cd backend && cp .env.example .env && go run ./cmd/quickstart
cd frontend && npm ci && npm run dev
```

## Implementation conventions

- APIs: responses use `foundation/httpresp`; a `reason` goes in `data` via `httpresp.Custom`; routes are mounted on the `Public`, `User` and `Admin` groups.
- Errors: a module's sentinel errors live in `domain/errors.go`; the HTTP layer maps them in one `respondAppError` function.
- Database: new tables and columns go through GORM `AutoMigrate`; changes to existing data go in `backend/migrations/YYYYMMDD_<description>.sql`, run by hand after a backup.
- Outbound calls use the injected `platform.Config.HTTPClient`.
- Frontend: every request goes through `frontend/lib/api.ts`; errors are `ApiError` (`status`, `reason`, `requestId`), and 5xx errors show the request ID.
- Logging and observability: [OBSERVABILITY.md](../OBSERVABILITY.md).

## Deployment

Once every CI check passes, the `deploy` job deploys through Dokploy; configuration and the release
acceptance checklist are in [DEPLOYMENT.md](../DEPLOYMENT.md).

## Project-specific checks

| Check | What it does |
| --- | --- |
| `observability-config` | alert rule tests, Alloy configuration check |
| `deploy` | deploys after a push to `main` once all checks pass |
