# AGENTS.md

Rules for AI coding assistants (Codex, Claude Code, Cursor, …) working in
go-modules. Subdirectories may add their own `AGENTS.md`; the closest one wins
on conflict. Full standards: `docs/standards/` — read the linked document before
working in that area.
Project architecture and commands: `docs/standards/PROJECT.md`.

The sections above "Project-specific rules" are maintained by keel
(https://github.com/brizenchi/keel) and updated with `keel update`; edit the
project section freely.

## Must

- Pass the request context (cancellation, deadline, trace) through every call down
  to the database and outbound HTTP.
- Use structured logging with a fixed message and key/value fields; let the logger
  add `request_id`/`trace_id` from context. Never log secrets, tokens, request bodies
  or query strings.
- Handle each error once: add context and return it, or handle it at the boundary
  (HTTP handler, consumer, job). Map known errors to responses in one place; unknown
  errors log once at ERROR and return a generic message. → `docs/standards/CODE_STYLE.md`
- HTTP APIs: `/api/v1` paths, one response envelope, correct status codes, a stable
  `reason` for errors the client must branch on, `page`/`limit`/`items`/`total`
  pagination, RFC 3339 UTC times, integer money in minor units. → `docs/standards/API_STANDARD.md`
- When an API changes, update its callers' types/clients in the same change.
- Database: never edit an applied migration; add a new one. Parameterized queries only;
  list queries have `LIMIT` and a stable order; expand → migrate → contract for
  breaking schema changes. → `docs/standards/DATABASE.md`
- Check resource ownership on every read/write; return 404 for other users' resources.
  → `docs/standards/SECURITY_STANDARD.md`
- Add or update tests with every behaviour change; a bug fix starts with a failing test.
  No network access in tests. → `docs/standards/TESTING.md`
- Fake secrets in tests must look fake: `sk_test_not-a-real-key`, `whsec_not-a-real-secret`.

## Go
- `gofmt -s`; `go vet`; `go test -race`. `ctx context.Context` is the first parameter,
  never stored in structs; use `db.WithContext(ctx)` and `http.NewRequestWithContext`.
- Wrap errors with `fmt.Errorf("…: %w", err)`; compare with `errors.Is/As`; log with
  `slog.InfoContext(ctx, …)`. → `docs/standards/GO.md`

## Node / TypeScript
- Strict TypeScript, no `any`. All backend calls go through the shared API client; never
  call `fetch` from components. Show users messages derived from `status`/`reason`, never
  the raw server message; show the request id for 5xx. → `docs/standards/NODE.md`
- Browser-exposed env vars (`NEXT_PUBLIC_*`, `VITE_*`) never contain secrets.

## Must not

- Commit real secrets or `.env` files; print secrets in output.
- Push to `main`, force-push, rewrite pushed history, or bypass hooks with `--no-verify`.
- Add a dependency without a stated reason (`docs/standards/CI_QUALITY.md#依赖管理`).
- Use float types for money or panic/exceptions for expected business outcomes.

## Commits and pull requests

Conventional Commits: `type(scope): subject` (≤ 72 chars) — types
`feat fix perf refactor docs test build ci chore revert`. The PR title becomes the squash
commit. → `docs/standards/GIT_WORKFLOW.md`

## Before finishing

Run the checks for the parts you changed (listed in
`.github/pull_request_template.md`) and report failures as they are; never claim
success without running them.

## Project-specific rules

Details: `docs/standards/PROJECT.md`, `docs/ARCHITECTURE.md`, `docs/OBSERVABILITY.md`.

### Repository
- One Go module (`github.com/brizenchi/go-modules`) plus `templates/quickstart`, joined
  through `go.work`. Do not add nested `go.mod` files.
- Layers:
  - `foundation/*` — generic infrastructure; must not import `modules` or templates (`make purity-check`).
  - `modules/*` — reusable business modules (ports + adapters + events); modules never import each other.
  - `templates/quickstart` — composition root and product code; the only place that wires modules together.
  - `templates/quickstart-nextjs` — frontend.
- Public API changes in `foundation/*` or `modules/*` are additive only and update the package
  `CHANGELOG.md` ("Unreleased").

### Implementation
- HTTP responses use `foundation/httpresp`; HTTP handlers map domain errors in one
  `respondAppError` function (see `modules/auth/http/handler.go`). A stable error `reason`
  goes in `data` via `httpresp.Custom`.
- Frontend types live in `templates/quickstart-nextjs/lib/api.ts`; update them with the endpoint.
- Outbound HTTP in the template goes through the injected `platform.Config.HTTPClient`.
- Never edit an existing file in `templates/quickstart/migrations/`; new tables/columns go through
  GORM models (AutoMigrate), data changes through a new `YYYYMMDD_<desc>.sql`.
- Logging, tracing and metrics come from `foundation/slog` / `foundation/tracing`; never add
  `request_id` / `trace_id` by hand.

### Commands
```bash
make fmt && make test-race && make purity-check      # foundation + modules
cd templates/quickstart && go test ./...             # backend template
cd templates/quickstart-nextjs && npm run verify     # frontend
```

### Procedures (Claude Code skills; any assistant can read them)
`.claude/skills/add-endpoint`, `.claude/skills/add-migration`, `.claude/skills/add-third-party` (`SKILL.md`).

### Deployment
Production deploys only from CI after all checks pass (`.github/workflows/ci.yml` → `deploy`).
Never trigger deployments manually unless asked. → `docs/DEPLOYMENT.md`
