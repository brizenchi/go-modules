import assert from "node:assert/strict";
import test from "node:test";
import { imageMarkdown, imagePlan, insertImage, insertImageAt, insertPlannedImage, mapLimit, parseImagePlan, translationTargets, writingPlan } from "../lib/article-writing";
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

test("AI writing starts in the open language when it is ticked, otherwise the first ticked one", () => {
  assert.deepEqual(writingPlan(["ja", "zh", "en"], "zh"), { source: "zh", targets: ["en", "ja"] });
  assert.deepEqual(writingPlan(["de", "ko"], "zh"), { source: "ko", targets: ["de"] });
  assert.deepEqual(writingPlan(["en"], "en"), { source: "en", targets: [] });
  assert.equal(writingPlan([], "en"), null);
});

test("translations run a few at a time and each finishes on its own", async () => {
  let running = 0; let peak = 0; const done: string[] = [];
  await mapLimit(["en", "ja", "ko", "es", "fr", "de"], 3, async item => {
    running++; peak = Math.max(peak, running);
    await new Promise(resolve => setTimeout(resolve, item === "en" ? 20 : 5));
    running--; done.push(item);
  });
  assert.equal(peak, 3);
  assert.equal(done.length, 6);
  assert.notEqual(done[0], "en", "a slow language does not hold the others back");
});

test("one uploaded image goes under the same section in every language with that language's alt text", () => {
  const plan = imagePlan("zh", [{ section: "如何开始", alt: "设置界面", description: "成员列表" }], {
    en: [{ section: "Getting started", alt: "Settings screen", description: "Members" }],
    ja: [{ section: "はじめかた（誤訳）", alt: "設定画面", description: "メンバー" }],
    de: []
  });
  assert.deepEqual(Object.keys(plan.items[0]).sort(), ["en", "ja", "zh"], "a language without matching suggestions is left out");
  const url = "/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3";
  const bodies = {
    zh: "介绍。\n\n## 如何开始\n\n步骤。\n\n## 常见问题\n\n问答。",
    en: "Intro.\n\n## Getting started\n\nSteps.\n\n## FAQ\n\nAnswers.",
    ja: "紹介。\n\n## はじめに\n\n手順。\n\n## よくある質問\n\n回答。"
  };
  const next = insertPlannedImage(bodies, plan.items[0], plan.source, url);
  assert.ok(next.zh?.includes(`## 如何开始\n\n![设置界面](${url})`));
  assert.ok(next.en?.includes(`## Getting started\n\n![Settings screen](${url})`));
  assert.ok(next.ja?.includes(`## はじめに\n\n![設定画面](${url})`), "a heading the model renamed falls back to the same position");
  assert.equal(next.de, undefined);
});

test("saved image suggestions are restored only when well formed", () => {
  const plan = imagePlan("en", [{ section: "Setup", alt: "Members", description: "" }], {});
  assert.deepEqual(parseImagePlan(JSON.stringify(plan)), plan);
  for (const raw of [null, "", "{", "[]", JSON.stringify({ source: "xx", items: [] }), JSON.stringify({ source: "en", items: [{ en: { section: 1, alt: "a", description: "" } }] })]) {
    assert.equal(parseImagePlan(raw), null, String(raw));
  }
});
