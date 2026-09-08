import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { Breadcrumbs } from "../components/breadcrumbs";
import { StructuredData } from "../components/structured-data";
import { articleStructuredData, breadcrumbStructuredData, serializeStructuredData } from "../lib/structured-data";
import { canonicalURL } from "../lib/seo";

const items = [{ href: "/", label: "Home" }, { href: "/docs", label: "Guide" }, { label: "Connect your domains" }];

test("breadcrumb schema describes the visible hierarchy with canonical absolute URLs", () => {
  const schema = breadcrumbStructuredData(items, "/docs/domains?ref=private#configuration");
  assert.equal(schema["@type"], "BreadcrumbList");
  assert.deepEqual(schema.itemListElement.map((item) => item.position), [1, 2, 3]);
  assert.deepEqual(schema.itemListElement.map((item) => item.name), items.map((item) => item.label));
  assert.deepEqual(schema.itemListElement.map((item) => item.item), ["/", "/docs", "/docs/domains"].map(canonicalURL));
});

test("breadcrumbs render an ordered navigation trail and identify the current page", () => {
  const html = renderToStaticMarkup(<Breadcrumbs items={items} path="/docs/domains" label="Breadcrumb" structured />);
  assert.match(html, /<nav[^>]*aria-label="Breadcrumb"/);
  assert.match(html, /<ol>/);
  assert.equal((html.match(/<li>/g) ?? []).length, items.length);
  assert.match(html, /href="\/docs"/);
  assert.match(html, /<span aria-current="page">Connect your domains<\/span>/);
  assert.equal((html.match(/application\/ld\+json/g) ?? []).length, 1);
  const privateHTML = renderToStaticMarkup(<Breadcrumbs items={items} path="/account" label="Breadcrumb" />);
  assert.ok(!privateHTML.includes("application/ld+json"));
  assert.equal(renderToStaticMarkup(<Breadcrumbs items={[]} path="/" label="Breadcrumb" />), "");
});

test("article schema uses only supplied facts and safely renders JSON-LD", () => {
  const guide = articleStructuredData({ title: "Connect your domains", description: "Domain setup", path: "/docs/domains", language: "zh-CN" });
  assert.equal(guide.mainEntityOfPage["@id"], canonicalURL("/docs/domains"));
  assert.equal(guide.inLanguage, "zh-CN");
  assert.ok(!("datePublished" in guide));
  assert.ok(!("author" in guide));
  const article = articleStructuredData({ title: "Article", description: "Summary", path: "/blog/test", language: "en", publishedAt: "2026-09-06" });
  assert.equal(article.datePublished, "2026-09-06");
  const hostile = { headline: '</script><script>alert("example")</script>', description: "中文 < & >" };
  const serialized = serializeStructuredData(hostile);
  assert.ok(!serialized.includes("<"));
  assert.deepEqual(JSON.parse(serialized), hostile);
  const html = renderToStaticMarkup(<StructuredData data={hostile} />);
  assert.equal((html.match(/<script/g) ?? []).length, 1);
  assert.equal((html.match(/<\/script>/g) ?? []).length, 1);
});
