# AGENTS.md

Rules for AI coding assistants (Codex, Claude Code, Cursor, …) working in this
repository. Subdirectories add their own `AGENTS.md`; the closest one wins on
conflict. Full standards live in `docs/standards/` — read the linked document
before working in that area.

## Repository

- One Go module (`github.com/brizenchi/go-modules`) plus `templates/quickstart`
  joined through `go.work`. Do not add nested `go.mod` files.
- Layers (see `docs/ARCHITECTURE.md`):
  - `foundation/*` — generic infrastructure; must not import `modules` or templates.
  - `modules/*` — reusable business modules (ports + adapters + events); modules never import each other.
  - `templates/quickstart` — composition root and product code; the only place that wires modules together.
  - `templates/quickstart-nextjs` — frontend.

## Must

- Pass `ctx` through every call; use `db.WithContext(ctx)` and `http.NewRequestWithContext(ctx, …)`.
- Log with `slog.InfoContext(ctx, "fixed message", "key", value)`. Never add
  `request_id`/`trace_id` by hand; never log secrets, tokens, request bodies or query strings.
- Handle each error once: wrap and return (`fmt.Errorf("…: %w", err)`), or handle it at the
  boundary. HTTP handlers map domain errors in one `respondAppError` function; unknown errors
  return a generic 500 message. → `docs/standards/CODE_STYLE.md`
- HTTP responses use `foundation/httpresp`; follow paths, status codes, pagination
  (`page`/`limit`/`items`/`total`), RFC 3339 UTC times and integer money. → `docs/standards/API_STANDARD.md`
- When an endpoint changes, update the frontend types in `templates/quickstart-nextjs/lib/api.ts` in the same change.
- Outbound HTTP in the template goes through the injected `platform.Config.HTTPClient`.
- Database: never edit an existing file in `templates/quickstart/migrations/`; add a new one.
  Parameterized queries only; list queries have `LIMIT` and a stable order. → `docs/standards/DATABASE.md`
- Check resource ownership on every read/write (`WHERE id = ? AND user_id = ?`). → `docs/standards/SECURITY_STANDARD.md`
- Add or update tests with every behaviour change; a bug fix starts with a failing test.
  No network in tests. → `docs/standards/TESTING.md`
- Fake secrets in tests must look fake: `sk_test_not-a-real-key`, `whsec_not-a-real-secret`.
- Public API changes in `foundation/*` or `modules/*` are additive only and update the package `CHANGELOG.md` ("Unreleased").

## Must not

- Commit real secrets or `.env` files; print secrets in output.
- Push to `main`, force-push, or rewrite pushed history.
- Add a dependency without a reason (see `docs/standards/CI_QUALITY.md#依赖管理`); keep `foundation/*` lightweight.
- Use `context.Background()` inside request handling, `panic` for control flow, or float types for money.

## Commits

Conventional Commits: `type(scope): subject` — types `feat fix perf refactor docs test build ci chore revert`,
scope = package or module (`tracing`, `auth`, `billing`, `quickstart`, `nextjs`, `docs`, …).
→ `docs/standards/GIT_WORKFLOW.md`

## Verify before finishing

```bash
make fmt && make test-race && make purity-check     # foundation + modules
cd templates/quickstart && go test ./...            # template
cd templates/quickstart-nextjs && npm run verify    # frontend (when touched)
```

Report failures as they are; do not claim success without running the checks.

## Where to look

| Topic | Document |
| --- | --- |
| Architecture and ownership | `docs/ARCHITECTURE.md`, `docs/adr/` |
| Configuration | `docs/CONFIG_STANDARD.md` |
| Logging, tracing, metrics, alerts | `docs/OBSERVABILITY.md` |
| All standards (index) | `docs/standards/README.md` |
| Step-by-step procedures (any assistant can follow them) | `.claude/skills/add-endpoint`, `add-migration`, `add-third-party` (`SKILL.md`) |
