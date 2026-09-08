# Public content and SEO structure

## Scope

Keep the existing green, paper-toned visual language and bilingual UI. Improve the public website without changing authentication, admin, billing, or backend behavior.

## Implementation

- Turn `/docs` into a searchable, grouped guide directory with concise summaries. Publish each existing guide at `/docs/[slug]`, preserving its content and stable section identifier.
- Give guides and blog articles a compact editorial header, readable body, local table of contents, and contextual onward links. Replace detailed homepage instructions with links to these guides.
- Use semantic breadcrumb navigation and matching `BreadcrumbList` JSON-LD on public pages. Add accurate article structured data without invented dates or authors.
- Organize the footer into product, learning, and support/legal navigation. Maintain crawlable links, accessible headings, and responsive layouts.
- Include guide URLs in canonical metadata and the sitemap; unknown guide slugs return 404. Keep former guide fragment identifiers on directory entries for bookmarks.

## Verification

Test guide integrity, search, internal links, sitemap entries, and structured-data serialization. Run frontend tests, lint, and production build. Inspect rendered pages on desktop and mobile, including direct article links and initial server-rendered content.

## Boundaries

The current locale switch shares URLs: initial server rendering and metadata remain English. Independent translated SEO routes are a separate change. Search appearance and ranking are not guaranteed; validate the deployed site with Google's Rich Results Test and Search Console.

References: [Breadcrumbs](https://developers.google.com/search/docs/appearance/structured-data/breadcrumb), [Article](https://developers.google.com/search/docs/appearance/structured-data/article), [Crawlable links](https://developers.google.com/search/docs/crawling-indexing/links-crawlable).
