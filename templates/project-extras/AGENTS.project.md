## Project-specific rules

Details: `docs/standards/PROJECT.md`, `docs/ARCHITECTURE.md`, `docs/OBSERVABILITY.md`.

### Layout
- `backend/` — Go service built from the go-modules quickstart template; depends on
  `github.com/brizenchi/go-modules` (`foundation/*`, `modules/*`). Change that library
  upstream, never copy it here.
- `frontend/` — Next.js app.

### Implementation
- HTTP responses use `foundation/httpresp`; handlers map domain errors in one
  `respondAppError` function. A stable error `reason` goes in `data` via `httpresp.Custom`.
- Product features go in `backend/internal/feature/<name>/` (copy `internal/feature/note`);
  provider selection in `backend/internal/platform`; event wiring in `backend/internal/bootstrap`.
- Frontend types live in `frontend/lib/api.ts`; update them with the endpoint.
- Outbound HTTP goes through the injected `platform.Config.HTTPClient`.
- Never edit an existing file in `backend/migrations/`; new tables/columns via GORM models,
  data changes via a new `YYYYMMDD_<desc>.sql`.

### Commands
```bash
cd backend && go test ./...
cd frontend && npm run verify
```

### Procedures
`.claude/skills/add-endpoint`, `.claude/skills/add-migration`, `.claude/skills/add-third-party` (`SKILL.md`).

### Deployment
Production deploys only from CI after all checks pass (`.github/workflows/ci.yml` → `deploy`). → `docs/DEPLOYMENT.md`
