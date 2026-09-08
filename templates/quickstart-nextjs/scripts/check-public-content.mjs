import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const root = ".next/server/app";
const articlePages = ["docs", "blog"].flatMap((directory) =>
  readdirSync(join(root, directory)).filter((file) => file.endsWith(".html")).map((file) => `${directory}/${file}`)
);
const pages = ["index.html", "pricing.html", "docs.html", "blog.html", "updates.html", "contact.html", "privacy.html", "terms.html", ...articlePages];
const canonicals = new Set();
const sitemap = readFileSync(join(root, "sitemap.xml.body"), "utf8");
const withoutScripts = (html) => html.replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, "");

for (const page of pages) {
  const html = readFileSync(join(root, page), "utf8");
  const rendered = withoutScripts(html);
  assert.equal((rendered.match(/<h1\b/g) ?? []).length, 1, `${page}: one H1`);
  assert.equal((rendered.match(/<main\b/g) ?? []).length, 1, `${page}: one main landmark`);
  assert.ok(rendered.includes('id="main-content"'), `${page}: skip-link target`);
  const canonicalHref = html.match(/<link rel="canonical" href="([^"]+)"/)?.[1];
  assert.ok(canonicalHref, `${page}: canonical exists`);
  const canonical = new URL(canonicalHref).href;
  assert.ok(!canonicals.has(canonical), `${page}: unique canonical`);
  canonicals.add(canonical);
  assert.ok(sitemap.includes(`<loc>${canonical}</loc>`), `${page}: sitemap coverage`);
  const schemas = [...html.matchAll(/<script\b[^>]*type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g)].map((match) => JSON.parse(match[1]));
  if (page !== "index.html") {
    const breadcrumb = schemas.find((schema) => schema["@type"] === "BreadcrumbList");
    assert.equal(breadcrumb?.itemListElement.at(-1).item, canonical, `${page}: breadcrumb canonical`);
  }
  if (articlePages.includes(page)) {
    const article = schemas.find((schema) => schema["@type"] === "Article");
    assert.equal(article?.url, canonical, `${page}: article canonical`);
    assert.ok(rendered.includes("<article"), `${page}: article in initial HTML`);
  }
  const ids = [...rendered.matchAll(/\sid="([^"]+)"/g)].map((match) => match[1]);
  assert.equal(new Set(ids).size, ids.length, `${page}: unique anchors`);
  for (const match of rendered.matchAll(/href="#([^"]+)"/g)) {
    assert.ok(ids.includes(match[1]), `${page}: #${match[1]} target exists`);
  }
}

for (const directory of ["docs", "blog"]) {
  const html = withoutScripts(readFileSync(join(root, `${directory}.html`), "utf8"));
  assert.ok(!html.includes("<pre"), `${directory}: directory contains summaries, not configuration blocks`);
  for (const article of articlePages.filter((page) => page.startsWith(`${directory}/`))) {
    assert.ok(html.includes(`href="/${article.replace(/\.html$/, "")}"`), `${article}: crawlable directory link`);
  }
}

console.log(`Public content checks passed: ${pages.length} pages, ${articlePages.length} articles, headings, anchors, canonical URLs, JSON-LD and sitemap.`);
