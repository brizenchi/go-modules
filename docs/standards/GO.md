# Go

General rules: [CODE_STYLE.md](./CODE_STYLE.md).

## Tools

| Tool | Checks | Where it runs |
| --- | --- | --- |
| `gofmt -s` | formatting | pre-commit hook, CI `go-* / check` |
| `go vet` | mistakes the compiler misses | CI `go-* / check` |
| `go test -race` | tests and data races | CI `go-* / check` |
| `golangci-lint` | errcheck, staticcheck, gosec and more; uses `.golangci.yml` when present | CI `go-* / lint` |
| `govulncheck` | vulnerable dependencies | CI `go-* / vuln` (does not block merging) |

## Naming

| What | Rule | Example |
| --- | --- | --- |
| Packages | short lowercase words, no underscores, not plural | `httpx`, `billing` |
| Files | snake_case, named after their responsibility | `access_log.go`, `provider_test.go` |
| Exported identifiers | MixedCaps; initialisms in one case | `UserID`, `HTTPClient` |
| Interfaces | describe behaviour | `UserStore`, `Sender` |
| Error variables | `Err` prefix; message starts with the package name | `ErrNotFound = errors.New("billing: invoice not found")` |
| Constructors | `New` / `NewXxx`, taking a `Config` struct | `NewClient(Config{...})` |

## Errors

- Wrap with context: `fmt.Errorf("load invoice %s: %w", id, err)`.
- Compare with `errors.Is` / `errors.As`, never by string.
- Each package defines its sentinel errors in one file (for example `errors.go`).
- At the HTTP boundary, one function maps errors to responses; its `default` branch logs once with `slog.ErrorContext` and returns a fixed message.
- No panics in business code; panics during a request are caught by recovery middleware.

## context

- `ctx context.Context` is the first parameter and is never stored in a struct.
- Pass it all the way down: `db.WithContext(ctx)`, `http.NewRequestWithContext(ctx, …)`.
- Background work that must outlive the request uses `context.WithoutCancel(ctx)`, which keeps the trace.

## Logging

- Use the standard library's `log/slog` with JSON output.
- Business code logs with `slog.InfoContext(ctx, "fixed message", "key", value)`: a fixed phrase, with the varying values as fields.
- Never log secrets, tokens or full request bodies.

## Concurrency

- Every goroutine has a clear end; wait for it with `errgroup` or `sync.WaitGroup` and cancel it through ctx.
- Protect shared state with locks; tests run with `-race`.

## Testing

- Table-driven tests with `t.Run`; names describe the scenario: `TestRefund_RejectsDoubleRefund`.
- Tests that change global state (`slog.SetDefault` and similar) restore it with `t.Cleanup` and do not use `t.Parallel()`.
- Fake outbound HTTP with `httptest.Server`; use an in-memory database or test containers.
- Prefer the standard library's `testing` package.

## Dependencies and APIs

- Public APIs change additively; breaking changes need a new major version (`/v2`).
- Run `go mod tidy` and commit `go.mod` and `go.sum`.
