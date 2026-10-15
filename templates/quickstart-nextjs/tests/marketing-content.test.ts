import assert from "node:assert/strict";
import { test } from "node:test";
import { marketingCategories, marketingPages, marketingRoutes, aboutPage, resourcesPage } from "../content/marketing";
import { publicContentPaths } from "../lib/seo";

test("marketing pages have unique published paths and complete bilingual content", () => {
  const paths = new Set(publicContentPaths());
  assert.equal(new Set(marketingRoutes.map(page => page.path)).size, marketingRoutes.length);
  assert.equal(marketingPages.length, 9);
  for (const page of [...marketingPages, aboutPage, resourcesPage]) {
    assert.ok(paths.has(page.path));
    for (const field of [page.title, page.description, page.intro]) {
      assert.ok(field.en.trim()); assert.ok(field.zh.trim());
    }
    for (const link of [...page.related, ...page.sources]) {
      assert.ok(link.label.en && link.label.zh);
      if (link.href.startsWith("/") && !["/credits", "/notes", "/files"].includes(link.href)) assert.ok(paths.has(link.href.split("#")[0]), link.href);
    }
    assert.ok(page.sections.length > 0);
  }
  for (const category of marketingCategories) assert.ok(marketingPages.some(page => page.kind === category.kind));
  for (const page of marketingPages) {
    assert.ok(page.steps.length > 0 && page.facts.length > 0 && page.limitations.length > 0 && page.faq.length > 0);
    assert.ok(marketingCategories.some(category => page.path.startsWith(`${category.path}/`)));
  }
});

test("public capability copy matches the read-only account dialog and stateless AI integration", () => {
  const authentication = marketingPages.find(page => page.path === "/features/authentication")!;
  const accountCopy = JSON.stringify(authentication.facts);
  assert.doesNotMatch(accountCopy, /session refresh|会话刷新/iu);
  assert.match(accountCopy, /read.only|只读/iu);
  const usage = marketingPages.find(page => page.path === "/use-cases/usage-based-products")!;
  const aiAnswer = usage.faq.find(item => /AI/.test(item.question.en))!;
  assert.match(aiAnswer.answer.en, /chat/iu);
  assert.match(aiAnswer.answer.zh, /聊天/u);
  assert.match(aiAnswer.answer.en, /stateless/iu);
  assert.match(aiAnswer.answer.zh, /由具体产品定义/u);
  assert.doesNotMatch(aiAnswer.answer.en, /daily|per.day/iu);
});
