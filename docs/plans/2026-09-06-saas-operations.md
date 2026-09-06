# SaaS Operations Implementation Plan

> Execute this plan task by task in the current workspace; preserve the existing, uncommitted user-console implementation that this work extends.

**Goal:** Deliver a usable operator console, credit lifecycle and paid export example, secure uploads, editable business settings, and bilingual public content/SEO for the free SaaS starter.

**Architecture:** Keep reusable auth/billing/referral modules intact. Add host-owned Go features and protected API routes, reuse the existing user/admin identity groups, extend the Next.js console with typed clients, and keep public content local and editable. Independent backend credit, backend operations, and public-content work runs alongside root-owned frontend/integration work.

**Tech Stack:** Go, Gin, GORM, PostgreSQL deployment / isolated SQLite tests, Next.js 15, React 18, TypeScript, CSS modules, Node tests.

## Accepted scope and constraints

- Include users, subscriptions, payment records, referrals, credit grants/refunds, audit history and business settings in the operator console. Authorization lives on the server; client role checks only control navigation.
- Credit grants, expiry, consumption and refund require stable idempotency, ownership, insufficient-balance protection and an auditable reason. Existing balances are preserved through a supplied explicit migration, never an automatic repair.
- A note-to-Markdown export is the paid business example. Show its actual configured cost, deduct once per export request and let retries retrieve the same output without another charge.
- Image uploads validate actual file content and size and enforce ownership; production object-storage configuration is explicit.
- Public content includes bilingual docs/pricing/blog/updates/contact/privacy/terms, search and metadata/sitemap/robots. Retain existing locale routing and OAuth URLs.
- Defer DAU/retention dashboards, AI generation, additional payment providers and Cloudflare-specific hosting.
- Do not migrate or repair any configured database, send real email, execute live payments, publish, or push. Persist migration artifacts; use isolated test databases and local preview fixtures.

## Task 1 — Credit lifecycle

Files: internal/user credit models/service/tests; internal/feature/credits HTTP/tests; dedicated migration SQL.

1. Cover concurrent spend, expiry, retry conflicts and refunds with SQLite tests.
2. Preserve GrantCredits signature; add balance, paginated ledger and controlled admin writes.
3. Add note ownership checks and atomic paid exports with persisted outputs.
4. Supply versioned legacy migration, compatibility notes and exact API DTOs.

## Task 2 — Operations, settings and uploads

Files: internal/feature/operations; internal/hostcfg upload config; dedicated migration SQL/tests.

1. Implement bounded, searchable lists and honest payment projections from stored billing events.
2. Implement activated-referral reward replay with reason/audit; reject pending invitations.
3. Add allowlisted public business settings and administrative writes with validation.
4. Add bounded image upload/read APIs, real MIME validation and owner checks.
5. Cover ordinary-user denial, invalid updates, pagination and private-file access.

## Task 3 — Customer and operator interfaces

Files: lib/operations-api.ts, reusable session/resource controls, app/admin, app/credits, app/notes, app/files and CSS modules; shared navigation integration.

1. Build typed clients and request/session race guards.
2. Build a separate administrator navigation with searchable lists, pagination, explicit failures and deliberate grant/refund/reconciliation forms.
3. Build balance/ledger, note editor/export and owned-file upload interfaces.
4. Preserve existing user console, add appropriate links, and keep new user-facing strings bilingual.

## Task 4 — Public content and configuration

Files: public content routes/data, docs/pricing, site-settings helper, metadata, sitemap/robots, CONTENT.md.

1. Implement editable bilingual local articles, search and changelog.
2. Add configurable contact links and starter privacy/terms content.
3. Exclude private routes from discovery; generate metadata only for real public routes.
4. Connect saved public brand/support settings to public and console UI.

## Task 5 — Integrate and verify

Files: host_routes.go, host_migrate.go, env/config examples, README/runbook and focused integration tests.

1. Register new host features and schema models without executing configured-environment migrations.
2. Run Go tests including race checks, frontend tests, lint and production build.
3. Exercise user/admin journeys against isolated local fixtures; verify permissions, invitation replay, export idempotency and negative cases.
4. Inspect frontend navigation, loading/error states and layout in the browser.
5. Record precise completed validation, deployment prerequisites and any live-service checks not performed.
