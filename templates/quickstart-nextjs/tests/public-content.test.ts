import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { discoveryIndex, isManagedPath, managedMetadata, publishedDirectories, publishedIndex, publishedPaths, publishedArticle } from "../lib/public-content";
import { appEnv } from "../lib/env";
import { canonicalURL } from "../lib/seo";

const originalFetch = globalThis.fetch;
const configured = appEnv.apiConfigured;
afterEach(() => { globalThis.fetch = originalFetch; appEnv.apiConfigured = configured; });
const article = { path: "/blog/custom", locales: ["en"], published_at: "2026-10-15T00:00:00Z", translation: { title: "Custom article", summary: "Published summary", seo_title: "", seo_description: "" } };
const response = (data: unknown) => new Response(JSON.stringify({ code: 200, data }), { headers: { "Content-Type": "application/json" } });

test("only valid editorial paths reach CMS resolution; malformed paths remain a 404", async () => {
 for (const path of ["/", "/features", "/blog/real-guide", "/features/ai-integration"]) assert.equal(isManagedPath(path), true);
 for (const path of ["/features/UPPER", "/blog/nested/path", "/docs/bad_slug", "/admin", "/pricing"]) assert.equal(isManagedPath(path), false);
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => { throw new Error("Invalid route must not call the CMS"); }) as typeof fetch;
 assert.equal(await managedMetadata("/features/UPPER", "en"), null);
});

test("managed article metadata keeps share artwork and only live language alternatives", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async url => String(url).includes("/site/settings") ? response({ presentation_version: 1, brand_name: "Live product" }) : response({ article })) as typeof fetch;
 const metadata = await managedMetadata(article.path, "en");
 assert.equal(metadata?.title, "Custom article · Live product");
 assert.equal(metadata?.alternates?.canonical, canonicalURL(article.path));
 assert.deepEqual(metadata?.alternates?.languages, { en: canonicalURL(article.path), "x-default": canonicalURL(article.path) });
 assert.deepEqual(metadata?.openGraph?.images, [{ url: canonicalURL("/opengraph-image"), width: 1200, height: 630, alt: appEnv.appName }]);
 assert.deepEqual(metadata?.twitter?.images, [canonicalURL("/opengraph-image")]);
});

test("live build paths replace unpublished seed routes while retaining fixed layouts", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => response({ items: [article], total: 1 })) as typeof fetch;
 assert.deepEqual(await publishedPaths("en", ["/", "/pricing", "/blog/removed"]), ["/", "/pricing", "/blog/custom"]);
 globalThis.fetch = (async () => response({ article: null })) as typeof fetch;
 assert.deepEqual(await publishedArticle("/blog/removed", "en"), { managed: true, article: null });
});

test("legacy APIs without content endpoints remain usable during frontend upgrades", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 404 }), { status: 404 })) as typeof fetch;
 assert.deepEqual(await publishedArticle("/blog/removed", "en"), { managed: false, article: null });
 assert.ok((await discoveryIndex("en")).length > 0);
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 503 }), { status: 503 })) as typeof fetch;
 await assert.rejects(publishedArticle("/blog/removed", "en"));
});

test("discovery never replaces an empty managed catalog with unpublished starter content", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => response({ items: [], total: 0 })) as typeof fetch;
 assert.deepEqual(await discoveryIndex("zh"), []);
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 503 }), { status: 503 })) as typeof fetch;
 await assert.rejects(discoveryIndex("en"));
});

test("static discovery remains bilingual and dates only known publications", async () => {
 appEnv.apiConfigured = false;
 const en = await discoveryIndex("en");
 const zh = await discoveryIndex("zh");
 assert.deepEqual(en.map(row => row.path), zh.map(row => row.path));
 assert.notEqual(en[0].translation.title, zh[0].translation.title);
 assert.ok(en.filter(row => row.kind === "docs").every(row => row.published_at === undefined));
 assert.ok(en.some(row => row.kind === "use-cases"));
});

test("managed discovery uses only the current language's published response and strips article bodies", async () => {
 appEnv.apiConfigured = true;
 let requested = "";
 globalThis.fetch = (async url => {
  requested = String(url);
  return response({ items: [{ ...article, id: "fixture", kind: "blog", position: 1, translation: { ...article.translation, body: "[Guide](/docs/setup)" } }], total: 1 });
 }) as typeof fetch;
 const rows = await discoveryIndex("zh");
 assert.ok(requested.includes("locale=zh"));
 assert.deepEqual(rows.map(row => row.path), ["/blog/custom"]);
 assert.deepEqual(rows[0].links, ["/docs/setup"]);
 assert.equal("body" in rows[0].translation, false);
});


test("homepage SEO comes from the published page instead of legacy settings", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async url => String(url).includes("/site/settings") ? response({ presentation_version: 1, brand_name: "Live product", headline_en: "Old heading", description: "Old description" }) : response({ article: { ...article, kind: "page", path: "/" } })) as typeof fetch;
 const metadata = await managedMetadata("/", "en");
 assert.equal(metadata?.title, "Custom article · Live product");
 assert.equal(metadata?.description, "Published summary");
 assert.equal(metadata?.openGraph && "type" in metadata.openGraph ? metadata.openGraph.type : undefined, "website");
});

test("empty CMS keeps structural pages available without inventing published articles", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async url => String(url).includes("/resolve") ? response({ article: null }) : response({ items: [], total: 0 })) as typeof fetch;
 const metadata = await managedMetadata("/features", "en");
 assert.equal(metadata?.title, `Features · ${appEnv.appName}`);
 assert.deepEqual(metadata?.robots, { index: false, follow: true });
 assert.deepEqual(await publishedPaths("en", ["/", "/features", "/resources", "/about", "/pricing"]), ["/", "/features", "/resources", "/about", "/pricing"]);
 assert.equal(await managedMetadata("/blog/unpublished", "en"), null);
});

test("a published directory without pages in a language is not indexed or offered as an alternate", async () => {
 appEnv.apiConfigured = true;
 const directory = { ...article, id: "compare", kind: "page", path: "/compare", locales: ["en", "zh"], translation: { ...article.translation, title: "Comparisons", body: "" } };
 const child = { ...article, id: "child", kind: "compare", path: "/compare/starter", position: 1, locales: ["en"], translation: { ...article.translation, body: "" } };
 globalThis.fetch = (async url => {
  const value = String(url);
  if (value.includes("/resolve")) return response({ article: directory });
  if (value.includes("/site/articles")) return response(value.includes("locale=en") ? { items: [directory, child], total: 2 } : { items: [directory], total: 1 });
  return response({ presentation_version: 1, brand_name: "Live product" });
 }) as typeof fetch;
 const en = await managedMetadata("/compare", "en");
 assert.equal(en?.robots, undefined);
 assert.deepEqual(en?.alternates?.languages, { en: canonicalURL("/compare"), "x-default": canonicalURL("/compare") });
 const zh = await managedMetadata("/compare", "zh");
 assert.deepEqual(zh?.robots, { index: false, follow: true });
 assert.deepEqual(await publishedDirectories("zh"), new Set());
 assert.deepEqual(await publishedDirectories("en"), new Set(["/compare"]));
});

test("an article in Japanese has its own canonical and alternates; other Japanese pages do not exist", async () => {
 appEnv.apiConfigured = true;
 const japanese = { ...article, id: "ja", kind: "docs", path: "/docs/setup", locales: ["en", "ja"], translation: { ...article.translation, title: "セットアップ", summary: "設定します。", body: "" } };
 globalThis.fetch = (async url => String(url).includes("/site/settings") ? response({ presentation_version: 1, brand_name: "Live product" }) : response({ article: japanese })) as typeof fetch;
 const metadata = await managedMetadata("/docs/setup", "ja");
 assert.equal(metadata?.alternates?.canonical, canonicalURL("/ja/docs/setup"));
 assert.deepEqual(metadata?.alternates?.languages, { en: canonicalURL("/docs/setup"), ja: canonicalURL("/ja/docs/setup"), "x-default": canonicalURL("/docs/setup") });
 assert.equal(metadata?.openGraph && "locale" in metadata.openGraph ? metadata.openGraph.locale : "", "ja_JP");
 assert.equal(await managedMetadata("/features", "ja"), null);
 assert.equal(await managedMetadata("/pricing", "ja"), null);
});

test("a backend without article languages lists none instead of failing", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 400, msg: "invalid locale" }), { status: 400 })) as typeof fetch;
 assert.deepEqual(await publishedIndex("ja"), []);
 assert.deepEqual(await publishedArticle("/docs/setup", "ja"), { managed: true, article: null });
 await assert.rejects(publishedIndex("en"), "interface languages still surface errors");
});
