---
name: add-endpoint
description: Add or change an HTTP endpoint or product feature in the quickstart backend (and its frontend types) following the repository's API, layering, security and testing standards. Use when asked to add an API, route, handler, or a new feature under templates/quickstart/internal/feature.
---

# Add an HTTP endpoint / feature

Standards: `docs/standards/API_STANDARD.md`, `docs/standards/CODE_STYLE.md`,
`docs/standards/SECURITY_STANDARD.md`, `docs/standards/TESTING.md`.

## 1. Decide where it lives
- Product feature → `templates/quickstart/internal/feature/<name>/`. Copy the layout of
  `internal/feature/note`: `<name>.go` (entity, `New`, `Models`, `Register`),
  `repository.go` (only layer touching `*gorm.DB`), `service.go` (rules, no gin/SQL),
  `handler.go` (HTTP in/out only).
- Change to a shared module (`modules/*/http`) only if every SaaS needs it; keep the module's
  public API additive and update its `CHANGELOG.md`.

## 2. Route
- Path under `/api/v1`, lowercase, `-` between words, plural collections.
- Mount on the right group in `Register(g hostapi.Groups)`: `g.Public`, `g.User` or `g.Admin`.
  User data is never on `Public`.
- Register the feature in `internal/http/host_routes.go`; add its models to
  `internal/bootstrap/host_migrate.go` (`hostModels`).

## 3. Handler
- Bind and validate input (lengths, ranges, enums); invalid → `httpresp.BadRequest`.
- Pass `c.Request.Context()` down; never `context.Background()`.
- Respond with `httpresp.OK(c, data)`; JSON fields snake_case; times RFC 3339 UTC; money as
  integer minor units; lists return `{items, total, page, limit}` with `limit` ≤ 100.
- Map errors in one `respondAppError`: known domain errors → specific status (+ `data.reason`
  like `NOTE_NOT_FOUND` when the frontend must branch); unknown → log once with
  `slog.ErrorContext` and `httpresp.InternalError` with a generic message.
- Writes that must not run twice require `Idempotency-Key`.

## 4. Service / repository
- Every query: `db.WithContext(ctx)`; filter user data by owner (`WHERE id = ? AND user_id = ?`);
  return 404 for other users' resources.
- Stable ordering + `LIMIT` on lists; parameterized queries only.
- Multi-step writes in a transaction; no external HTTP calls inside it.
- New tables/columns via the GORM model; changing existing data → new file in `migrations/`
  (see the `add-migration` skill).

## 5. Frontend contract
- Add/adjust request and response types and a wrapper function in
  `templates/quickstart-nextjs/lib/api.ts` in the same change; add a test in `tests/api.test.ts`
  if request building or error handling changed.

## 6. Tests (required)
- `service_test.go`: rules, edge cases, ownership, idempotency.
- Handler/route test with `httptest`: status codes, envelope, auth group (401 without token,
  404 for another user's resource).
- SQLite in-memory DB; no network.

## 7. Verify
```bash
cd templates/quickstart && go test ./...
cd templates/quickstart-nextjs && npm run verify   # if frontend changed
make fmt
```
Commit as `feat(<feature>): <subject>`.
