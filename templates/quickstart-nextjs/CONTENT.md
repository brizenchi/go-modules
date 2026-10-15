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

`content/site.ts` is the public presentation entry point: default brand, bilingual homepage heading and introduction, primary/secondary buttons, homepage feature and journey copy, FAQ, preview images, footer and sharing card copy. `NEXT_PUBLIC_APP_NAME` overrides the default brand at build time (blank uses the configured default). Header, customer workspace, administrator shell, page metadata and sharing card all use that resolved name. Rebuild after changing the file or environment setting. The homepage screenshots are actual account-center captures with a fake demonstration account; replace both locale assets when you customize the product.

The contact page reads `GET {NEXT_PUBLIC_API_BASE_URL}/site/settings`, using the normal `{code: 200, data: ...}` envelope. The public response contains service details, support channels and export cost:

```json
{
  "brand_name": "Your product",
  "description": "Your product description",
  "support_email": "help@example.com",
  "support_url": "https://help.example.com",
  "export_credit_cost": 1
}
```

`lib/site-settings.ts` exports `getPublicSiteSettings(signal?)`, `publicSiteSettingsFallback`, `normalizePublicSiteSettings`, and `SITE_SETTINGS_EVENT`. After an admin saves settings, dispatch `new Event(SITE_SETTINGS_EVENT)` on `window` to refresh mounted consumers. Only these public fields belong in the endpoint; never return service credentials.

Optional build-time fallback settings are `NEXT_PUBLIC_SUPPORT_EMAIL` and `NEXT_PUBLIC_SUPPORT_URL`. They apply while the API is unreachable. A successful API response with blank support fields intentionally leaves contact channels unconfigured. The frontend accepts a plain email address and an HTTPS help URL without embedded credentials. It does not send mail itself; the email button opens the user’s mail app. There is no invented inbox or promised response time.

## SEO and languages

Set `NEXT_PUBLIC_APP_URL` to the actual public frontend origin before building. `lib/seo.ts` creates absolute canonical URLs and page-specific Open Graph / Twitter metadata. Canonicals omit queries such as invitation and checkout markers. `app/opengraph-image.tsx` generates the default 1200 × 630 PNG sharing card using the configured frontend brand; no remote image service is called.

`/sitemap.xml` includes only published public routes in both languages. Private pages emit `noindex, nofollow` metadata and their routes also receive an `X-Robots-Tag` header. `/robots.txt` allows crawlers to read that directive; blocking a page in robots.txt does not guarantee deindexing. Backend authorization protects private data.

Public pages render semantic breadcrumbs with matching `BreadcrumbList` JSON-LD. Guides and blog articles also render `Article` JSON-LD, a single page title, readable body sections, and contextual links. Structured data is present in the initial HTML and safely escaped. Only existing blog publication dates are emitted; guides do not invent publication dates or author identities. Supply accurate author and image information if you add it to the content model later. These structures help describe content, but do not guarantee rich results or rankings. Validate deployed URLs in Google's Rich Results Test and Search Console.

English pages keep their original paths; Chinese public pages use `/zh` (for example `/features/credits` and `/zh/features/credits`). Both are statically rendered with the correct initial text, HTML language, metadata, canonical, and reciprocal `en` / `zh-CN` / `x-default` alternatives. Public language follows the URL; workspace language remains a browser preference. Use `LocalizedLink` for public links so navigation keeps the current language.

`content/marketing.ts` defines feature, use-case, comparison, example, about, and resource pages. New exported entries generate both language routes and sitemap entries. Each detail page explains the outcome, factual capabilities, steps, limits, FAQs, related content, and sources. Keep review dates factual and distinguish implemented behavior from work the product owner still needs to build. Do not add fabricated users, benchmarks, or testimonials.

Runtime site settings update support channels and the example export cost. The API's `brand_name` and `description` remain service information; they do not override the frontend's build-time public identity or translated homepage copy. Use `content/site.ts`, `NEXT_PUBLIC_APP_NAME` and `NEXT_PUBLIC_APP_URL` for public presentation and rebuild, so the initial HTML, visible brand, metadata and sharing card agree.

### Customer account interactions

Account information opens in a compact native dialog from the account menu or workspace overview. Show the sign-in email as read-only and offer sign-out; do not expose avatar URLs, profile editing, session refresh, or WebSocket tickets in the customer interface. `/account` contains common account services: subscriptions, credits and referrals. Notes and uploads have no default frontend pages. `/examples` links directly to account, billing and referral services. Retired note/upload addresses redirect to the account center. Subscription, invoices, payment methods, and referral history retain dedicated pages; new purchases start on `/pricing`.

The customer area uses the contained `CustomerShell`; mobile navigation opens a side drawer. Sign-in and account information share `ResponsiveDialog`, which becomes a bottom sheet on narrow screens. Sign-in retains API-discovered Google/GitHub/email availability and the original validated return destination. Checkout progress and errors appear beside the selected offer.
