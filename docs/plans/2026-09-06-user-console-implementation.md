# Quickstart User Console Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make settings, subscription management, and referrals immediately visible through a polished account center without a redundant overview page.

**Architecture:** Keep all API contracts and the existing account, billing, and referral routes. Teach the shared `SiteShell` to render a three-item account shell on those routes, route successful sign-in directly to `/account`, and retain `/dashboard` only as a compatibility redirect.

**Tech Stack:** Next.js 15, React 18, TypeScript, CSS Modules/global CSS, Node test runner, ESLint

---

### Task 1: Add direct account-center route tests

**Files:**
- Create: `templates/quickstart-nextjs/lib/workspace.ts`
- Create: `templates/quickstart-nextjs/tests/workspace.test.ts`

1. Write failing tests for the three account-center routes, their order, and active navigation.
2. Run `npm test` and confirm the old overview item fails the new expectations.
3. Implement the direct navigation helper with no synthetic overview destination.
4. Run `npm test` and confirm the new tests pass.

### Task 2: Build the persistent console shell

**Files:**
- Modify: `templates/quickstart-nextjs/components/site-shell.tsx`
- Modify: `templates/quickstart-nextjs/components/sign-in-dialog.tsx`
- Modify: `templates/quickstart-nextjs/app/globals.css`

1. Add the three-item account navigation with accessible inline icons and active state.
2. Render a dark desktop navigation rail, compact page header, utility bar, and mobile navigation for `/account`, `/billing`, and `/referrals`.
3. Expose those same destinations in the account popover and allow public sign-in success to navigate to `/account`.
4. Preserve the existing public header, dialog behavior on feature pages, locale switching, sign-out, and resource summaries.

### Task 3: Remove the redundant overview layer

**Files:**
- Modify: `templates/quickstart-nextjs/app/dashboard/page.tsx`

1. Replace the overview UI with a server-side redirect to `/account`.
2. Remove its CSS, presentation helpers, and overview-only tests.
3. Keep the legacy URL working for old bookmarks without showing a dashboard.

### Task 4: Make the working feature pages feel like product screens

**Files:**
- Modify: `templates/quickstart-nextjs/app/account/page.tsx`
- Modify: `templates/quickstart-nextjs/app/billing/page.tsx`
- Modify: `templates/quickstart-nextjs/app/referrals/page.tsx`
- Modify: `templates/quickstart-nextjs/app/login/page.tsx`

1. Replace developer/demo-oriented page headings with concise customer-facing language.
2. Route successful email and OAuth login directly to `/account`.
3. Keep every existing form, retry action, subscription mutation, portal action, and referral action intact.

### Task 5: Verify behavior and visual quality

**Files:**
- Verify: `templates/quickstart-nextjs/**/*`

1. Run `npm test`, `npm run lint`, and `npm run build`.
2. Run the local frontend against a deterministic mock API and capture desktop and mobile account, billing, and referral route checks.
3. Inspect navigation visibility, overflow, focus, loading/error handling, and responsive layout; fix defects before handoff.
4. Run `git diff --check` and report that no commit or push was performed.
