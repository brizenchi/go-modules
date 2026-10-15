import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { managedMetadata, publishedPaths, publishedArticle } from "../lib/public-content";
import { appEnv } from "../lib/env";
import { canonicalURL } from "../lib/seo";

const originalFetch = globalThis.fetch;
const configured = appEnv.apiConfigured;
afterEach(() => { globalThis.fetch = originalFetch; appEnv.apiConfigured = configured; });
const article = { path: "/blog/custom", locales: ["en"], published_at: "2026-10-15T00:00:00Z", translation: { title: "Custom article", summary: "Published summary", seo_title: "", seo_description: "" } };
const response = (data: unknown) => new Response(JSON.stringify({ code: 200, data }), { headers: { "Content-Type": "application/json" } });

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

test("only absent legacy content endpoints permit code fallback", async () => {
 appEnv.apiConfigured = true;
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 404 }), { status: 404 })) as typeof fetch;
 assert.deepEqual(await publishedArticle("/blog/removed", "en"), { managed: false, article: null });
 globalThis.fetch = (async () => new Response(JSON.stringify({ code: 503 }), { status: 503 })) as typeof fetch;
 await assert.rejects(publishedArticle("/blog/removed", "en"));
});
