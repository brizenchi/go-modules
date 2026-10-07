# AGENTS.md — modules/

Reusable business modules (`auth`, `billing`, `email`, `referral`). Root `AGENTS.md` still applies.

- Layering inside a module: `http → app → domain / port`; `adapter → port / domain`;
  `New()` wires the injected ports. See `docs/ARCHITECTURE.md`.
- A module never imports another module. Cross-module behaviour goes through events that
  the template subscribes to (`templates/quickstart/internal/bootstrap/subscriptions.go`).
- Modules do not own the host `User`: depend on ports such as `auth/port.UserStore`.
- Domain errors are sentinels in `domain/errors.go` (`ErrXxx = errors.New("<module>: …")`).
  HTTP handlers map them in a single `respondAppError`; unknown errors log once and return a
  generic 500 message.
- New third-party providers are new adapters implementing an existing port; do not bend the
  port to one provider. Adapters accept an optional `HTTPClient *http.Client` and pass `ctx`
  to every SDK call.
- Modules do not import OpenTelemetry; observability comes from the injected HTTP client,
  the DB wrapper and the template's middleware.
- Public API changes are additive and recorded in the module `CHANGELOG.md`.
