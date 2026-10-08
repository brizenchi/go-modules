# Testing

## Layers

| Kind | Scope | Notes |
| --- | --- | --- |
| Unit | pure logic: business rules, calculations, validation, mapping | the most numerous and the fastest; no network, no database |
| Data access | SQL, ORM behaviour | in-memory database or test containers |
| API | routing, authentication, validation, response shape, status codes | start the HTTP handler in-process |
| External service adapters | calls to third-party APIs, parsing responses | a local fake server instead of the real service |
| End-to-end / release acceptance | critical user journeys | few; a manual checklist or scripts |

Keep logic out of HTTP and database code where possible, so unit tests can cover it.

## When tests are required

- New public functions, endpoints and business rules.
- **A bug fix starts with a test that reproduces the bug.**
- Money, quotas, permissions and idempotency: cover the normal case, boundaries, repeated
  requests and unauthorised access.
- Validation of new configuration options.

## How to write them

- Test names describe the scenario and the expectation; use table-driven or parameterised tests for several inputs.
- Tests are independent: no reliance on execution order, and global state is restored afterwards.
- Failed assertions print the actual value.
- Inject time and randomness when they must be deterministic; never sleep to wait.

## External dependencies

- **Tests do not access the network**; third-party APIs are always faked.
- Tests that need a specific database feature are behind a switch, skipped by default, and run in a dedicated CI job.

## Test data

- Fake secrets **look fake at a glance**: `sk_test_not-a-real-key`, `whsec_not-a-real-secret`.
  Realistic-looking fakes are blocked by GitHub push protection and gitleaks.
- Email addresses use `example.com` or `example.test`.
- Never use real user data or production data.

## Running them

Go:

```bash
gofmt -s -l . && go vet ./... && go test -race ./...
```

Node (in the project directory):

```bash
npm run lint && npm test && npm run build    # or pnpm / yarn
```

The matching CI checks are listed in [CI_QUALITY.md](./CI_QUALITY.md#what-ci-runs).

## Coverage

- Shared libraries and core business modules: at least **70%**, and pull requests do not lower it.
- Application code: cover the critical paths; no fixed percentage.
- Coverage is a signal; what matters is whether the important branches are tested.
