import assert from "node:assert/strict";
import { test } from "node:test";
import { canonicalURL, privatePaths, publicMetadata, sitemapEntries, siteOrigin } from "../lib/seo";
import { articles } from "../content/articles";

test("canonical URLs strip tracking and invitation values and reject foreign origins", () => {
  assert.equal(canonicalURL("/blog?ref=secret#content"), `${siteOrigin}/blog`);
  for (const path of ["https://example.com", "//example.com", "/\\example.com", "/\t/example.com"]) assert.throws(() => canonicalURL(path));
  const metadata = publicMetadata("Guide", "Guide summary", "/blog/test", { publishedAt: "2026-09-06" });
  assert.equal(metadata.alternates?.canonical, `${siteOrigin}/blog/test`);
  assert.ok(metadata.openGraph && "type" in metadata.openGraph && metadata.openGraph.type === "article");
});

test("sitemap includes only public content, with no account or invitation URLs", () => {
  const entries = sitemapEntries();
  assert.equal(new Set(entries.map((entry) => entry.url)).size, entries.length);
  for (const entry of entries) {
    const url = new URL(entry.url);
    assert.equal(url.origin, siteOrigin);
    assert.equal(url.search, "");
    assert.ok(!privatePaths.some((path) => url.pathname === path || url.pathname.startsWith(`${path}/`)));
  }
  for (const article of articles) assert.ok(entries.some((entry) => entry.url.endsWith(`/blog/${article.slug}`)));
});
