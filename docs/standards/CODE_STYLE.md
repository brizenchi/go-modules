# Code style (all languages)

Language-specific rules: [Go](./GO.md) [Node / TypeScript](./NODE.md).
Project-specific conventions: [PROJECT.md](./PROJECT.md).

## Principles

- Read the surrounding code first and **stay consistent** with it: naming, comment
  density and error handling match the rest of the module.
- Leave what tools can check (formatting, lint) to the tools; it is not discussed in review.
- Prefer the standard library and existing dependencies; a new dependency needs a stated
  reason (see [CI_QUALITY.md](./CI_QUALITY.md#dependencies)).
- No abstractions "for later": add interfaces and extension points only when there is a
  second implementation or an external dependency to isolate.

## Naming across layers

| What | Rule | Example |
| --- | --- | --- |
| Database tables and columns | snake_case; plural table names | `invoices`, `created_at` |
| JSON fields | snake_case (one style across the whole API) | `current_period_end` |
| URL paths | lowercase, words joined with `-` | `/api/v1/credit-transactions` |
| Environment variables | UPPER_SNAKE_CASE with a project prefix | `APP_DB_HOST` |
| Log fields | snake_case; one name per meaning | `request_id`, `user_id`, `duration_ms` |
| Metrics | domain prefix, stable names | `app.user.logins` |

## Error handling

### Handle each error once

An error is either **returned** to the caller (with added context) or **handled here**
(logged, degraded, turned into a response) — never both, or the same error shows up in
the logs several times.

### Responsibilities by layer
| Layer | Does |
| --- | --- |
| Data access, external service adapters | adds context to low-level errors and returns them; turns "not found" and similar cases into domain errors |
| Business logic | branches on domain errors; adds context to anything else and returns it |
| **Boundary** (HTTP handler, message consumer, scheduled job) | maps domain errors to responses or retry policies; logs at ERROR **only for unknown errors** |

### At the boundary
- Map errors in one place (for example a single `respondError` function), not in every handler.
- Unknown errors never expose internals (SQL, stack traces, third-party responses) to the
  caller — only a fixed message and the request ID.
- Client errors (invalid input, not signed in) are not logged at ERROR.

### Exceptions and panics
- Do not use panics or exceptions for expected business outcomes.
- At startup, unrecoverable configuration errors may exit the process.
- Uncaught exceptions while handling a request are caught by the framework's central
  handler: return 500, log the full stack trace, and include the request ID.

## Context propagation

- The request context (cancellation, deadline, trace, request ID) is passed from the entry
  point all the way to the database and outbound calls.
- Log with the context so `request_id` and `trace_id` are added automatically.
- Never replace the caller's context with a new empty one while handling a request.

## Concurrency

- Every background task has a clear end, and an owner that waits for it and can cancel it.
- Shared state is protected by locks or replaced by message passing.
- Tests run with the race detector (`-race` in Go).

## Comments

- Public functions and types have doc comments.
- Comments explain **why**, not what the code already says.
- `TODO`s name an owner or an issue: `TODO(name): ... #123`.

## Configuration and constants

- Configuration is injected as structured objects; business code does not read environment variables.
- No magic numbers: timeouts, retry counts and limits are constants or configuration.
- Secrets come only from environment variables or a secret manager (see [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#secret-management)).
