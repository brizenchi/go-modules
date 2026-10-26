import assert from "node:assert/strict";
import test from "node:test";
import { imageMarkdown, insertImage, insertImageAt, translationTargets } from "../lib/article-writing";
import { markdownBlocks } from "../lib/markdown";

const image = imageMarkdown("成员\n列表 [示例]", "/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3");

test("image markdown keeps alt text on one line without brackets", () => {
  assert.equal(image, "![成员 列表 示例](/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3)");
});

test("a suggested image goes right below its section heading", () => {
  const body = "介绍。\n\n## 如何开始\n\n先配置支付。\n\n## 常见问题\n\n问答。";
  const next = insertImage(body, image, "如何开始");
  assert.equal(next, "介绍。\n\n## 如何开始\n\n" + image + "\n\n先配置支付。\n\n## 常见问题\n\n问答。");
  assert.deepEqual(markdownBlocks(next).map(block => block.kind), ["paragraph", "heading", "image", "paragraph", "heading", "paragraph"]);
  assert.ok(insertImage(body, image, "不存在").endsWith(`问答。\n\n${image}\n`), "unknown sections append at the end");
});

test("an image inserted at the cursor becomes its own paragraph", () => {
  assert.equal(insertImageAt("第一段。第二段。", image, 4), `第一段。\n\n${image}\n\n第二段。`);
  assert.equal(insertImageAt("", image, 0), `${image}\n`);
  assert.equal(insertImageAt("正文", image, 99), `正文\n\n${image}\n`);
});

test("translation offers every language except the source", () => {
  assert.deepEqual(translationTargets("zh"), ["en", "ja", "ko", "es", "fr", "de"]);
  assert.ok(!translationTargets("ja").includes("ja") && translationTargets("ja").length === 6);
});
