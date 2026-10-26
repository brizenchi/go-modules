// Acceptance check against an explicitly supplied disposable/local CMS. This
// checks visible initial HTML, after removing every script, against live records.
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import ts from "typescript";
import vm from "node:vm";

const [build, base] = process.argv.slice(2);
if (!build || !base) throw new Error("Usage: node scripts/check-managed-html.mjs <.next> <loopback-api-base>");
const origin = new URL(base);
assert.ok(["127.0.0.1", "localhost", "[::1]"].includes(origin.hostname) && origin.protocol === "http:", "Use an isolated local API");
const sandbox = { exports: {} };
vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../lib/markdown.ts", import.meta.url), "utf8"), { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText, sandbox);
const { markdownBlocks } = sandbox.exports;
function decode(text) {
 return text.replace(/&#x([a-f\d]+);/gi, (_, n) => String.fromCodePoint(parseInt(n, 16))).replace(/&#(\d+);/g, (_, n) => String.fromCodePoint(Number(n))).replace(/&(amp|lt|gt|quot|apos|#x27|#39);/g, (_, n) => ({ amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", "#x27": "'", "#39": "'" })[n]);
}
const normalize = text => decode(text).replace(/\s+/g, " ").trim();
const plain = text => text.replace(/\[([^\]]+)\]\([^\s)]+\)/g, "$1").replace(/\*\*([^*]+)\*\*/g, "$1").replace(/`([^`]+)`/g, "$1");
async function get(path) { const response = await fetch(`${base.replace(/\/$/, "")}${path}`); assert.ok(response.ok, `API ${response.status}`); return (await response.json()).data; }
let pages = 0, blocks = 0;
for (const locale of ["en", "zh", "ja", "ko", "es", "fr", "de"]) {
 const rows = [];
 for (let page = 1; ; page++) { const data = await get(`/site/articles?locale=${locale}&page=${page}&limit=100`); rows.push(...data.items); if (rows.length >= data.total) break; assert.ok(data.items.length, "Pagination progressed"); }
 for (const row of rows) {
  const { article } = await get(`/site/articles/resolve?${new URLSearchParams({ path: row.path, locale })}`);
  // Languages without an interface translation publish articles only.
  if (!["en", "zh"].includes(locale) && !/^\/(blog|docs|features|use-cases|compare)\//.test(row.path)) continue;
  const route = `${locale === "en" ? "" : `/${locale}`}${row.path === "/" ? "" : row.path}` || "/";
  const html = readFileSync(join(build, "server/app", route === "/" ? "index.html" : `${route.slice(1)}.html`), "utf8");
  const rendered = html.replace(/<script\b[^>]*>[\s\S]*?<\/script>/g, "");
  const visible = normalize(rendered.replace(/<\/(?:p|div|li|h[1-6]|section|pre|blockquote)>/g, " ").replace(/<[^>]+>/g, ""));
  assert.ok(visible.includes(normalize(article.translation.title)), `${route}: visible CMS title`);
  assert.ok(visible.includes(normalize(article.translation.summary)), `${route}: visible CMS summary`);
  assert.ok(normalize(html.match(/<title>([^<]+)<\/title>/)?.[1] || "").includes(article.translation.seo_title || article.translation.title), `${route}: published search title`);
  for (const block of markdownBlocks(article.translation.body)) {
   if (block.kind === "image") { if (block.alt) assert.ok(rendered.includes(`alt="${block.alt.replaceAll("&", "&amp;").replaceAll('"', "&quot;")}"`), `${route}: published image alt missing: ${block.alt}`); continue; }
   const texts = block.kind === "list" ? block.items : [block.text];
   for (const source of texts) { const text = normalize(block.kind === "code" ? source : plain(source)); assert.ok(visible.includes(text), `${route}: published ${block.kind} missing from initial HTML: ${text.slice(0, 80)}`); blocks++; }
  }
  pages++;
 }
}
assert.ok(pages > 0, "CMS contains publications");
console.log(`Managed HTML checks passed: ${pages} published language pages, ${blocks} content blocks visible without scripts.`);
