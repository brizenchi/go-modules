import assert from "node:assert/strict";
import { test } from "node:test";
import { marketingCategories, marketingPages, marketingRoutes, aboutPage, resourcesPage } from "../content/marketing";
import { publicContentPaths } from "../lib/seo";

test("marketing pages have unique published paths and complete bilingual content", () => {
  const paths = new Set(publicContentPaths());
  assert.equal(new Set(marketingRoutes.map(page => page.path)).size, marketingRoutes.length);
  assert.equal(marketingPages.length, 8);
  for (const page of [...marketingPages, aboutPage, resourcesPage]) {
    assert.ok(paths.has(page.path));
    for (const field of [page.title, page.description, page.intro]) {
      assert.ok(field.en.trim()); assert.ok(field.zh.trim());
    }
    for (const link of [...page.related, ...page.sources]) {
      assert.ok(link.label.en && link.label.zh);
      if (link.href.startsWith("/") && !["/credits", "/notes"].includes(link.href)) assert.ok(paths.has(link.href.split("#")[0]), link.href);
    }
    assert.ok(page.sections.length > 0);
  }
  for (const category of marketingCategories) assert.ok(marketingPages.some(page => page.kind === category.kind));
  for (const page of marketingPages) {
    assert.ok(page.steps.length > 0 && page.facts.length > 0 && page.limitations.length > 0 && page.faq.length > 0);
    assert.ok(marketingCategories.some(category => page.path.startsWith(`${category.path}/`)));
  }
});
