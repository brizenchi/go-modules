import assert from "node:assert/strict";
import { test } from "node:test";
import type { ContentArticle, ContentTranslation } from "../lib/api";
import { articleState, contentListQuery, listedLanguages, moveAvailability, noContentFilters, reorderable, withLanguage } from "../lib/content-filters";

const text = (title: string, body = "Body"): ContentTranslation => ({ title, summary: "", body, seo_title: "", seo_description: "" });
const row = (translations: ContentArticle["translations"], published: ContentArticle["published"] = {}) => ({ translations, published });

test("article language state matches the backend's published, changed, draft and missing", () => {
 const live = { ...text("Guide"), published_at: "2026-10-01T00:00:00Z" };
 assert.equal(articleState(row({ en: text("Guide") }, { en: live }), "en"), "published");
 assert.equal(articleState(row({ en: text("Guide", "Edited") }, { en: live }), "en"), "changed");
 assert.equal(articleState(row({}, { en: live }), "en"), "published");
 assert.equal(articleState(row({ ja: text("ガイド") }), "ja"), "draft");
 assert.equal(articleState(row({ ja: text(" ", " ") }), "ja"), "missing", "an empty draft is not a version");
 assert.equal(articleState(row({}), "de"), "missing");
});

test("the list shows the languages that are written or published", () => {
 assert.deepEqual(listedLanguages(row({ ja: text("ガイド"), ko: text("", "") }, { fr: { ...text("Guide"), published_at: "2026-10-01T00:00:00Z" } })), ["ja", "fr"]);
});

test("list filters send only chosen values, and missing needs a language", () => {
 assert.equal(contentListQuery(1, noContentFilters).toString(), "page=1&limit=20");
 assert.equal(contentListQuery(3, { query: "pay", kind: "docs", locale: "ja", status: "missing" }).toString(), "page=3&limit=20&query=pay&kind=docs&locale=ja&status=missing");
 assert.deepEqual(withLanguage({ ...noContentFilters, locale: "ja", status: "missing" }, ""), { ...noContentFilters, locale: "", status: "" });
 assert.deepEqual(withLanguage({ ...noContentFilters, locale: "ja", status: "draft" }, ""), { ...noContentFilters, locale: "", status: "draft" });
 assert.deepEqual(withLanguage({ ...noContentFilters, status: "changed" }, "de"), { ...noContentFilters, locale: "de", status: "changed" });
});

test("rows move within their section, and only when no search, language or state filter hides neighbours", () => {
 assert.equal(reorderable(noContentFilters), true);
 assert.equal(reorderable({ ...noContentFilters, kind: "features" }), true);
 for (const hidden of [{ query: "x" }, { locale: "ja" as const }, { status: "draft" as const }]) assert.equal(reorderable({ ...noContentFilters, ...hidden }), false);
 const rows = [{ kind: "features" as const }, { kind: "features" as const }, { kind: "use-cases" as const }, { kind: "page" as const }];
 assert.deepEqual(moveAvailability(rows, 0, 1, 4, 20), { up: false, down: true }, "first row of the first page");
 assert.deepEqual(moveAvailability(rows, 1, 1, 4, 20), { up: true, down: false }, "last of its section");
 assert.deepEqual(moveAvailability(rows, 2, 1, 4, 20), { up: false, down: false }, "alone in its section");
 assert.deepEqual(moveAvailability(rows, 3, 1, 4, 20), { up: false, down: false }, "pages have fixed places");
 assert.deepEqual(moveAvailability(rows.slice(0, 2), 0, 2, 40, 20), { up: true, down: true }, "the neighbour may be on the previous page");
});
