import assert from "node:assert/strict";
import test from "node:test";
import { defaultNavigation } from "../content/navigation";
import type { PublishedArticle, SiteNavigation } from "../lib/api";
import { addHeaderItem, articleOptions, moveItem, navigationPayload, navigationProblems, removeItem, unusedFooterSections, unusedHeaderSections } from "../lib/navigation-editor";

const published = (path: string, kind: PublishedArticle["kind"], title: string) => ({ id: path, path, kind, title, translation: { title, summary: "", body: "", seo_title: "", seo_description: "" } }) as unknown as PublishedArticle;

test("menu choices are published articles from either language, listed once", () => {
  const options = articleOptions([
    published("/features/credits", "features", "Credits"), published("/", "page", "Home"),
    published("/features/credits", "features", "积分"), published("/docs/setup", "docs", "配置")
  ]);
  assert.deepEqual(options, [{ path: "/docs/setup", kind: "docs", title: "配置" }, { path: "/features/credits", kind: "features", title: "Credits" }]);
});

test("items move within bounds and can be removed", () => {
  assert.deepEqual(moveItem(["a", "b", "c"], 0, 1), ["b", "a", "c"]);
  assert.deepEqual(moveItem(["a", "b", "c"], 2, 1), ["a", "b", "c"]);
  assert.deepEqual(moveItem(["a", "b"], 0, -1), ["a", "b"]);
  assert.deepEqual(removeItem(["a", "b", "c"], 1), ["a", "c"]);
});

test("adding a section picks a menu it supports and never repeats a section", () => {
  const start: SiteNavigation = { ...defaultNavigation, header: defaultNavigation.header.slice(0, 2) };
  let navigation = addHeaderItem(start, "about");
  assert.deepEqual(navigation.header.at(-1), { section: "about", label: { en: "", zh: "" }, menu: "none", items: [] });
  navigation = addHeaderItem(navigation, "resources");
  assert.equal(navigation.header.at(-1)?.menu, "auto");
  assert.equal(addHeaderItem(navigation, "about"), navigation);
  assert.equal(addHeaderItem(defaultNavigation, "docs").header.length, 6);
  assert.ok(!unusedHeaderSections(navigation).includes("features"));
  assert.ok(!unusedFooterSections(defaultNavigation).includes("pricing"));
  assert.ok(unusedFooterSections(defaultNavigation).includes("docs"));
});

test("the editor reports what the backend would reject", () => {
  assert.deepEqual(navigationProblems(defaultNavigation), []);
  const broken: SiteNavigation = { version: 1,
    header: [{ section: "features", label: { en: "x".repeat(41), zh: "" }, menu: "manual", items: [] }, { section: "pricing", label: { en: "", zh: "" }, menu: "auto", items: [] }],
    footer: [{ title: { en: "Product", zh: "" }, links: [] }, { title: { en: "More", zh: "更多" }, links: ["features", "features"] }]
  };
  assert.deepEqual(navigationProblems(broken).sort(), ["footer-links", "footer-title", "label", "menu"]);
});

test("the saved payload is trimmed and keeps chosen pages only for chosen menus", () => {
  const payload = navigationPayload({ version: 1,
    header: [{ section: "features", label: { en: " Product ", zh: "" }, menu: "auto", items: ["/features/x"] }],
    footer: [{ title: { en: " Product ", zh: " 产品 " }, links: ["features"] }]
  });
  assert.deepEqual(payload, { version: 1, header: [{ section: "features", label: { en: "Product", zh: "" }, menu: "auto", items: [] }], footer: [{ title: { en: "Product", zh: "产品" }, links: ["features"] }] });
});
