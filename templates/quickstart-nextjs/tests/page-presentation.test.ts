import assert from "node:assert/strict";
import { test } from "node:test";
import { pagePresentation } from "../lib/page-presentation";

test("missing homepage uses saved site identity without inventing an article", () => {
 const site = { brand_name: "Saved brand", headline_en: "Saved heading", description: "Saved introduction", support_email: "", support_url: "", export_credit_cost: 1 };
 const page = pagePresentation("/", "en", site, "Default brand");
 assert.equal(page.translation.title, "Saved heading");
 assert.equal(page.translation.summary, "Saved introduction");
 assert.equal(page.translation.body, "");
 assert.equal("published_at" in page, false);
 assert.equal("id" in page, false);
});

test("missing translations and directory introductions never reveal English content or seeded copy", () => {
 const site = { brand_name: "Saved brand", headline_en: "English only", description: "English only", support_email: "", support_url: "", export_credit_cost: 1 };
 const page = pagePresentation("/", "zh", site, "Default brand");
 assert.equal(page.translation.title, "Saved brand");
 assert.equal(page.translation.summary, "");
 const directory = pagePresentation("/resources", "zh", site, "Default brand");
 assert.equal(directory.translation.title, "指南");
 assert.equal(directory.translation.summary, "");
 assert.equal(directory.translation.body, "");
});
