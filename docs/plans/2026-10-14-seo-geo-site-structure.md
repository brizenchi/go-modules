# SEO and GEO site structure implementation plan

**Goal:** Turn the existing free Next.js + Go SaaS starter website into a coherent, bilingual public information architecture that connects specific user needs to working examples and product actions.

**Architecture:** Preserve existing English URLs and add static Chinese equivalents under `/zh`. Public pages use URL-selected language, shared content models, page-specific metadata and crawlable links. Account and administrator routes retain their existing addresses and use consistent non-indexing metadata.

**Tech stack:** Next.js 15 App Router, React 18, strict TypeScript, existing CSS tokens, node:test. No new dependencies.

The user approved the structure and implementation in this conversation. Continue in the shared checkout with existing frontend changes intact; do not commit, publish or change backend services.

## 1. Content and page models

- Add `content/marketing.ts`: bilingual capabilities, scenarios, an implementation comparison and a concrete notes/export example.
- Cover authentication, subscription billing and credits; subscription SaaS, usage-based products and paid exports; starter versus building from scratch.
- Give each page a clear purpose, working destination, factual explanation, limitations, FAQ, relevant guides and source links.
- Add pure model tests for unique paths, complete bilingual content and valid relationships.
- Present examples as examples; do not invent testimonials, usage metrics, customers or provider guarantees.

## 2. Public templates and navigation

- Add reusable index/detail/about/resources components and CSS in `components/content/marketing.*`.
- Add `components/localized-link.tsx`; preserve query/hash and localize only public routes.
- Update shared header, footer, breadcrumbs and public content links. Header: product, use cases, pricing, resources; workspace access remains an account action.
- Connect homepage capabilities and real notes workflow to public detail pages, then to existing account/product actions.
- Keep headings, keyboard controls, focus visibility and 320px layouts usable.

## 3. Locale URLs and metadata

- Add failing tests for public locale routing, canonical URLs, reciprocal language alternatives and private route exclusion.
- Separate default and Chinese root route layouts so initial HTML contains the correct language and content without depending on browser storage.
- Preserve old English paths, dynamically generated guide/article slugs and existing workspace URLs.
- Generate sitemap entries for all real public pages in both languages. Localize metadata, breadcrumbs and structured data consistently.
- Apply `noindex` to all account/auth/admin areas, without treating robots.txt as authorization or preventing crawlers from seeing the intended directive.

## 4. Verification and handoff

- Extend the built-content checker to inspect both languages, all page templates, internal routes, canonical/hreflang, schemas and private metadata.
- Run `npm run verify` with Node 22 in an isolated temporary copy so the user's development server is not interrupted.
- Inspect desktop/mobile navigation, locale switching and linked product journeys in the browser.
- Document route/content conventions and any unverified live search/payment outcomes. Do not claim rankings or AI citations from local verification.

## Completed verification

- Implemented 14 new public page paths: four directories, eight detail pages, about, and resources; all available in English and Chinese.
- Final `npm run verify` passed in an isolated copy with Node 22: 145 tests, lint, production build, and 72 public HTML pages / 28 guide and article pages checked.
- HTML checks cover headings, anchors, internal content links, canonical and language alternatives, sitemap coverage, structured data, and private noindex metadata.
- Browser verified desktop Chinese directory, 320px Chinese and English subscription detail, corresponding-language navigation, and FAQ expansion. No horizontal overflow; viewport restored.
- Updated CONTENT.md and DESIGN.md. No commit, deployment, backend change, or live search-ranking claim.
