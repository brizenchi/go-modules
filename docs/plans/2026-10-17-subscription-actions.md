# Subscription Actions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show Change plan and Cancel subscription, replacing Cancel subscription with Renew after cancellation.

**Architecture:** Keep the current billing APIs and payment rules. Move scheduled-change management into the plan dialog; cancellation explicitly clears a future schedule before stopping renewal. Read current subscription state before writes and retain account/session guards.

**Tech Stack:** Next.js, React, TypeScript, node:test.

---

### Task 1: Action and lifecycle regression tests

**Files:** `templates/quickstart-nextjs/tests/billing-presentation.test.ts`, `templates/quickstart-nextjs/tests/subscription-management.test.ts`.

1. Expect Change plan for ongoing subscriptions, including canceled renewal, pending payments and schedules.
2. Cover Cancel/Renew labels, schedule removal before cancellation, failures, lost responses and stale sessions.
3. Run `npm test`; observe failure before implementation.

### Task 2: State handling and presentation

**Files:** `templates/quickstart-nextjs/lib/billing-presentation.ts`, `templates/quickstart-nextjs/lib/subscription-management.ts`, `templates/quickstart-nextjs/components/subscription-change-dialog.tsx`, `templates/quickstart-nextjs/app/(default)/(private)/(customer)/billing/page.tsx`, adjacent CSS, `templates/quickstart-nextjs/BILLING_UX.md`.

1. Keep two visible recurring-subscription controls and remove Billing settings.
2. Show future-plan information without a separate card button; manage that schedule inside Change plan.
3. Keep pending payment recovery inside the plan dialog.
4. Confirm cancellation explicitly, including removal of any scheduled change; never report successful cancellation from a failed or stale request.

### Task 3: Verification

1. Run `npm run verify` in an isolated frontend copy without configured environment files.
2. Use only the in-memory API fixture to check normal, scheduled, canceled renewal and pending-payment states, including Chinese labels.
3. Check `git diff --check`; leave changes local without pushing or deploying.
