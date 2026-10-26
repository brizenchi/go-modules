import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(process.argv[2] || ".next", "server/app");
const sitemap = readFileSync(join(root, "sitemap.xml.body"), "utf8");
const urls = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map(match => new URL(match[1]));
const paths = new Set(urls.map(url => url.pathname));
const pages = urls.map(url => url.pathname === "/" ? "index.html" : `${url.pathname.slice(1)}.html`);
// Article languages beyond the interface languages: /ja/docs/setup and so on.
const languageTags = { en: "en", zh: "zh-CN", ja: "ja", ko: "ko", es: "es", fr: "fr", de: "de" };
const prefixed = "zh|ja|ko|es|fr|de";
const languageOf = path => path.match(new RegExp(`^/(${prefixed})(?=/|$)`))?.[1] || "en";
const baseOf = path => path.replace(new RegExp(`^/(${prefixed})(?=/|$)`), "") || "/";
const inLanguage = (base, language) => language === "en" ? base : base === "/" ? `/${language}` : `/${language}${base}`;
const articlePages = pages.filter(page => new RegExp(`^((${prefixed})/)?(docs|blog)/`).test(page));
const canonicals = new Set();
const brands = new Map();
const pageFile = path => path === "/" ? "index.html" : `${path.slice(1)}.html`;
const shellPaths = new Set();
// First-install directories/home still work before editorial initialization,
// but must not masquerade as indexable publications or enter the sitemap.
for (const base of ["/", "/features", "/use-cases", "/compare", "/resources", "/docs", "/blog", "/about", "/updates"]) {
  for (const path of [base, base === "/" ? "/zh" : `/zh${base}`]) {
    if (paths.has(path)) continue;
    const file = join(root, pageFile(path));
    assert.ok(existsSync(file), `${path}: structural route remains available before initialization`);
    const html = readFileSync(file, "utf8");
    assert.match(html, /<meta name="robots" content="[^\"]*noindex/, `${path}: empty structural page is not indexed`);
    shellPaths.add(path);
  }
}
for (const path of ["/features", "/use-cases", "/compare", "/examples", "/about", "/resources"]) {
  assert.ok((paths.has(path) || shellPaths.has(path)) && (paths.has(`/zh${path}`) || shellPaths.has(`/zh${path}`)), `${path}: bilingual route coverage`);
}
const withoutScripts = (html) => html.replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, "");

for (const page of pages) {
  const html = readFileSync(join(root, page), "utf8");
  const rendered = withoutScripts(html);
  assert.equal((rendered.match(/<h1\b/g) ?? []).length, 1, `${page}: one H1`);
  assert.equal((rendered.match(/<main\b/g) ?? []).length, 1, `${page}: one main landmark`);
  assert.ok(rendered.includes('id="main-content"'), `${page}: skip-link target`);
  if (page === "index.html" || page === "zh.html") assert.ok(!rendered.includes('class="content-breadcrumbs"'), `${page}: homepage has no synthetic Index breadcrumb`);
  const canonicalHref = html.match(/<link rel="canonical" href="([^"]+)"/)?.[1];
  assert.ok(canonicalHref, `${page}: canonical exists`);
  const canonical = new URL(canonicalHref).href;
  const path = new URL(canonical).pathname;
  assert.ok(rendered.includes(`lang="${languageTags[languageOf(path)]}"`), `${page}: server-rendered language`);
  assert.ok(!html.includes('content="noindex'), `${page}: public content is indexable`);
  const basePath = baseOf(path);
  const expectedLanguages = [
    ...Object.entries(languageTags).filter(([language]) => paths.has(inLanguage(basePath, language))).map(([, tag]) => tag),
    "x-default"
  ];
  for (const language of [...Object.values(languageTags), "x-default"]) {
    const alternate = [...html.matchAll(/<link[^>]+>/g)].find(match => match[0].includes(`hrefLang="${language}"`) || match[0].includes(`hreflang="${language}"`))?.[0];
    if (!expectedLanguages.includes(language)) {
      assert.ok(!alternate, `${page}: unpublished ${language} alternative is absent`);
      continue;
    }
    assert.ok(alternate, `${page}: ${language} alternative exists`);
    assert.ok(paths.has(new URL(alternate.match(/href="([^"]+)"/)[1]).pathname), `${page}: alternative is a published page`);
  }
  for (const match of rendered.matchAll(/href="(\/[^"#?]*)[^"]*"/g)) {
    const target = match[1];
    if (/^\/(?:(?:zh|ja|ko|es|fr|de)(?:\/|$)|features(?:\/|$)|use-cases(?:\/|$)|compare(?:\/|$)|examples(?:\/|$)|docs(?:\/|$)|blog(?:\/|$)|resources$|about$)/.test(target)) {
      assert.ok(paths.has(target) || shellPaths.has(target), `${page}: available internal link ${target}`);
    }
  }
  assert.ok(!canonicals.has(canonical), `${page}: unique canonical`);
  canonicals.add(canonical);
  const title = html.match(/<title>([^<]*)<\/title>/)?.[1] || "";
  const brand = title.includes(" · ") ? title.slice(title.lastIndexOf(" · ") + 3) : "";
  assert.ok(brand, `${page}: title ends with the site brand`);
  brands.set(brand, [...(brands.get(brand) || []), page]);
  assert.match(html, /<meta name="description" content="[^"]+"/, `${page}: search result description`);
  assert.ok(sitemap.includes(`<loc>${canonical}</loc>`), `${page}: sitemap coverage`);
  const schemas = [...html.matchAll(/<script\b[^>]*type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g)].map((match) => JSON.parse(match[1]));
  if (page !== "index.html" && page !== "zh.html") {
    const breadcrumb = schemas.find((schema) => schema["@type"] === "BreadcrumbList");
    assert.equal(breadcrumb?.itemListElement.at(-1).item, canonical, `${page}: breadcrumb canonical`);
  }
  if (articlePages.includes(page)) {
    const article = schemas.find((schema) => schema["@type"] === (new RegExp(`^((${prefixed})/)?docs/`).test(page) ? "TechArticle" : "BlogPosting"));
    assert.equal(article?.url, canonical, `${page}: article canonical`);
    assert.ok(rendered.includes("<article"), `${page}: article in initial HTML`);
  }
  if (new RegExp(`^((${prefixed})/)?(features|use-cases|compare)/`).test(page)) {
    assert.equal(schemas.find(schema => schema["@type"] === "WebPage")?.url, canonical, `${page}: product page is a WebPage, not an Article`);
  }
  if (page === "index.html" || page === "zh.html") {
    const types = schemas.flatMap(schema => schema["@graph"] || [schema]).map(node => node["@type"]);
    for (const type of ["Organization", "WebSite"]) assert.ok(types.includes(type), `${page}: ${type} structured data`);
  }
  if (rendered.includes('id="faq"')) assert.ok(schemas.some(schema => schema["@type"] === "FAQPage"), `${page}: visible FAQ has FAQPage data`);
  const ids = [...rendered.matchAll(/\sid="([^"]+)"/g)].map((match) => match[1]);
  assert.equal(new Set(ids).size, ids.length, `${page}: unique anchors`);
  for (const match of rendered.matchAll(/href="#([^"]+)"/g)) {
    assert.ok(ids.includes(match[1]), `${page}: #${match[1]} target exists`);
  }
}

// llms.txt may only point AI assistants to published pages or structural directories.
const llms = readFileSync(join(root, "llms.txt.body"), "utf8");
assert.match(llms, /^# .+\n\n> .+/, "llms.txt: title and summary");
for (const match of llms.matchAll(/\]\((https?:\/\/[^)]+)\)/g)) {
  const url = new URL(match[1]);
  if (url.origin !== urls[0].origin) continue;
  assert.ok(paths.has(url.pathname) || shellPaths.has(url.pathname), `llms.txt: ${url.pathname} is published`);
}

// A directory is listed only when it has pages: an empty section stays out of the sitemap.
for (const path of paths) {
  const base = path.replace(/^\/zh(?=\/|$)/, "") || "/";
  if (!["/features", "/use-cases", "/compare", "/docs", "/blog"].includes(base)) continue;
  const main = withoutScripts(readFileSync(join(root, pageFile(path)), "utf8")).match(/<main\b[^>]*>([\s\S]*?)<\/main>/)?.[1] || "";
  assert.match(main, new RegExp(`href="${path}/[a-z0-9-]+"`), `${path}: listed directory links at least one of its pages`);
}

// Pages rendered from static metadata must still carry the brand saved in admin.
assert.equal(brands.size, 1, `one brand across public titles: ${[...brands].map(([brand, files]) => `${brand} (${files.slice(0, 4).join(", ")})`).join("; ")}`);

for (const directory of ["docs", "blog", "zh/docs", "zh/blog"]) {
  const html = withoutScripts(readFileSync(join(root, `${directory}.html`), "utf8"));
  assert.ok(!html.includes("<pre"), `${directory}: directory contains summaries, not configuration blocks`);
  for (const article of articlePages.filter((page) => page.startsWith(`${directory}/`))) {
    assert.ok(html.includes(`href="/${article.replace(/\.html$/, "")}"`), `${article}: crawlable directory link`);
  }
}

for (const path of ["account", "billing", "referrals", "credits", "login", "invite", "admin"]) {
  const html = readFileSync(join(root, `${path}.html`), "utf8");
  assert.match(html, /<meta name="robots" content="[^"]*noindex/, `${path}: private page noindex`);
  assert.ok(!html.includes('rel="canonical"'), `${path}: no public canonical inherited`);
  if (["account", "billing", "referrals", "credits"].includes(path)) {
    const rendered = withoutScripts(html);
    for (const tag of ["header", "main", "h1"]) {
      assert.equal((rendered.match(new RegExp(`<${tag}\\b`, "g")) ?? []).length, 1, `${path}: one ${tag} inside the shared account layout`);
    }
    assert.equal((rendered.match(/aria-label="Account navigation"/g) ?? []).length, 1, `${path}: one account navigation`);
  }
}

// A footer link is useful, but public product/learning paths must also be
// discoverable through the header and page content. Check the built HTML so a
// client-only filter or hard-coded recommendation cannot disguise a missing link.
const edges = new Map();
const graphPaths = new Set([...paths, ...shellPaths]);
for (const path of graphPaths) {
  const rendered = withoutScripts(readFileSync(join(root, pageFile(path)), "utf8"));
  const primary = [
    rendered.match(/<header\b[^>]*>([\s\S]*?)<\/header>/)?.[1] || "",
    rendered.match(/<main\b[^>]*>([\s\S]*?)<\/main>/)?.[1] || ""
  ].join("");
  const targets = [...primary.matchAll(/href="(\/[^"#]*)"/g)]
    .map(match => new URL(match[1].replaceAll("&amp;", "&"), urls[0]))
    .filter(url => url.origin === urls[0].origin && graphPaths.has(url.pathname))
    .map(url => url.pathname);
  edges.set(path, targets);
}
const depths = {};
for (const home of ["/", "/zh"]) {
  const depth = new Map([[home, 0]]);
  const queue = [home];
  for (const source of queue) {
    for (const target of edges.get(source) || []) {
      const language = languageOf(target);
      // Each interface graph stays in its language; article languages join either.
      if ((language === "en" || language === "zh") && (language === "zh") !== (home === "/zh")) continue;
      if (!depth.has(target)) { depth.set(target, depth.get(source) + 1); queue.push(target); }
    }
  }
  depths[home] = depth;
}
for (const path of paths) {
  const language = languageOf(path);
  const depth = language === "en" ? depths["/"].get(path) : language === "zh" ? depths["/zh"].get(path) : Math.min(depths["/"].get(path) ?? Infinity, depths["/zh"].get(path) ?? Infinity);
  assert.ok(depth !== undefined && depth <= 3, `${path}: reachable within three clicks without the global footer (depth: ${depth ?? "unreachable"})`);
}
for (const home of ["/", "/zh"]) {
  const guidePage = home === "/" ? "resources.html" : "zh/resources.html";
  const guideMain = withoutScripts(readFileSync(join(root, guidePage), "utf8")).match(/<main\b[^>]*>([\s\S]*?)<\/main>/)?.[1] || "";
  for (const directory of ["docs", "blog", "updates"]) {
    const target = `${home === "/zh" ? "/zh" : ""}/${directory}`;
    if (paths.has(target)) assert.ok(guideMain.includes(`href="${target}"`), `${guidePage}: real ${directory} directory link`);
    else assert.ok(!guideMain.includes(`href="${target}"`), `${guidePage}: no link to the empty ${directory} directory`);
  }
}

console.log(`Public content checks passed: ${pages.length} pages, ${articlePages.length} articles, headings, anchors, canonical URLs, JSON-LD, sitemap and three-click discovery without the global footer.`);

for (const path of new Set([...paths, ...shellPaths])) {
  const base = baseOf(path);
  if (!/^\/(docs|blog|updates)(?:\/|$)/.test(base)) continue;
  const html = readFileSync(join(root, pageFile(path)), "utf8");
  const schemas = [...html.matchAll(/<script\b[^>]*type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g)].map(match => JSON.parse(match[1]));
  const breadcrumb = schemas.find(schema => schema["@type"] === "BreadcrumbList");
  // Home › section › page: the section is the page's own directory, whatever the menus show.
  const directory = `${path.startsWith("/zh/") ? "/zh" : ""}/${base.split("/")[1]}`;
  const items = breadcrumb?.itemListElement || [];
  assert.equal(items.length, base.split("/").length === 2 ? 2 : 3, `${path}: two levels below home`);
  assert.equal(new URL(items[1]?.item || "https://invalid.test").pathname, directory, `${path}: breadcrumb parent is its own section`);
  if (["/docs", "/blog", "/updates"].includes(base)) assert.match(html, /data-guide-navigation/, `${path}: sibling guide navigation in initial HTML`);
}
console.log("Breadcrumb checks passed: documentation, articles and updates sit directly below home.");
