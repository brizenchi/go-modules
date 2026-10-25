import assert from "node:assert/strict";
import { test } from "node:test";
import { canonicalURL, exposedSitemapEntries, privateMetadata, privatePaths, publicMetadata, publicPageMetadata, sitemapEntries, siteOrigin } from "../lib/seo";
import { siteConfig } from "../content/site";
import { appEnv } from "../lib/env";
import { articles } from "../content/articles";
import { documentation } from "../content/docs";

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
  for (const guide of documentation) assert.ok(entries.some((entry) => entry.url === canonicalURL(`/docs/${guide.id}`)));
});

test("guide metadata describes an article without inventing a publication date", () => {
  for (const guide of documentation) {
    const metadata = publicMetadata(guide.title.en, guide.summary.en, `/docs/${guide.id}`, {});
    assert.equal(metadata.description, guide.summary.en);
    assert.equal(metadata.alternates?.canonical, canonicalURL(`/docs/${guide.id}`));
    assert.ok(metadata.openGraph && "type" in metadata.openGraph && metadata.openGraph.type === "article");
    assert.ok(!("publishedTime" in metadata.openGraph));
  }
});

test("localized metadata identifies its own canonical and reciprocal language alternatives", () => {
  const metadata = publicMetadata("价格方案", "比较可用套餐", "/zh/pricing?source=guide#plans");
  assert.equal(metadata.alternates?.canonical, `${siteOrigin}/zh/pricing`);
  assert.deepEqual(metadata.alternates?.languages, { en: `${siteOrigin}/pricing`, "zh-CN": `${siteOrigin}/zh/pricing`, "x-default": `${siteOrigin}/pricing` });
  assert.ok(metadata.openGraph && "locale" in metadata.openGraph && metadata.openGraph.locale === "zh_CN");
});

test("every public sitemap entry has an equivalent Chinese and English URL", () => {
  const entries = sitemapEntries();
  const urls = new Set(entries.map(entry => entry.url));
  assert.ok(urls.has(`${siteOrigin}/zh`));
  for (const entry of entries) {
    assert.ok(entry.alternates?.languages.en);
    assert.ok(entry.alternates?.languages["zh-CN"]);
    assert.ok(urls.has(entry.alternates!.languages.en));
    assert.ok(urls.has(entry.alternates!.languages["zh-CN"]));
  }
});

test("private metadata explicitly suppresses indexing and does not inherit the homepage canonical", () => {
  const metadata = privateMetadata("Account");
  assert.deepEqual(metadata.robots, { index: false, follow: false });
  assert.equal(metadata.alternates?.canonical, null);
  assert.deepEqual(metadata.alternates?.languages, {});
  assert.equal(metadata.openGraph, null);
});

test("both homepage metadata variants use the configured public identity", () => {
  for (const locale of ["en", "zh"] as const) {
    const metadata = publicPageMetadata("/", locale)!;
    assert.equal(metadata.title, `${siteConfig.homepageTitle[locale]} · ${appEnv.appName}`);
    assert.equal(metadata.description, siteConfig.description[locale]);
    assert.ok(metadata.openGraph && "siteName" in metadata.openGraph);
    assert.equal(metadata.openGraph.siteName, appEnv.appName);
  }
});

test("sitemap alternates match each page's hreflang set, including x-default", () => {
  for (const entry of sitemapEntries()) {
    assert.equal(entry.alternates.languages["x-default"], entry.alternates.languages.en);
  }
  const subscriptions = sitemapEntries().find(entry => entry.url === canonicalURL("/features/subscriptions"));
  assert.match(subscriptions?.lastModified || "", /^\d{4}-\d{2}-\d{2}$/);
});

test("an empty directory leaves the sitemap and the alternates of its other language", () => {
  const url = (path: string) => canonicalURL(path);
  const entries = [
    { url: url("/compare"), alternates: { languages: { en: url("/compare"), "zh-CN": url("/zh/compare"), "x-default": url("/compare") } } },
    { url: url("/zh/compare"), alternates: { languages: { en: url("/compare"), "zh-CN": url("/zh/compare"), "x-default": url("/compare") } } },
    { url: url("/docs"), alternates: { languages: { en: url("/docs"), "zh-CN": url("/zh/docs"), "x-default": url("/docs") } } },
    { url: url("/zh/docs"), alternates: { languages: { en: url("/docs"), "zh-CN": url("/zh/docs"), "x-default": url("/docs") } } },
    { url: url("/pricing") },
    { url: url("/compare/starter") }
  ];
  const exposed = exposedSitemapEntries(entries, { en: new Set(["/compare"]), zh: new Set(["/docs"]) });
  assert.deepEqual(exposed.map(entry => new URL(entry.url).pathname), ["/compare", "/zh/docs", "/pricing", "/compare/starter"]);
  assert.deepEqual(exposed[0].alternates?.languages, { en: url("/compare"), "x-default": url("/compare") });
  // With English hidden, the fallback for other languages points to Chinese.
  assert.deepEqual(exposed[1].alternates?.languages, { "zh-CN": url("/zh/docs"), "x-default": url("/zh/docs") });
});
