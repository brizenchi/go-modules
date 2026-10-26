import assert from "node:assert/strict";
import test from "node:test";
import { customerBreadcrumbs, customerFooterLinks, defaultNavigation, navigationSections, normalizeNavigation, publicBreadcrumbs, resolveFooter, resolveHeader, siteLegalLinks, siteLinks } from "../content/navigation";
import type { NavigationIndex } from "../lib/content-discovery";
import { publicContentPaths } from "../lib/seo";
import { breadcrumbStructuredData } from "../lib/structured-data";

const index: NavigationIndex = {
  entries: [
    { path: "/features/credits", kind: "features", title: "Credits", position: 2 },
    { path: "/features/authentication", kind: "features", title: "Login", position: 1 },
    { path: "/compare/starter", kind: "compare", title: "Starter or scratch", position: 1 },
    { path: "/docs/setup", kind: "docs", title: "Setup", position: 1 }
  ],
  directories: ["/features", "/compare", "/docs", "/resources", "/about"]
};

test("by default the header has Features and Use cases, each listing its published pages", () => {
  assert.deepEqual(defaultNavigation.header.map(item => [item.section, item.menu]), [["features", "auto"], ["use-cases", "auto"]]);
  const header = resolveHeader(defaultNavigation, { ...index, directories: [...index.directories, "/use-cases"], entries: [...index.entries, { path: "/use-cases/saas", kind: "use-cases", title: "SaaS", position: 1 }] }, "en", "/features/credits");
  assert.deepEqual(header.map(item => [item.href, item.label, item.current]), [["/features", "Features", true], ["/use-cases", "Use cases", false]]);
  assert.deepEqual(header[0].links.map(link => [link.href, link.current]), [["/features/authentication", false], ["/features/credits", true]]);
  assert.deepEqual(header[0].all, { href: "/features", label: "View all" });
  // Single level: a use-case menu lists use cases only, never a comparison sub-group.
  assert.deepEqual(header[1].links.map(link => link.href), ["/use-cases/saas"]);
});

test("saved menus control sections, names, order and chosen pages; empty sections disappear", () => {
  const navigation = normalizeNavigation({ version: 1, header: [
    { section: "docs", label: { en: "", zh: "帮助" }, menu: "none", items: [] },
    { section: "use-cases", label: { en: "Solutions", zh: "" }, menu: "manual", items: ["/compare/starter", "/docs/missing", "/features/credits"] },
    { section: "blog", label: { en: "", zh: "" }, menu: "auto", items: [] },
    { section: "pricing", label: { en: "", zh: "" }, menu: "none", items: [] }
  ], footer: [] });
  const header = resolveHeader(navigation, { ...index, directories: [...index.directories, "/use-cases"] }, "zh", "/zh/compare/starter");
  assert.deepEqual(header.map(item => [item.href, item.label]), [["/docs", "帮助"], ["/use-cases", "使用场景"], ["/pricing", "价格"]], "blog is hidden: no published articles");
  assert.deepEqual(header[0].links, [], "a menu can be turned off");
  assert.deepEqual(header[1].links.map(link => link.href), ["/compare/starter", "/features/credits"], "chosen order; unpublished pages are skipped");
  assert.equal(header[1].current, true);
  assert.equal(resolveHeader(navigation, { ...index, directories: [...index.directories, "/use-cases"] }, "en", "/")[1].label, "Solutions");
});

test("without the published index every section shows, without menus", () => {
  const header = resolveHeader(defaultNavigation, null, "en", "/");
  assert.deepEqual(header.map(item => [item.href, item.links.length]), [["/features", 0], ["/use-cases", 0]]);
});

test("saved menus are read defensively and nothing saved means the defaults", () => {
  for (const value of [null, undefined, {}, { version: 2 }, "menu"]) assert.equal(normalizeNavigation(value), defaultNavigation);
  const navigation = normalizeNavigation({ version: 1, header: [
    { section: "admin", menu: "none" },
    { section: "features", label: { en: " Product " }, menu: "manual", items: ["https://example.com", "/features/credits", "/features/credits"] },
    { section: "features", menu: "auto" },
    { section: "pricing", menu: "auto", items: ["/features/credits"] },
    { section: "docs", menu: "manual", items: [] }
  ], footer: [
    { title: { en: "Product", zh: "产品" }, links: ["features", "/account", "features"] },
    { title: { en: "", zh: "" }, links: ["pricing"] },
    { title: { en: "More" }, links: ["features"] }
  ] });
  assert.deepEqual(navigation.header, [
    { section: "features", label: { en: "Product", zh: "" }, menu: "manual", items: ["/features/credits"] },
    { section: "pricing", label: { en: "", zh: "" }, menu: "none", items: [] },
    { section: "docs", label: { en: "", zh: "" }, menu: "none", items: [] }
  ]);
  assert.deepEqual(navigation.footer, [{ title: { en: "Product", zh: "产品" }, links: ["features"] }]);
});

test("footer columns link published sections only and drop columns left empty", () => {
  assert.deepEqual(resolveFooter(defaultNavigation, null, "zh").map(group => group.title), ["产品", "资源", "关于与支持"]);
  const footer = resolveFooter(defaultNavigation, new Set(["/features"]), "en");
  assert.deepEqual(footer.map(group => [group.title, group.links.map(link => link.href)]), [["Product", ["/features", "/pricing"]], ["About & support", ["/contact"]]]);
  const links = defaultNavigation.footer.flatMap(group => group.links);
  assert.equal(new Set(links).size, links.length, "no footer link repeats");
  const published = new Set(publicContentPaths());
  for (const link of links) assert.ok(published.has(navigationSections[link].link.href), link);
  for (const link of siteLegalLinks) assert.ok(published.has(link.href), link.href);
});

test("customer and public footers share the same published help destination", () => {
  assert.ok(customerFooterLinks.includes(siteLinks.help));
  assert.equal(navigationSections.contact.link, siteLinks.help);
  assert.equal(siteLinks.help.href, "/contact");
  assert.equal(siteLinks.help.label.zh, "帮助与支持");
});

test("customer pages are peers and preserve homepage language", () => {
  assert.deepEqual(customerBreadcrumbs("/account", "zh").map(item => item.href), ["/zh", "/account"]);
  assert.deepEqual(customerBreadcrumbs("/billing", "en").map(item => item.label), ["Home", "Subscription & billing"]);
  assert.deepEqual(customerBreadcrumbs("/credits", "zh").map(item => item.label), ["首页", "积分明细"]);
  assert.equal(customerBreadcrumbs("/referrals", "zh").at(-1)?.label, "推荐奖励");
  for (const locale of ["en", "zh"] as const) {
    for (const path of ["/account", "/billing", "/referrals", "/credits"]) {
      const items = customerBreadcrumbs(path, locale);
      assert.equal(items.length, 2);
      assert.equal(items[0].href, locale === "zh" ? "/zh" : "/");
      assert.equal(items[1].href, path);
    }
  }
});

test("breadcrumbs stay two levels below home whatever the navigation shows", () => {
  for (const locale of ["en", "zh"] as const) {
    const prefix = locale === "zh" ? "/zh" : "";
    for (const directory of ["/docs", "/blog", "/updates"]) {
      const items = publicBreadcrumbs(`${prefix}${directory}`, locale);
      assert.deepEqual(items.map(item => item.href), [prefix || "/", `${prefix}${directory}`]);
      const schema = breadcrumbStructuredData(items, `${prefix}${directory}`);
      assert.equal(new URL(schema.itemListElement[1].item).pathname, `${prefix}${directory}`);
    }
    for (const path of ["/docs/setup", "/blog/custom-title"]) {
      const directory = `/${path.split("/")[1]}`;
      const supplied = [{ href: "/", label: "Home" }, { href: directory, label: "Category" }, { label: "Published article title" }];
      const items = publicBreadcrumbs(path, locale, supplied);
      assert.deepEqual(items.map(item => item.href), [prefix || "/", `${prefix}${directory}`, undefined]);
      assert.equal(items[2].label, "Published article title");
      assert.deepEqual(publicBreadcrumbs(path, locale, items), items);
    }
  }
  assert.deepEqual(publicBreadcrumbs("/resources", "zh").map(item => item.label), ["首页", "资源"]);
  for (const path of ["/contact", "/features", "/about", "/docs-other"]) assert.equal(publicBreadcrumbs(path, "en").length, 2);
});
