import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { createContentTranslation, readContentTranslation, readContentTranslationSettings, translationPhaseLabel, isTerminalTranslationStatus, countZhCharacters, countEnWords, translationLengthState, translationCheckState, otherLocale, hasTranslatableContent, isValidContentPath } from "../lib/content-translation";
import type { ContentTranslationMeta } from "../lib/api";
const original = globalThis.fetch;
afterEach(() => { globalThis.fetch = original; });
const ok = (data: unknown, code = 200) => new Response(JSON.stringify({ code, msg: code === 200 ? "ok" : "accepted", data }), { headers: { "Content-Type": "application/json" } });

test("translation create, query and settings requests build stable paths and intent keys", async () => {
 const calls: Array<{ path: string; method?: string; key: string | null; body: unknown }> = [];
 globalThis.fetch = (async (url, options) => { calls.push({ path: String(url), method: options?.method, key: new Headers(options?.headers).get("Idempotency-Key"), body: options?.body ? JSON.parse(String(options?.body)) : undefined }); return ok({ id: "job-1", article_id: "a", status: "queued", phase: "queued", article_version: 3, created_at: "", updated_at: "" }); }) as typeof fetch;
 const input = { version: 3, source_locale: "zh" as const, target_locale: "en" as const, reason: "AI 翻译：zh → en" };
 await createContentTranslation("admin", "a", input, "intent-1");
 await createContentTranslation("admin", "a", input, "intent-1");
 await readContentTranslation("admin", "job-1");
 await readContentTranslationSettings("admin");
 assert.equal(calls[0].method, "POST"); assert.equal(calls[0].key, "intent-1"); assert.deepEqual(calls[0].body, input); assert.ok(calls[0].path.endsWith("/admin/articles/a/translations"));
 assert.deepEqual(calls[0], calls[1]);
 assert.equal(calls[2].method, undefined); assert.equal(calls[2].key, null); assert.ok(calls[2].path.endsWith("/admin/content-translations/job-1"));
 assert.ok(calls[3].path.endsWith("/admin/content-translation-settings")); assert.equal(calls[3].key, null);
});

test("202 Accepted envelopes resolve for async job creation", async () => {
 let calls = 0;
 globalThis.fetch = (async () => { calls++; return ok({ id: "job-202", article_id: "a", status: "queued", phase: "queued", article_version: 1, created_at: "", updated_at: "" }, 202); }) as typeof fetch;
 const job = await createContentTranslation("admin", "a", { version: 1, source_locale: "zh", target_locale: "en", reason: "AI 翻译：zh → en" }, "intent-202");
 assert.equal(calls, 1);
 assert.equal(job.id, "job-202");
 assert.equal(job.status, "queued");
});

test("path validation mirrors the backend before any request is sent", () => {
 assert.equal(isValidContentPath("blog", "/blog/new-article-flow-test"), true);
 assert.equal(isValidContentPath("blog", "/blog/"), false);
 assert.equal(isValidContentPath("blog", "/blog/我的文章"), false);
 assert.equal(isValidContentPath("blog", "/blog/My-Post"), false);
 assert.equal(isValidContentPath("blog", "/blog/multi-part-slug"), true);
 assert.equal(isValidContentPath("docs", "/docs/overview"), true);
 assert.equal(isValidContentPath("docs", "/blog/overview"), false);
 assert.equal(isValidContentPath("page", "/"), true);
 assert.equal(isValidContentPath("page", "/pricing"), false);
 assert.equal(isValidContentPath("page", "/blog/x"), false);
});

test("empty source drafts are refused before burning a model call", () => {
 const filled = { title: "标题", summary: "摘要", body: "正文", seo_title: "", seo_description: "" };
 assert.equal(hasTranslatableContent(filled), true);
 assert.equal(hasTranslatableContent({ ...filled, title: "  " }), false);
 assert.equal(hasTranslatableContent({ ...filled, body: "" }), false);
 assert.equal(hasTranslatableContent({ title: "", summary: "", body: "", seo_title: "", seo_description: "" }), false);
});

test("phase labels and terminal statuses cover the bounded pipeline", () => {
 assert.deepEqual(translationPhaseLabel("queued"), { en: "Waiting", zh: "等待处理" });
 assert.deepEqual(translationPhaseLabel("translating"), { en: "Translating", zh: "正在翻译" });
 assert.deepEqual(translationPhaseLabel("reviewing"), { en: "Checking", zh: "正在检查" });
 assert.deepEqual(translationPhaseLabel("repairing"), { en: "Fixing", zh: "正在修正" });
 assert.deepEqual(translationPhaseLabel("rechecking"), { en: "Rechecking", zh: "正在复查" });
 assert.deepEqual(translationPhaseLabel("done"), { en: "Finished", zh: "已完成" });
 assert.equal(isTerminalTranslationStatus("succeeded"), true);
 assert.equal(isTerminalTranslationStatus("failed"), true);
 assert.equal(isTerminalTranslationStatus("running"), false);
 assert.equal(isTerminalTranslationStatus("queued"), false);
});

test("length limits count zh characters and en words at exact boundaries", () => {
 const zh = { title: "标 题", summary: "摘要", body: "正文 内容", seo_title: "", seo_description: "" };
 assert.equal(countZhCharacters(zh), 8);
 const en = { title: "a b", summary: "c", body: "d e f", seo_title: "", seo_description: "" };
 assert.equal(countEnWords(en), 6);
 const settings = { enabled: true, locales: ["en", "zh"] as Array<"en" | "zh">, max_zh_characters: 8, max_en_words: 6 };
 assert.deepEqual(translationLengthState(zh, "zh", settings), { count: 8, limit: 8, over: false });
 assert.equal(translationLengthState({ ...zh, title: "标题!" }, "zh", settings).over, true);
 assert.equal(translationLengthState(en, "en", settings).over, false);
 assert.equal(translationLengthState({ ...en, body: "d e f g" }, "en", settings).over, true);
});

test("check state turns stale after manual edits, flags updated sources, never upgrades a warning", () => {
 const meta: ContentTranslationMeta = { source_locale: "zh", source_hash: "s", draft_hash: "d", check_status: "passed", model: "m", completed_at: "2026-10-26T00:00:00Z", job_id: "j" };
 assert.equal(translationCheckState(undefined, false, false), "none");
 assert.equal(translationCheckState(meta, false, false), "passed");
 assert.equal(translationCheckState(meta, true, false), "stale");
 assert.equal(translationCheckState(meta, false, true), "source-updated");
 assert.equal(translationCheckState(meta, true, true), "stale");
 const warning: ContentTranslationMeta = { ...meta, check_status: "warning", issues: [{ code: "format", field: "body", message_zh: "链接缺失" }] };
 assert.equal(translationCheckState(warning, false, false), "warning");
 assert.equal(translationCheckState(warning, false, true), "source-updated");
 assert.equal(otherLocale("zh"), "en");
 assert.equal(otherLocale("en"), "zh");
});
