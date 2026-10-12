# Billing Lifecycle Implementation Plan

**Goal:** Complete payment visibility, safe subscription changes, entitlement checks, purchase history and recoverable synchronization in the quickstart template.

**Architecture:** Preserve existing shared APIs and add optional provider capabilities and response fields. Keep product authorization and reconciliation scheduling in the template. Preserve no-op invitation rewards. Use canonical Stripe state for billing; use bounded local entitlement validity for authorization. No automatic refund policy is invented.

**Tech Stack:** Go, Gin, GORM, existing stripe-go/v76, Next.js/TypeScript; no new dependencies.

## Tasks
1. Add failing Stripe contract tests for modern invoice/subscription fields; support both shapes and expand payment/schedule snapshots.
2. Add pending payment and scheduled change fields, cancellation of scheduled changes through optional provider capability, safe schedule handling, integer monetary fields. Test service/provider/HTTP behavior.
3. Add host membership policy and authenticated endpoints/middleware with strict expiration and ownership; cover lifetime, canceling, unpaid, delayed renewal and missing data.
4. Expose user-scoped purchase records from persisted billing events and credit balance links; test ownership and pagination.
5. Add provider event retrieval/feed and webhook replay, plus a leased bounded template reconciliation job for missed/failed events and authoritative snapshots. Test retry/lease/stale-event behavior without network.
6. Update frontend types and billing UI for effective cancellation, pending payment, future plans, purchase history and credits. Remove normal UI's 3-day cancellation. Add contract/state tests.
7. Update changelog and guides. Run gofmt, go vet, race tests, purity check, backend template tests and frontend verify. Check availability of sandbox tooling/configuration without printing secrets; do not claim Stripe E2E success without executing it.

## Verification
Use failing regression tests before fixes. Existing edits from the invitation task remain in place. Add no nested modules, applied migration edits, deployment or real payment actions. New optional schema uses AutoMigrate. Sandbox purchases use separate test users and test credentials only; if unavailable report that boundary explicitly.

## Completed verification
- Full repository race tests, purity check, billing vet, backend template vet/race and frontend verify executed.
- Real Stripe test-mode API and Test Clock checks executed with isolated customers and automatic cleanup.
- Live contract failures fixed with failing offline regressions: schedule creation parameters, reactivation clearing, and cancellation after a schedule phase applies.
- Repeatable opt-in script: `scripts/billing-sandbox/main.go`. Full browser Checkout/webhook fulfillment remains a separate acceptance checklist in `BILLING_TESTING.md`.
