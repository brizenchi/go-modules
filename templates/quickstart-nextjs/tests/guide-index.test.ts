import assert from "node:assert/strict";
import test from "node:test";
import { filterGuideIndex, type GuideIndexEntry } from "../lib/guide-index";

const entry = (kind: GuideIndexEntry["kind"], title: string, summary: string): GuideIndexEntry => ({
  id: `${kind}-${title}`, path: `/${kind}/fixture`, kind,
  translation: { title, summary }
});
const rows = [entry("features", "Credits", "Shared accounting"), entry("docs", "Configure payments", "Connect your Stripe account"), entry("blog", "邀请奖励", "查看注册归因和奖励进展"), entry("use-cases", "Subscriptions", "A product workflow")];

test("guide discovery keeps learning content separate from product capability pages", () => {
  assert.deepEqual(filterGuideIndex(rows, "resources", "all", ""), [rows[1], rows[2]]);
  assert.deepEqual(filterGuideIndex(rows, "resources", "docs", ""), [rows[1]]);
  assert.deepEqual(filterGuideIndex(rows, "resources", "blog", ""), [rows[2]]);
  assert.deepEqual(filterGuideIndex(rows, "features", "all", ""), [rows[0]]);
});

test("guide search combines title and summary terms with the selected category", () => {
  assert.deepEqual(filterGuideIndex(rows, "resources", "all", "  PAYMENTS stripe "), [rows[1]]);
  assert.deepEqual(filterGuideIndex(rows, "resources", "all", "邀请 归因"), [rows[2]]);
  assert.deepEqual(filterGuideIndex(rows, "resources", "docs", "邀请"), []);
  assert.deepEqual(filterGuideIndex(rows, "resources", "all", "no-match"), []);
});
