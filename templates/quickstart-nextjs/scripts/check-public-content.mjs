import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const root = ".next/server/app";
const sitemap = readFileSync(join(root, "sitemap.xml.body"), "utf8");
const urls = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map(match => new URL(match[1]));
const paths = new Set(urls.map(url => url.pathname));
const pages = urls.map(url => url.pathname === "/" ? "index.html" : `${url.pathname.slice(1)}.html`);
const articlePages = pages.filter(page => /^(zh\/)?(docs|blog)\//.test(page));
const canonicals = new Set();
for (const path of ["/features", "/use-cases", "/compare", "/examples", "/about", "/resources"]) {
  assert.ok(paths.has(path) && paths.has(`/zh${path}`), `${path}: bilingual route coverage`);
}
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
  const path = new URL(canonical).pathname;
  const chinese = path === "/zh" || path.startsWith("/zh/");
  assert.ok(rendered.includes(`lang="${chinese ? "zh-CN" : "en"}"`), `${page}: server-rendered language`);
  assert.ok(!html.includes('content="noindex'), `${page}: public content is indexable`);
  for (const language of ["en", "zh-CN", "x-default"]) {
    const alternate = [...html.matchAll(/<link[^>]+>/g)].find(match => match[0].includes(`hrefLang="${language}"`) || match[0].includes(`hreflang="${language}"`))?.[0];
    assert.ok(alternate, `${page}: ${language} alternative exists`);
    assert.ok(paths.has(new URL(alternate.match(/href="([^"]+)"/)[1]).pathname), `${page}: alternative is a published page`);
  }
  for (const match of rendered.matchAll(/href="(\/[^"#?]*)[^"]*"/g)) {
    const target = match[1];
    if (/^\/(?:zh(?:\/|$)|features(?:\/|$)|use-cases(?:\/|$)|compare(?:\/|$)|examples(?:\/|$)|docs(?:\/|$)|blog(?:\/|$)|resources$|about$)/.test(target)) {
      assert.ok(paths.has(target), `${page}: published internal link ${target}`);
    }
  }
  assert.ok(!canonicals.has(canonical), `${page}: unique canonical`);
  canonicals.add(canonical);
  assert.ok(sitemap.includes(`<loc>${canonical}</loc>`), `${page}: sitemap coverage`);
  const schemas = [...html.matchAll(/<script\b[^>]*type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g)].map((match) => JSON.parse(match[1]));
  if (page !== "index.html" && page !== "zh.html") {
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

for (const directory of ["docs", "blog", "zh/docs", "zh/blog"]) {
  const html = withoutScripts(readFileSync(join(root, `${directory}.html`), "utf8"));
  assert.ok(!html.includes("<pre"), `${directory}: directory contains summaries, not configuration blocks`);
  for (const article of articlePages.filter((page) => page.startsWith(`${directory}/`))) {
    assert.ok(html.includes(`href="/${article.replace(/\.html$/, "")}"`), `${article}: crawlable directory link`);
  }
}

for (const path of ["account", "billing", "referrals", "credits", "files", "notes", "login", "invite", "admin"]) {
  const html = readFileSync(join(root, `${path}.html`), "utf8");
  assert.match(html, /<meta name="robots" content="[^"]*noindex/, `${path}: private page noindex`);
  assert.ok(!html.includes('rel="canonical"'), `${path}: no public canonical inherited`);
}

console.log(`Public content checks passed: ${pages.length} pages, ${articlePages.length} articles, headings, anchors, canonical URLs, JSON-LD and sitemap.`);
