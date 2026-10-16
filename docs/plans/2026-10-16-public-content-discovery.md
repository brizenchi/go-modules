# Public content discovery implementation plan

> Execute in this session under the user's approved scope. Keep changes local.

**Goal:** Make public product and learning pages discoverable through homepage, directory and contextual links, with a useful footer as a secondary index.

**Architecture:** Preserve public URLs and the four direct header links. Server components supply the current language's published index to lightweight client presentation components; only an unconfigured or legacy API uses the static starter catalog. Shared pure selectors choose recent guides and related entries without reviving unpublished content.

**Tech stack:** Next.js 15, React 18, strict TypeScript, CSS modules, node:test, offline Chrome acceptance.

## Approved design

- Homepage: three published use cases, a scenario overview link, three recent published guides, updates, demo and project information links.
- Product archives: visible peer navigation for features, use cases, comparisons and demo.
- Guides: real documentation/article/update directory links before search and article cards.
- Detail pages: related published content, plus existing parent/home breadcrumbs. Keep shallow breadcrumbs.
- Footer: consistent brand treatment, Product / Guides / About & support columns, separate legal row. No repeated destinations or menus in the header.
- Fix misleading header selection on standalone/support pages. Keep customer and administrator navigation outside this change.

## Implementation and verification

1. Add failing pure selector tests and a build assertion for real directory anchors and homepage reachability without the global footer. Reproduce the current gap against the prior disposable build.
2. Add `lib/content-discovery.ts`, a static catalog adapter and a server-side discovery loader. Test ordering, deduplication, related links, empty catalogs, locale and removal behavior.
3. Add shared discovery cards, directory navigation and related content components/CSS. Extract homepage presentation into a client component so the route can supply server-read entries in both languages.
4. Wire discovery into managed and static article/archive paths. Never substitute seed entries for an empty managed catalog. Avoid nonexistent or unpublished destinations in recommendations.
5. Refine `content/navigation.ts` and footer spacing/brand/accessibility. Update design/README documentation with scope and data source.
6. Run frontend `npm run verify` in a disposable copy, against a disposable local preview backend. Run offline React interactions and Chrome layout checks for English/Chinese desktop/mobile, including the new directory links and cards. Inspect screenshots and check `git diff --check`.

## Scope boundaries

No dependency, backend, billing, database, production, commit, push or deployment changes. Do not modify or restart existing user services. Search traffic and ranking are not acceptance criteria for this UI change.

## Source basis

Local knowledge base: `再聊内链建设：内链重要性不亚于外链` (lines 75–103), and `如何制作出SEO友好的网页？先从学习谷歌是如何理解我们网页开始` (lines 115–125). Apply category listings, parent links, related content and short click paths; directory names and card styling are implementation choices.

## Verification outcome

- Reproduced the missing/too-deep discovery path against the previous disposable build before implementation.
- Frontend verify passed for both disposable CMS and static builds: 213 unit tests, lint, production build, 72 public pages and same-language three-click reachability without the global footer.
- Offline browser acceptance passed: 136 public/auth interaction assertions, 100 discovery/layout assertions and 12 CMS archives at three viewport widths. Inspected English/Chinese homepage, directory, detail and footer screenshots.
- No backend, seed data, production, dependency, commit, push or deployment changes. Stopped the disposable acceptance backend; did not restart existing user services.
