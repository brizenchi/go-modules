# Quickstart SaaS Visual Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Next.js quickstart's warm glassmorphism styling with the approved Attio × Clerk-inspired SaaS visual system while preserving every existing route and integration flow.

**Architecture:** Keep the current App Router pages, API helpers, authentication state, and component contracts. Evolve the shared `SiteShell`, add a static product-preview component for the marketing home, and replace the global CSS design tokens and component presentation without changing backend behavior.

**Tech Stack:** Next.js 15, React 18, TypeScript, CSS, Node test runner, ESLint

---

### Task 1: Establish the shared visual shell

**Files:**
- Modify: `templates/quickstart-nextjs/components/site-shell.tsx`
- Modify: `templates/quickstart-nextjs/app/layout.tsx`
- Modify: `templates/quickstart-nextjs/app/globals.css`

1. Add a compact brand mark and simplify the brand hierarchy.
2. Refine header, navigation, locale, account menu, breadcrumbs, shared hero, and table-of-contents markup only where new styling needs a stable hook.
3. Replace warm gradients, glass effects, orange accents, and oversized radii with the approved neutral-and-green design tokens.
4. Add keyboard focus, responsive, and reduced-motion styles.

### Task 2: Build the homepage product story

**Files:**
- Create: `templates/quickstart-nextjs/components/product-preview.tsx`
- Modify: `templates/quickstart-nextjs/app/page.tsx`
- Modify: `templates/quickstart-nextjs/components/marketing.tsx`
- Modify: `templates/quickstart-nextjs/app/globals.css`

1. Rewrite the home hero copy into a concise product promise while preserving links and environment diagnostics.
2. Add an accessible, static dashboard preview that communicates auth, billing, referrals, and API health.
3. Reorder capability, route, and operational sections into a clearer marketing narrative.
4. Style the preview, bento features, route list, metrics, and final CTA using the shared tokens.

### Task 3: Verify secondary routes and responsive behavior

**Files:**
- Verify: `templates/quickstart-nextjs/app/pricing/page.tsx`
- Verify: `templates/quickstart-nextjs/app/docs/page.tsx`
- Verify: `templates/quickstart-nextjs/app/login/page.tsx`
- Verify: `templates/quickstart-nextjs/app/account/page.tsx`
- Verify: `templates/quickstart-nextjs/app/billing/page.tsx`
- Verify: `templates/quickstart-nextjs/app/referrals/page.tsx`
- Verify: `templates/quickstart-nextjs/app/invite/page.tsx`

1. Run `npm run test` in `templates/quickstart-nextjs`.
2. Run `npm run lint` in `templates/quickstart-nextjs`.
3. Run `npm run build` in `templates/quickstart-nextjs`.
4. Capture desktop and mobile screenshots of `/` and inspect one dense secondary route such as `/pricing`.
5. Correct overflow, spacing, contrast, and interaction regressions found during visual review.

