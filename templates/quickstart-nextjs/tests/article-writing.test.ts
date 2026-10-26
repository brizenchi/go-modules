import assert from "node:assert/strict";
import test from "node:test";
import { canPublish, defaultAlt, imageMarkdown, insertImageAt, insertImageAtCursor, mapLimit, publishOptions, translationTargets, writingPlan } from "../lib/article-writing";
import { markdownBlocks } from "../lib/markdown";

const image = imageMarkdown("成员\n列表 [示例]", "/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3");

test("image markdown keeps alt text on one line without brackets", () => {
  assert.equal(image, "![成员 列表 示例](/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3)");
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

test("an image inserted from the body editor reports where its alt text is, so it can be selected", () => {
  const url = "/api/v1/site/images/00000000-0000-4000-8000-000000000000?w=4&h=3";
  const inserted = insertImageAtCursor("第一段。第二段。", "团队 设置", url, 4);
  assert.equal(inserted.body, `第一段。\n\n![团队 设置](${url})\n\n第二段。`);
  assert.equal(inserted.body.slice(inserted.altStart, inserted.altEnd), "团队 设置");
  const empty = insertImageAtCursor("", "", url, 0);
  assert.equal(empty.body, `![](${url})\n`);
  assert.equal(empty.altStart, 2); assert.equal(empty.altEnd, 2);
});

test("file names become alt text unless they say nothing about the picture", () => {
  assert.equal(defaultAlt("team-member_settings.png"), "team member settings");
  assert.equal(defaultAlt("成员列表.webp"), "成员列表");
  for (const name of ["image.png", "IMG_2041.jpeg", "Screenshot 2026-10-26 at 10.12.03.png", "截屏2026-10-26 10.12.03.png", "屏幕截图 2026-10-26.png", "20261026.jpg", "pasted image.png"]) assert.equal(defaultAlt(name), "", name);
});

test("publishing offers complete languages that are new or changed", () => {
  const text = (title: string, summary = "S", body = "B") => ({ title, summary, body, seo_title: "", seo_description: "" });
  const options = publishOptions({
    translations: { en: text("Guide"), zh: text("指南", "摘要", "改过的正文"), ja: text("ガイド", ""), de: text("Anleitung") },
    published: { zh: { ...text("指南", "摘要"), published_at: "2026-10-01T00:00:00Z" }, de: { ...text("Anleitung"), published_at: "2026-10-01T00:00:00Z" } }
  });
  assert.deepEqual(options.filter(canPublish).map(option => option.language), ["en", "zh"]);
  assert.deepEqual(options.find(option => option.language === "ja"), { language: "ja", state: "draft", complete: false });
  assert.equal(options.find(option => option.language === "de")?.state, "published");
  assert.equal(options.find(option => option.language === "ko")?.state, "missing");
});
