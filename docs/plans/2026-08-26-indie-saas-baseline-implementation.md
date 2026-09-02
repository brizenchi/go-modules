# Indie SaaS Baseline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use executing-plans to implement this plan task-by-task.

**Goal:** Make quickstart a minimal reliable auth, Stripe Checkout, email and logging baseline that can be reused across roughly ten indie SaaS products.

**Architecture:** Keep shared provider logic in `modules/*` and product policy in the copied template. Use Stripe's retry window plus the existing persisted billing event record, error-propagating synchronous listeners, subscription snapshot upserts and a host-owned idempotent credit ledger; avoid queues and distributed infrastructure.

**Tech Stack:** Go 1.25, Gin, GORM/PostgreSQL, Stripe, Resend, Next.js 14, TypeScript.

---

### Task 1: Harden authentication tokens and Google identity

**Files:**
- Modify: `modules/auth/adapter/jwt/signer.go`
- Test: `modules/auth/adapter/jwt/signer_test.go`
- Modify: `modules/auth/adapter/google/provider.go`
- Test: `modules/auth/adapter/google/provider_test.go`

**Steps:** Add failing issuer/token-type/verified-email tests, implement algorithm and issuer validation plus explicit access/ticket claim types, then run the two adapter test packages.

### Task 2: Validate production configuration before touching the database

**Files:**
- Modify: `templates/quickstart/internal/bootstrap/config.go`
- Test: `templates/quickstart/internal/bootstrap/config_schema_test.go`
- Modify: `templates/quickstart/deploy/config.yaml.example`
- Modify: `templates/quickstart/.env.example`

**Steps:** Add failing validation tests, implement production-only secret/provider/OAuth/Stripe/CORS checks, invoke validation during config loading, then run bootstrap tests.

### Task 3: Close the Stripe subscription and credit reliability loop

**Files:**
- Modify: `modules/billing/port/eventbus.go`
- Modify: `modules/billing/adapter/eventbus/inproc.go`
- Test: `modules/billing/adapter/eventbus/inproc_test.go`
- Modify: `modules/billing/port/repository.go`
- Modify: `modules/billing/app/webhook.go`
- Test: `modules/billing/app/webhook_test.go`
- Modify: `modules/billing/billing.go`
- Modify: `templates/quickstart/internal/platform/billing_provider.go`
- Modify: `templates/quickstart/internal/user/model.go`
- Modify: `templates/quickstart/internal/user/repository.go`
- Test: `templates/quickstart/internal/user/repository_test.go`
- Modify: `templates/quickstart/internal/platform/migrate.go`
- Modify: `templates/quickstart/internal/bootstrap/host_hooks.go`

**Steps:** Make listener failures observable, persist snapshots before marking events processed, wire the repository, implement idempotent credit grants by provider event ID, and cover retry/duplicate behavior with tests.

### Task 4: Remove unsupported payment surface and standardize HTTP logs

**Files:**
- Modify: `templates/quickstart-nextjs/app/billing/page.tsx`
- Delete: `templates/quickstart-nextjs/components/stripe-topup-form.tsx`
- Modify: `templates/quickstart-nextjs/lib/api.ts`
- Test: `templates/quickstart-nextjs/tests/api.test.ts`
- Modify: `templates/quickstart-nextjs/package.json`
- Modify: `foundation/ginx/access_log.go`
- Test: `foundation/ginx/ginx_test.go`
- Modify: `templates/quickstart/internal/bootstrap/app.go`

**Steps:** Keep fixed Stripe Checkout products only, remove Stripe Elements dependencies, use route-template logging fields, add safe HTTP timeouts, then run focused frontend and foundation tests.

### Task 5: Make project initialization and release verification repeatable

**Files:**
- Create: `scripts/init-quickstart.sh`
- Create: `scripts/verify-quickstart-release.sh`
- Modify: `Makefile`
- Modify: `.github/workflows/ci.yml`

**Steps:** Copy backend/frontend, replace module/service/package identity, verify a specified published go-modules tag with `GOWORK=off`, and expose both workflows through Make/CI.

### Task 6: Update dependencies and deployment documentation

**Files:**
- Modify: root/template Go module files
- Modify: `templates/quickstart-nextjs/package*.json`
- Modify: `README.md`
- Modify: `CONTRIBUTING.md`
- Modify: `docs/INTEGRATION.md`
- Modify: `docs/SETUP_ZH.md`
- Modify: template READMEs

**Steps:** Upgrade vulnerable dependencies with the smallest compatible versions, document every required environment variable and exact Google/GitHub/Stripe callback URL, then run full Go, race, frontend, detached-template, build and vulnerability checks.

