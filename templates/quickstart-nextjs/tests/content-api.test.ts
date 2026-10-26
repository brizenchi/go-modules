import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { saveContent, publishContent } from "../lib/content-api";
import { apiRequest } from "../lib/api";
const original = globalThis.fetch;
afterEach(() => { globalThis.fetch = original; });
const ok = () => new Response(JSON.stringify({ code: 200, data: {} }), { headers: { "Content-Type": "application/json" } });
test("article drafts and language publication carry revision and stable intent keys", async () => {
 const calls: Array<{ path: string; method?: string; key: string | null; body: unknown }> = [];
 globalThis.fetch = (async (url, options) => { calls.push({ path: String(url), method: options?.method, key: new Headers(options?.headers).get("Idempotency-Key"), body: JSON.parse(String(options?.body)) }); return ok(); }) as typeof fetch;
 const draft = { version: 3, path: "/blog/guide", kind: "blog" as const, position: 1, translations: { zh: { title: "指南", summary: "摘要", body: "正文", seo_title: "", seo_description: "" } }, source_locale: "zh" as const, reason: "编辑中文草稿" };
 await saveContent("admin", "article-id", draft, "draft-intent");
 await saveContent("admin", "article-id", draft, "draft-intent");
 await publishContent("admin", "article-id", 4, "zh", "publish", "发布中文", "publication-intent");
 assert.deepEqual(calls[0], calls[1]); assert.equal(calls[0].method, "PUT"); assert.equal(calls[0].key, "draft-intent"); assert.equal((calls[0].body as { source_locale?: string }).source_locale, "zh");
 assert.deepEqual(calls[2].body, { version: 4, locale: "zh", action: "publish", reason: "发布中文" }); assert.equal(calls[2].key, "publication-intent");
});
test("public server reads may cache but authenticated requests remain uncached", async () => {
 const modes: Array<RequestCache | undefined> = [];
 globalThis.fetch = (async (_url, options) => { modes.push(options?.cache); return ok(); }) as typeof fetch;
 await apiRequest("/site/settings", { cache: "force-cache", next: { revalidate: 30 } });
 await apiRequest("/admin/articles", { authToken: "admin", cache: "force-cache" });
 assert.deepEqual(modes, ["force-cache", "no-store"]);
});
