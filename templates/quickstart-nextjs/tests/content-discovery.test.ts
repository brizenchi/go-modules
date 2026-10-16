import assert from "node:assert/strict";
import test from "node:test";
import { contentReferences, latestGuides, relatedContent, uniqueContent, type DiscoveryEntry } from "../lib/content-discovery";

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
