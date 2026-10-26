# Publishing content

The public website includes `/docs`, `/docs/[slug]`, `/pricing`, `/blog`, `/blog/[slug]`, `/updates`, `/contact`, `/privacy`, and `/terms`. Content is stored with the source code and published by rebuilding the Next.js application. No separate CMS or search service is required.

## Add or edit an article

1. Edit `content/articles.ts`. Each article needs a unique lowercase `slug`, a real publication date (`YYYY-MM-DD`), a category, title, summary, and sections.
2. Supply both `en` and `zh` text. Section IDs must be unique within the article; keep published slugs and IDs stable so existing links still work.
3. The list, full-text search, article routes, metadata, and sitemap use the same article collection. Unknown article slugs return 404. Search is local and matches title, category, summary, and body in the currently selected language.
4. Run `npm run verify` with Node 22 before publishing. It runs tests, lint, a production build, and checks the generated HTML for headings, anchors, canonical URLs, structured data, directory links, and sitemap coverage. Check the changed page in both languages and on a narrow screen.

The included articles are starter guides, not testimonials or customer case studies. Replace them with useful content for your own product. Keep unpublished drafts outside the exported `articles` collection. The current implementation builds all exported articles, including a future-dated entry, so only add content when ready to publish.

## Documentation and release notes

- Edit `content/docs.ts` to change the setup guides. Each entry needs a stable lowercase `id` (its URL slug), a `group` from `documentationGroups`, a concise bilingual `summary`, a title, and the full article content. Keep long instructions and configuration examples here, not on the homepage or directory.
- `/docs` displays grouped summaries and searches the full guide content in the selected language. `/docs/[slug]` renders each full guide with its own canonical, article metadata, table of contents, and previous/next links. Unknown slugs return 404. Keep IDs stable: old `/docs#id` bookmarks still land on the matching directory card, while new links should use `/docs/id`.
- Edit `content/updates.ts` for release notes. Use real dates and link to shipped pages or published guides. The newest entries should appear first.
- Edit `components/content/pricing.tsx` for plan copy. Purchases start on the pricing page after selecting the billing interval; guest sign-in stays in a dialog and OAuth returns to the selected pricing route. Available offers are checked against the backend before creating Stripe Checkout. Existing subscribers use the billing page for changes. Checkout success and cancellation return to the current pricing language; a return URL alone does not prove payment. The sample prices are unchanged from the original template; keep them aligned with the backend Stripe catalog before launching. The template is free, while these plans demonstrate your product’s commercial offer.

## Privacy and terms

`content/policies.ts` contains clearly labelled editable starter copy. It deliberately does not invent an operator, jurisdiction, retention period, refund commitment, or compliance certification. Supply the actual operator details, effective dates, data practices, and product rules before using these as your service’s published policies. Update both languages and remove the starter notices in `components/content/policy.tsx` only once the content is ready.

## Contact and site settings

`content/site.ts` supplies presentation defaults: default brand, bilingual homepage heading and introduction, primary/secondary buttons, homepage feature and journey copy, FAQ, preview images, footer and sharing card copy. `NEXT_PUBLIC_APP_NAME` overrides the default brand at build time (blank uses the configured default). With the current backend, Website settings overrides the visible brand, bilingual homepage heading/introduction and page metadata through server-rendered settings. Sharing artwork, buttons and other layout copy remain code-owned; rebuild after changing those or environment defaults. The homepage screenshots are actual account-center captures with a fake demonstration account; replace both locale assets when you customize the product.

The contact page reads `GET {NEXT_PUBLIC_API_BASE_URL}/site/settings`, using the normal `{code: 200, data: ...}` envelope. The public response contains non-secret website fields, support channels and the legacy example export cost:

```json
{
  "brand_name": "Your product",
  "description": "Your English product description",
  "description_zh": "中文产品介绍",
  "headline_en": "Your homepage headline",
  "headline_zh": "首页标题",
  "sender_email": "mail@example.com",
  "sender_name": "Your product",
  "presentation_version": 1,
  "support_email": "help@example.com",
  "support_url": "https://help.example.com",
  "export_credit_cost": 1
}
```

`lib/site-settings.ts` exports `getPublicSiteSettings(signal?)`, `publicSiteSettingsFallback`, `normalizePublicSiteSettings`, and `SITE_SETTINGS_EVENT`. After an admin saves settings, dispatch `new Event(SITE_SETTINGS_EVENT)` on `window` to refresh mounted consumers. Only these public fields belong in the endpoint; never return service credentials.

Optional build-time fallback settings are `NEXT_PUBLIC_SUPPORT_EMAIL` and `NEXT_PUBLIC_SUPPORT_URL`. They apply while the API is unreachable. A successful API response with blank support fields intentionally leaves contact channels unconfigured. The frontend accepts a plain email address and an HTTPS help URL without embedded credentials. It does not send mail itself; the email button opens the user’s mail app. There is no invented inbox or promised response time.

## SEO and languages

Set `NEXT_PUBLIC_APP_URL` to the actual public frontend origin before building. `lib/seo.ts` creates absolute canonical URLs and page-specific Open Graph / Twitter metadata. Canonicals omit queries such as invitation and checkout markers. `app/opengraph-image.tsx` generates the default 1200 × 630 PNG sharing card using the configured frontend brand; no remote image service is called.

`/sitemap.xml` includes only published public routes, with the same `hreflang` alternates (`en`, `zh-CN`, `ja`, `ko`, `es`, `fr`, `de` as published, plus `x-default`) as each page. `/llms.txt` lists the published pages by section for AI assistants.

A section exists in a language only when it has something to show: Features, Use cases, Comparisons, Documentation and Articles need at least one published page of their kind, Resources needs documentation or articles, and About/Updates need their own published page. A section without content leaves the header, footer, in-page directory links, sitemap, `llms.txt` and the other language's `hreflang`, and its directory page is `noindex`. Published text that links to an unpublished page renders that link as plain text. Publishing or unpublishing in admin updates all of these within the normal cache window; no code change is needed for a product that omits sections. Pricing, Live demo, Help & support and legal pages are code-owned and always present.

Articles can be written with AI in Admin → Content when `host.ai` is configured (for example Gemini: `base_url: https://generativelanguage.googleapis.com/v1beta/openai`, `model: gemini-3.8-flash`, `max_output_tokens: 32768`, `reasoning_effort: low`, `timeout_seconds: 180`). Open a language tab, enter the target keyword and the product facts, and choose **Write**: the model drafts the title, summary, a body with headings and an FAQ, search settings and, for a new article, the URL. It may state only the facts you supplied, links only to published pages, and suggests where images belong. Upload each suggested image (or use **Insert image at cursor**); images are public at `/api/v1/site/images/<id>` with their size recorded, and render lazily with alt text. **Translate and save drafts** translates the open language into the ticked languages, keeping links and images, and saves every language as a draft. Nothing is published until you review and publish each language. Without `host.ai` the editor says that AI writing is not configured; manual editing is unchanged.

Header and footer are edited in Admin → Site settings → Navigation & footer and saved with the other site settings. Until they are saved the header shows Home, Features, Use cases, Comparisons and Pricing as one level of links, and the footer has Product and About & support. Each header item names a section; its menu lists that section's published pages automatically, the pages you choose in your order (any published article), or nothing. Menus are a single level. Destinations are limited to site sections and published articles, menus are rendered into the initial HTML, and the sitemap follows published content rather than navigation, so changing menus does not remove a page from search. Breadcrumbs are always Home › section › page, independent of the menus. Private pages emit `noindex, nofollow` metadata and their routes also receive an `X-Robots-Tag` header. `/robots.txt` allows crawlers to read that directive; blocking a page in robots.txt does not guarantee deindexing. Backend authorization protects private data.

Public pages render semantic breadcrumbs with matching `BreadcrumbList` JSON-LD. Documentation renders `TechArticle` and blog articles `BlogPosting`; feature, use-case and comparison pages are `WebPage`, not articles. Each language's homepage renders `Organization`, `WebSite` and the `SoftwareApplication` described by `siteConfig.application`; visible FAQ sections also render `FAQPage`. A CMS snapshot's time is its latest publication, so it is emitted as `dateModified`. Content pages render a single page title, readable body sections, and contextual links. Structured data is present in the initial HTML and safely escaped. Only existing blog publication dates are emitted; guides do not invent publication dates or author identities. Supply accurate author and image information if you add it to the content model later. These structures help describe content, but do not guarantee rich results or rankings. Validate deployed URLs in Google's Rich Results Test and Search Console.

English pages keep their original paths; Chinese public pages use `/zh` (for example `/features/credits` and `/zh/features/credits`). Both use server-rendered HTML with short-lived ISR caching. Article language alternatives include only published translations; an unpublished translation returns 404 and is omitted from the language picker and sitemap. Public language follows the URL and is saved for subsequent account/admin navigation. Language choices in either the public picker or private picker persist under the same local-storage key. With no saved choice, private pages default to English regardless of browser/OS language. A direct public URL keeps its indexed language and carries it into the account area; private pages also synchronize language changes from other tabs. Use `LocalizedLink` for public links so navigation keeps the current language.

Articles (features, use cases, comparisons, documentation and blog posts) can also be published in Japanese, Korean, Spanish, French and German at `/ja`, `/ko`, `/es`, `/fr` and `/de` (for example `/ja/features/credits`). The site interface — header, footer, homepage, directories, pricing and the account area — exists in English and Chinese only, so these article pages use the English interface around the translated article, and the language prefix serves articles only: `/ja`, `/ja/pricing` and directory pages return 404. Each article page sets its own `<html lang>`, canonical, `og:locale` and `hreflang`; its links point to the same language when that page is published in it and otherwise to the English page. To add the interface in another language later, give `Locale` in `lib/locale.ts` the new code, supply the `t()` strings and add that language's root routes; the content languages and URLs already exist.

`content/marketing.ts` defines feature, use-case, comparison, example, about, and resource pages. New exported entries generate both language routes and sitemap entries. Each detail page explains the outcome, factual capabilities, steps, limits, FAQs, related content, and sources. Keep review dates factual and distinguish implemented behavior from work the product owner still needs to build. Do not add fabricated users, benchmarks, or testimonials.

Runtime Website settings supplies the visible brand, bilingual homepage heading/introduction and metadata; blank optional translated copy uses code defaults. Sender email/name is non-secret and takes effect on backend restart. `NEXT_PUBLIC_APP_URL` remains deployment-owned. The generated sharing image remains build-time artwork.

## Localized article publishing

Use `/admin/articles` to manage blog, docs, feature, use-case and comparison detail pages. Each article has one tab per language (EN, 中文, 日本語, 한국어, ES, FR, DE). Import a UTF-8 `.md` file up to 150 KB, edit the selected language, save a draft, preview, then publish that language explicitly. Titles, summaries and body are required for publication. A saved draft never replaces its live snapshot until published. Unpublish languages independently; changing a URL after creation is intentionally unsupported in v1.

The renderer supports paragraphs, headings, lists, quotes, fenced code, inline code, emphasis and safe links. Raw HTML/MDX is displayed as text, never executed; this is not a full rich text/page builder. Homepage, pricing, policies, updates and other specialized layouts remain code-owned.

`content/articles.ts`, `content/docs.ts` and `content/marketing.ts` seed existing entries through `internal/feature/content/seed.json`. Regenerate seeds from the repo root with `node scripts/sync-content-seed.cjs` after changing starter copy. Stored edits/publication tombstones take precedence over seeds, so a code update cannot silently republish a removed article.

Public HTML, article metadata, language alternatives, directories and sitemap read published snapshots. Next.js server runtime is required; static export is not supported. Reads revalidate after 30 seconds at the fetch/page layers, so later requests normally reflect publication within about one minute. A configured API must be reachable during builds. Only an absent legacy endpoint (404) uses code compatibility; ordinary backend errors do not resurrect unpublished code content.

### Customer account interactions

The account menu shows the sign-in identity and direct sign-out, plus account, billing and referral destinations. Do not add a second account-information dialog for the same identity. The account page shows a read-only email and direct sign-out; do not expose avatar URLs, profile editing, session refresh, or WebSocket tickets. `/account` contains common account services: subscriptions, credits and referrals. Notes and uploads have no default frontend pages. `/examples` links directly to account, billing and referral services. Retired note/upload addresses redirect to the account center. Subscription, invoices, payment methods, and referral history retain dedicated pages; new purchases start on `/pricing`.

The customer area uses the contained `CustomerShell`; mobile navigation opens a side drawer. Sign-in and account information share `ResponsiveDialog`, which becomes a bottom sheet on narrow screens. Sign-in retains API-discovered Google/GitHub/email availability and the original validated return destination. Checkout progress and errors appear beside the selected offer.
