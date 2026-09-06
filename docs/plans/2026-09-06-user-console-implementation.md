# Quickstart User Console Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make settings, subscription management, and referrals immediately visible through a polished signed-in user console with a real overview page.

**Architecture:** Keep all existing App Router URLs and API contracts. Teach the shared `SiteShell` to render a console shell on signed-in product routes, add a `/dashboard` page that composes existing resource states, and retain the public marketing shell elsewhere. Use pure view-model helpers for dashboard presentation so important behavior is unit-tested without coupling tests to React internals.

**Tech Stack:** Next.js 15, React 18, TypeScript, CSS Modules/global CSS, Node test runner, ESLint

---

### Task 1: Add console route and dashboard view-model tests

**Files:**
- Create: `templates/quickstart-nextjs/lib/workspace.ts`
- Create: `templates/quickstart-nextjs/lib/dashboard.ts`
- Create: `templates/quickstart-nextjs/tests/workspace.test.ts`
- Create: `templates/quickstart-nextjs/tests/dashboard.test.ts`

1. Write failing tests for console-route detection, active navigation, profile completion, normalized plan labels, and actionable setup items.
2. Run `npm test` and confirm the missing modules fail compilation.
3. Implement pure helpers with no fake analytics or browser dependencies.
4. Run `npm test` and confirm the new tests pass.

### Task 2: Build the persistent console shell

**Files:**
- Modify: `templates/quickstart-nextjs/components/site-shell.tsx`
- Modify: `templates/quickstart-nextjs/components/sign-in-dialog.tsx`
- Modify: `templates/quickstart-nextjs/app/globals.css`

1. Add the four-item console navigation with accessible inline icons and active state.
2. Render a dark desktop navigation rail, compact page header, utility bar, and mobile console navigation for `/dashboard`, `/billing`, `/referrals`, and `/account`.
3. Add Dashboard to the account popover and allow public sign-in success to navigate to `/dashboard`.
4. Preserve the existing public header, dialog behavior on feature pages, locale switching, sign-out, and resource summaries.

### Task 3: Add the real overview page

**Files:**
- Create: `templates/quickstart-nextjs/app/dashboard/page.tsx`
- Create: `templates/quickstart-nextjs/app/dashboard/dashboard.module.css`

1. Load the current session plus account profile, capability, subscription, referral statistics, and referral link through existing clients.
2. Render actual plan, credit, invitation, profile, and renewal data with independent loading/error states.
3. Add direct actions to billing, referrals, and account settings; show the sign-in dialog when no session exists.
4. Avoid charts or values that the backend does not provide.

### Task 4: Make the working feature pages feel like product screens

**Files:**
- Modify: `templates/quickstart-nextjs/app/account/page.tsx`
- Modify: `templates/quickstart-nextjs/app/billing/page.tsx`
- Modify: `templates/quickstart-nextjs/app/referrals/page.tsx`
- Modify: `templates/quickstart-nextjs/app/login/page.tsx`

1. Replace developer/demo-oriented page headings with concise customer-facing language.
2. Route successful email and OAuth login to `/dashboard`.
3. Keep every existing form, retry action, subscription mutation, portal action, and referral action intact.

### Task 5: Verify behavior and visual quality

**Files:**
- Verify: `templates/quickstart-nextjs/**/*`

1. Run `npm test`, `npm run lint`, and `npm run build`.
2. Run the local frontend against a deterministic mock API and capture desktop and mobile `/dashboard` screenshots plus account, billing, and referral route checks.
3. Inspect navigation visibility, overflow, focus, loading/error handling, and responsive layout; fix defects before handoff.
4. Run `git diff --check` and report that no commit or push was performed.
