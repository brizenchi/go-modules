import assert from "node:assert/strict";
import test from "node:test";
import { availableDirectories, contentReferences, isContentDirectory, isEditorialPath, latestGuides, linkablePath, navigationEntries, relatedContent, uniqueContent, type DiscoveryEntry } from "../lib/content-discovery";

function entry(path: string, kind: DiscoveryEntry["kind"], body = "", published_at?: string): DiscoveryEntry {
  return { id: path, path, kind, published_at, links: contentReferences(body), translation: { title: path, summary: "Fixture summary" } };
}

test("recent guides use real publication times, omit product pages and do not duplicate destinations", () => {
  const old = entry("/blog/old", "blog", "", "2026-01-01");
  const recent = entry("/docs/recent", "docs", "", "2026-10-16");
  const middle = entry("/blog/middle", "blog", "", "2026-08-01");
  const rows = [old, recent, recent, entry("/features/recent", "features", "", "2026-10-17"), middle];
  assert.deepEqual(latestGuides(rows).map(row => row.path), [recent.path, middle.path, old.path]);
  assert.deepEqual(latestGuides([]), []);
});

test("related content prioritizes explicit links and backlinks, excludes itself and missing publications", () => {
  const current = entry("/features/subscriptions", "features", "See [payments](/docs/payments#setup) and [removed](/docs/removed).");
  const peer = entry("/features/credits", "features");
  const payments = entry("/docs/payments", "docs");
  const scenario = entry("/use-cases/subscription-saas", "use-cases", "[Subscriptions](/features/subscriptions)");
  const unrelated = entry("/blog/unrelated", "blog");
  const rows = [current, peer, unrelated, scenario, payments, payments];
  assert.deepEqual(relatedContent(rows, current.path).map(row => row.path), [payments.path, scenario.path, peer.path]);
  assert.deepEqual(relatedContent([], current.path), []);
  assert.deepEqual(relatedContent(rows, "/unpublished"), []);
  assert.deepEqual(relatedContent([current], current.path), []);
});

test("link matching uses complete destinations and ignores external or similarly named pages", () => {
  const current = entry("/docs/account", "docs", "[External](https://example.test/blog/other) [Other account](/docs/account-other)");
  const linked = entry("/docs/account-other", "docs");
  const external = entry("/blog/other", "blog");
  const backlink = entry("/features/auth", "features", "[Chinese guide](/zh/docs/account?from=feature)");
  assert.deepEqual(relatedContent([current, external, linked, backlink], current.path).map(row => row.path), [linked.path, backlink.path]);
});

test("the discovery catalog accepts only public content paths matching their kind", () => {
  const valid = entry("/docs/setup", "docs");
  assert.deepEqual(uniqueContent([valid, valid, entry("/account", "docs"), entry("//example.test/page", "blog")]), [valid]);
  assert.deepEqual(latestGuides([entry("/docs/undated", "docs"), entry("/blog/invalid-date", "blog", "", "invalid")]), []);
});

test("header navigation receives only titles and order of published product pages", () => {
  const entries = navigationEntries([
    { id: "1", path: "/features/credits", kind: "features", position: 3, translation: { title: "Credits", summary: "Long summary" }, links: ["/docs/payments"] },
    { id: "2", path: "/features/credits", kind: "features", position: 3, translation: { title: "Credits", summary: "Duplicate" } },
    { id: "3", path: "/docs/payments", kind: "docs", translation: { title: "Payments", summary: "" } },
    { id: "4", path: "/compare", kind: "compare", translation: { title: "Directory", summary: "" } }
  ]);
  assert.deepEqual(entries, [{ path: "/features/credits", kind: "features", title: "Credits", position: 3 }]);
});

test("a directory exists only with published pages, and Guides with docs or articles", () => {
  const entry = (path: string, kind: DiscoveryEntry["kind"]): DiscoveryEntry => ({ id: path, path, kind, translation: { title: path, summary: "" } });
  assert.deepEqual(availableDirectories([entry("/features/credits", "features"), entry("/docs/setup", "docs"), entry("/compare", "compare")], new Set(["/about"])), ["/features", "/docs", "/resources", "/about"]);
  assert.deepEqual(availableDirectories([entry("/blog/news", "blog")], new Set()), ["/blog", "/resources"]);
  // Without a CMS the bundled about and updates pages always exist.
  assert.deepEqual(availableDirectories([], null), ["/about", "/updates"]);
  assert.equal(isContentDirectory("/compare"), true);
  assert.equal(isContentDirectory("/pricing"), false);
  assert.equal(isContentDirectory("/constructor"), false);
});

test("published text links only to pages and directories that exist", () => {
  const linkable = linkablePath(new Set(["/docs/setup"]), new Set(["/docs"]));
  assert.equal(linkable("/docs/setup"), true);
  assert.equal(linkable("/zh/docs/setup?ref=x#step"), true);
  assert.equal(linkable("/docs/removed"), false);
  assert.equal(linkable("/blog"), false);
  assert.equal(linkable("/docs"), true);
  for (const path of ["/", "/pricing", "/account", "/docs/Not_A_Slug"]) assert.equal(linkable(path), true, path);
  assert.equal(linkablePath(new Set(), null)("/blog"), true, "unknown directories stay linked");
  assert.equal(isEditorialPath("/compare/x"), true);
});
