# Publishing content

The public website includes `/docs`, `/pricing`, `/blog`, `/blog/[slug]`, `/updates`, `/contact`, `/privacy`, and `/terms`. Content is stored with the source code and published by rebuilding the Next.js application. No separate CMS or search service is required.

## Add or edit an article

1. Edit `content/articles.ts`. Each article needs a unique lowercase `slug`, a real publication date (`YYYY-MM-DD`), a category, title, summary, and sections.
2. Supply both `en` and `zh` text. Section IDs must be unique within the article; keep published slugs and IDs stable so existing links still work.
3. The list, full-text search, article routes, metadata, and sitemap use the same article collection. Unknown article slugs return 404. Search is local and matches title, category, summary, and body in the currently selected language.
4. Run `npm test`, `npm run lint`, and `npm run build` with Node 22 before publishing. Check the changed page in both languages and on a narrow screen.

The included articles are starter guides, not testimonials or customer case studies. Replace them with useful content for your own product. Keep unpublished drafts outside the exported `articles` collection. The current implementation builds all exported articles, including a future-dated entry, so only add content when ready to publish.

## Documentation and release notes

- Edit `content/docs.ts` to change the searchable setup documentation.
- Edit `content/updates.ts` for release notes. Use real dates and link to shipped pages or published guides. The newest entries should appear first.
- Edit `components/content/pricing.tsx` for plan copy. The sample prices are unchanged from the original template; keep them aligned with the backend Stripe catalog before launching. The template is free, while these plans demonstrate your product’s commercial offer.

## Privacy and terms

`content/policies.ts` contains clearly labelled editable starter copy. It deliberately does not invent an operator, jurisdiction, retention period, refund commitment, or compliance certification. Supply the actual operator details, effective dates, data practices, and product rules before using these as your service’s published policies. Update both languages and remove the starter notices in `components/content/policy.tsx` only once the content is ready.

## Contact and site settings

The contact page reads `GET {NEXT_PUBLIC_API_BASE_URL}/site/settings`, using the normal `{code: 200, data: ...}` envelope. The public response contains:

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

`/sitemap.xml` includes only real public routes and published article slugs. `/robots.txt` excludes account, billing, dashboard, referral, credits, files, notes, admin, login, invite, OAuth, and API paths, plus sensitive query patterns. Robots exclusions guide crawlers; backend authorization remains necessary for private data.

The existing language switch uses one URL for both languages and persists the preference in the browser. The initial server rendering and metadata are English. This is bilingual UI, **not separate indexed English and Chinese route trees**; the template does not claim `/zh` URLs or emit inaccurate `hreflang` alternatives. Add server-resolved locale routes separately if your product needs independently indexed translated pages.

Runtime site settings update visible branding/contact details. Deployment metadata and the default sharing card use `NEXT_PUBLIC_APP_NAME` / `NEXT_PUBLIC_APP_URL`; rebuild after changing those environment settings. Edit `siteDescription` and the sharing card copy for your own product.
