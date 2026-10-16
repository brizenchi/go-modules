# Public navigation and guides implementation plan

**Goal:** Make language/account menus distinct and useful, shorten the guide archive, and give secondary pages clear navigation paths.

**Architecture:** Keep the four primary links and existing published URLs. Shared navigation data supplies desktop/mobile dropdowns, footer groups and guide category links. Preserve the existing logout API/session ownership behavior; no backend or database changes.

**Tech Stack:** Next.js, React, TypeScript, CSS modules, node:test and isolated offline Chrome.

## 1. Menu states and account actions

- Separate selected language styling (checkmark/weight) from hover/focus fill and space the rows in `app/globals.css`.
- Add a shared `components/sign-out-button.tsx` using `logout`, `readSession` and `accountLogoutFailure`; prevent duplicate submission, show retryable errors and preserve a newer login.
- Remove the redundant account-information action/modal from `components/account-menu.tsx` and the default account page. Retain a read-only identity summary and direct sign-out.

## 2. Navigation and footer

- Add secondary link groups in `content/navigation.ts` and an accessible `components/public-navigation.tsx`: Features → overview/use cases/comparison/demo, Guides → guides/docs/articles/updates/help/about.
- Keep links in initial HTML; opening one dropdown closes another. Close on Escape, outside click, focus departure and route change; support Arrow/Home/End without changing normal Tab behavior.
- Organize footer into Product, Learn and Support. Move privacy/terms to the bottom; remove duplicated group/link names.

## 3. Guide discovery

- Add tested pure filtering in `lib/guide-index.ts`. Guides contain published documentation/articles; feature and use-case content retains its own pages and navigation.
- Update `components/published-content.tsx` and its CSS module to compact category filters/search and a responsive three/two/one-column card grid. Keep all published guide links in initial HTML; no invented thumbnails or content.
- Keep an equivalent directory in the static/no-API fallback.

## 4. Verification and delivery

- Start with failing guide/navigation assertions, then implement and rerun them.
- Run `npm run verify` in a disposable copy so the active development build is untouched; no configured database writes or service restarts.
- Inspect real React/CSS in isolated Chrome: language selection versus hover, direct logout success/failure/newer-session protection, submenu keyboard/dismissal, guide filtering and desktop/mobile overflow.
- Update frontend design/readme notes. Keep changes local; no commit, push or deployment.

## Verification results

- Frontend verification passed: 201 tests, lint, production build and 72 public-content checks. Both the static fallback and the CMS-backed build were checked in disposable directories.
- `scripts/check-public-ui.mjs`: 80 real browser assertions across English/Chinese and desktop/mobile. Includes real pointer hover, focused pending sign-out, retryable failures, duplicate-click protection, newer-session preservation, submenu keyboard/focus/visibility, filtering and footer layout. API/session boundaries use fixtures.
- The visibility regression first reproduced a clipped mobile submenu; the focused sign-out regression also reproduced blur hiding the pending action. Both pass after the fixes.
- `scripts/check-published-layout.mjs`: 12 CMS archive pages × 1440/768/390px, including three/two/one-column cards, usable title/search widths and no page overflow. Screenshots were inspected.
- Content/operations Go tests passed. The content seed generator now supports local TypeScript imports, fixing its prior `require is not defined` failure; regenerated only source-controlled seed copy and checked deterministic output.
- The CMS fixture used a new temporary SQLite database and local-only disabled providers on port 18082. It was stopped after acceptance; existing services and configured databases were untouched.
