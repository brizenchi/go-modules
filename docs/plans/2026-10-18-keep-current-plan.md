# Keep Current Plan Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** A subscriber who scheduled a future downgrade can keep their current plan through its pricing card.

**Architecture:** Add an explicit keep action for an ongoing subscription with a future schedule. Confirm once, re-read the reviewed subscription, remove only its unchanged schedule, then verify the current plan and billing cycle remain intact. Reuse the existing schedule-cancel endpoint; never create Checkout or submit a paid plan change for this action.

**Tech Stack:** React, TypeScript, node:test; existing Stripe billing APIs.

---

### Task 1: Regression coverage

Update `tests/subscription-change.test.ts`, `tests/subscription-management.test.ts`, `tests/pricing-checkout.test.ts`, and `tests/billing-presentation.test.ts` under `templates/quickstart-nextjs`.

Cover Premium with future Pro: its current card is enabled and says Cancel plan change; opening the dialog never writes. Browsing another interval still retains the actual current interval. Confirmation removes the reviewed schedule without charging. Changed state, failed removal and stale sessions cannot report success or remove a different schedule; a lost successful response can be retried without a second write.

Run `npm test` and observe failure before implementation.

### Task 2: Action, confirmation and documentation

Update `lib/subscription-change.ts`, `lib/subscription-management.ts`, `lib/pricing-checkout.ts`, `lib/billing-presentation.ts`, `components/subscription-change-dialog.tsx`, `components/content/pricing.tsx`, and `BILLING_UX.md`.

Keep the current card disabled when there is no future change. For a scheduled change, show Cancel plan change, following the official SaaS references in `BILLING_UX.md`, then a read-only confirmation with the canceled target/date and an explicit no-payment statement. Refresh state after confirmation and restore Current plan. Preserve current renewal cancellation and payment blockers.

### Task 3: Verification

Run `npm run verify` in an isolated frontend copy without environment files. Use only an in-memory API fixture for Premium -> scheduled Pro -> keep Premium, both locales, a non-current interval toggle and a repeated click. Confirm one schedule cancellation and zero Checkout/plan-change writes. Run `git diff --check`; leave production and deployment unchanged.
