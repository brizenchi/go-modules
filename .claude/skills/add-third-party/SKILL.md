---
name: add-third-party
description: Integrate a new external service or provider (OAuth, email, payment, storage, any HTTP API) as an adapter with tracing, logging, timeouts, secrets handling and alerts. Use when adding or replacing a third-party SDK/API call.
---

# Integrate a third-party service

Standards: `docs/ARCHITECTURE.md#服务商替换`, `docs/standards/SECURITY_STANDARD.md`,
`docs/OBSERVABILITY.md#出站调用`, `docs/standards/CI_QUALITY.md#依赖管理`.

## 1. Shape
- Find the port the provider fulfils (`auth/port.IdentityProvider`, `email/port.Sender`,
  `billing/port.Provider`, …). Implement it as a new adapter; do not change the port for one vendor.
- Reusable across SaaS → `modules/<module>/adapter/<vendor>`; product-specific →
  `templates/quickstart/internal/...`.
- Prefer the vendor's official SDK only if it accepts a custom `*http.Client` and a `context.Context`;
  otherwise call the HTTP API directly.

## 2. Adapter rules
- `Config` struct with an optional `HTTPClient *http.Client`; fall back to a client with a
  timeout when nil. Never use `http.DefaultClient` (no timeout).
- Pass `ctx` to every call (`http.NewRequestWithContext`, SDK params' `Context`).
- Map vendor errors to the module's domain errors; do not leak vendor messages to API clients.
- Do not log request/response bodies, tokens or API keys.
- Webhooks: verify signatures first, deduplicate by event id, re-fetch authoritative data.

## 3. Wiring (template)
- Select it in `internal/platform/*_provider.go`, passing `cfg.HTTPClient` (traced + logged by
  `foundation/httpx`).
- Config keys: non-secret defaults in `deploy/config.yaml.example`; secrets only via
  `APP_…` environment variables; validate required values at startup (fail fast on unknown
  provider names).
- Document setup in `docs/SETUP_ZH.md` and configuration in `docs/CONFIG_STANDARD.md`.

## 4. Tests
- `httptest.Server` fakes the vendor: success, 4xx, 5xx, timeout, malformed body.
- Assert `ctx` cancellation stops the call and that the injected client is used.
- Fake credentials look fake: `<prefix>_not-a-real-key`.

## 5. Operations
- The `OutboundDependencyErrors` alert (`deploy/alerts/rules.yaml`) covers the new host automatically.
- Add a status/fallback note to `docs/standards/INCIDENT_RESPONSE.md` if the vendor is critical.

## 6. Verify
```bash
make fmt && make test-race && make purity-check
cd templates/quickstart && go test ./...
```
Justify any new dependency in the PR (maintenance, license, transitive deps).
