import assert from "node:assert/strict";
import { test } from "node:test";
import { articles, findArticle, formatContentDate, searchArticles } from "../content/articles";
import { updates } from "../content/updates";
import { policies } from "../content/policies";
import { documentation, documentationGroups, findDocumentation, searchDocumentation } from "../content/docs";

test("content search finds body text in the selected language and supports multiple terms", () => {
  assert.equal(searchArticles("   ", "en").length, articles.length);
  assert.ok(searchArticles("RESEND credentials", "en").some((article) => article.slug === "from-template-to-your-saas"));
  assert.ok(searchArticles("邀请码", "zh").some((article) => article.slug === "understand-invitation-rewards"));
  assert.deepEqual(searchArticles("not-a-published-topic-564", "en"), []);
  assert.equal(findArticle("../privacy"), undefined);
});

test("published article routes and section anchors are unique and bilingual", () => {
  assert.equal(new Set(articles.map((article) => article.slug)).size, articles.length);
  for (const article of articles) {
    assert.match(article.slug, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    assert.match(article.publishedAt, /^\d{4}-\d{2}-\d{2}$/);
    assert.equal(new Set(article.sections.map((section) => section.id)).size, article.sections.length);
    for (const locale of ["en", "zh"] as const) {
      assert.ok(article.title[locale].trim());
      assert.ok(article.summary[locale].trim());
      for (const section of article.sections) {
        assert.ok(section.title[locale].trim());
        assert.ok(section.paragraphs.every((paragraph) => paragraph[locale].trim()));
      }
    }
  }
  for (const update of updates) {
    assert.ok(update.href === "/blog" || findArticle(update.href.replace("/blog/", "")));
  }
});

test("documentation and policy sections have translated titles and body copy", () => {
  for (const section of [...documentation, ...policies.privacy.sections, ...policies.terms.sections]) {
    assert.ok(section.title.en && section.title.zh);
    assert.ok(section.paragraphs.length > 0);
    for (const paragraph of section.paragraphs) assert.ok(paragraph.en && paragraph.zh);
  }
  assert.equal(formatContentDate("2026-09-06", "en"), "September 6, 2026");
  assert.equal(formatContentDate("2026-09-06", "zh"), "2026年9月6日");
});

test("guide routes have unique stable slugs, categories, summaries, and working internal guide links", () => {
  assert.equal(new Set(documentation.map((guide) => guide.id)).size, documentation.length);
  for (const guide of documentation) {
    assert.match(guide.id, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
    assert.equal(findDocumentation(guide.id), guide);
    assert.ok(documentationGroups.some((group) => group.id === guide.group));
    for (const locale of ["en", "zh"] as const) {
      assert.ok(guide.summary[locale].trim());
      assert.ok(guide.summary[locale].length < 200);
      assert.ok(!guide.paragraphs.some((paragraph) => paragraph[locale] === guide.summary[locale]));
    }
    for (const link of guide.links ?? []) {
      assert.ok(!link.href.startsWith("/docs#"));
      if (link.href.startsWith("/docs/")) assert.ok(findDocumentation(link.href.slice(6)));
    }
  }
  for (const group of documentationGroups) assert.ok(documentation.some((guide) => guide.group === group.id));
  for (const slug of ["../privacy", "unknown-guide", "payments/extra", "PAYMENTS"]) assert.equal(findDocumentation(slug), undefined);
});

test("guide directory searches the full article rather than just its teaser", () => {
  assert.equal(searchDocumentation("   ", "en").length, documentation.length);
  assert.ok(searchDocumentation("  STRIPE webhook ", "en").some((guide) => guide.id === "payments"));
  assert.ok(searchDocumentation("管理员 密码", "zh").some((guide) => guide.id === "admin"));
  assert.ok(searchDocumentation("4242", "en").some((guide) => guide.id === "try-demo"));
  assert.deepEqual(searchDocumentation("not-a-published-topic-564", "zh"), []);
});
