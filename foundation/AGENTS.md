# AGENTS.md — foundation/

Generic infrastructure shared by every service. Root `AGENTS.md` still applies.

- Allowed imports: stdlib, other `foundation/*` packages, and a small set of common
  libraries already in `go.mod` (gin, gorm, OpenTelemetry, go-redis, …). Never import
  `modules/*` or `templates/*`; `make purity-check` enforces this.
- Keep packages small and dependency-light. Prefer official/contrib OpenTelemetry
  instrumentation over hand-rolled code, but reject libraries that drag in unrelated
  heavy dependencies (e.g. the gorm OTel plugin pulled in ClickHouse/MySQL drivers).
- Public API is additive within a major version: add fields/options, never rename or
  change signatures. Update the package `CHANGELOG.md` under "Unreleased" and its `README.md`.
- Configuration comes in through a `Config` struct; zero values mean "off" or "sensible default".
  Do not read environment variables here, except standard `OTEL_*` handling in `tracing`.
- Every exported identifier has a doc comment. Coverage per package stays ≥ 70%.
- Tests that change globals (`slog.SetDefault`, `otel.Set*Provider`) restore them with `t.Cleanup`
  and do not run in parallel.
