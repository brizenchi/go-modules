import assert from "node:assert/strict";
import test from "node:test";
import { activePublicNavigationHref, customerBreadcrumbs, customerFooterLinks, guideDirectories, guideNavigation, headerAction, headerMenu, headerNavigation, publicBreadcrumbs, publicNavigation, siteLinks, siteFooterGroups, siteLegalLinks, visibleFooterGroups, visibleHeaderNavigation } from "../content/navigation";
import type { NavigationEntry } from "../lib/content-discovery";
import { publicContentPaths } from "../lib/seo";
import { breadcrumbStructuredData } from "../lib/structured-data";

test("header lists each public directory once; the logo is the home link", () => {
  assert.deepEqual(publicNavigation.map(item => item.href), ["/features", "/use-cases", "/docs", "/pricing", "/resources"]);
  assert.equal(activePublicNavigationHref("/"), "");
  assert.equal(activePublicNavigationHref("/features/credits"), "/features");
  assert.equal(activePublicNavigationHref("/use-cases/subscription-saas"), "/use-cases");
  assert.equal(activePublicNavigationHref("/zh/compare/starter-vs-from-scratch"), "/use-cases");
  assert.equal(activePublicNavigationHref("/docs/payments"), "/docs");
  assert.equal(activePublicNavigationHref("/blog/a-guide"), "/resources");
  assert.equal(activePublicNavigationHref("/updates"), "/resources");
  assert.equal(activePublicNavigationHref("/contact"), "");
  assert.equal(activePublicNavigationHref("/about"), "");
  for (const path of ["/account", "/admin", "/features-other", "/unknown"]) assert.equal(activePublicNavigationHref(path), "");
  assert.equal(headerAction.href, "/examples");
  assert.ok(new Set(publicContentPaths()).has(headerAction.href));
});

test("header menus link only published pages of their kind, in editorial order", () => {
  const entries: NavigationEntry[] = [
    { path: "/features/credits", kind: "features", title: "Credits", position: 2 },
    { path: "/features/authentication", kind: "features", title: "Login", position: 1 },
    { path: "/compare/starter-vs-from-scratch", kind: "compare", title: "Starter or scratch", position: 9 }
  ];
  const [features, useCases, docs, , guides] = headerNavigation;
  assert.deepEqual(headerMenu(features, entries, "en"), [{ title: "Capabilities", links: [{ href: "/features/authentication", label: "Login" }, { href: "/features/credits", label: "Credits" }], all: { href: "/features", label: "All features" } }]);
  // A directory without published pages keeps its own link but no empty group.
  assert.deepEqual(headerMenu(useCases, entries, "zh").map(group => group.title), ["方案比较"]);
  assert.deepEqual(headerMenu(useCases, [], "en"), []);
  assert.deepEqual(headerMenu(docs, entries, "en"), []);
  assert.deepEqual(headerMenu(guides, [], "zh")[0].links.map(link => link.href), ["/blog", "/updates"]);
  const many = Array.from({ length: 12 }, (_, index): NavigationEntry => ({ path: `/features/f-${index}`, kind: "features", title: `F${index}`, position: index }));
  assert.equal(headerMenu(features, many, "en")[0].links.length, 8);
});

test("footer secondary destinations remain published without duplicate links", () => {
  const published = new Set(publicContentPaths());
  for (const item of siteFooterGroups.flatMap(group => group.links)) assert.ok(published.has(item.href), item.href);
  const links = [...siteFooterGroups.flatMap(group => group.links), ...siteLegalLinks];
  assert.equal(new Set(links.map(item => item.href)).size, links.length);
  for (const group of siteFooterGroups) {
    assert.ok(group.links.every(item => item.label.en !== group.title.en && item.label.zh !== group.title.zh));
  }
});

test("customer and public footers share the same published help destination", () => {
  assert.ok(customerFooterLinks.includes(siteLinks.help));
  assert.ok(siteFooterGroups.some(group => group.links.includes(siteLinks.help)));
  assert.equal(siteLinks.help.href, "/contact");
  assert.equal(siteLinks.help.label.zh, "帮助与支持");
  const published = new Set(publicContentPaths());
  for (const link of [...publicNavigation, ...customerFooterLinks, ...siteFooterGroups.flatMap(group => group.links)]) {
    assert.ok(published.has(link.href), link.href);
  }
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

test("Documentation, Articles and Updates share Guides as their editorial parent", () => {
  assert.deepEqual(guideNavigation.map(item => item.href), ["/resources", "/docs", "/blog", "/updates"]);
  for (const locale of ["en", "zh"] as const) {
    const prefix = locale === "zh" ? "/zh" : "";
    for (const directory of guideDirectories) {
      const items = publicBreadcrumbs(`${prefix}${directory.href}`, locale);
      assert.deepEqual(items.map(item => item.href), [prefix || "/", `${prefix}/resources`, `${prefix}${directory.href}`]);
      assert.deepEqual(items.map(item => item.label), [siteLinks.home.label[locale], siteLinks.resources.label[locale], directory.label[locale]]);
      assert.equal(activePublicNavigationHref(`${prefix}${directory.href}`), directory.href === "/docs" ? "/docs" : "/resources");
      const schema = breadcrumbStructuredData(items, `${prefix}${directory.href}`);
      assert.equal(new URL(schema.itemListElement[1].item).pathname, `${prefix}/resources`);
    }
  }
});

test("guide article breadcrumbs preserve the published title without duplicating the parent", () => {
  for (const locale of ["en", "zh"] as const) {
    for (const path of ["/docs/setup", "/blog/custom-title"]) {
      const category = path.split("/")[1];
      const supplied = [{ href: "/", label: "Home" }, { href: "/resources", label: "Guides" }, { href: `/${category}`, label: "Category" }, { label: "Published article title" }];
      const items = publicBreadcrumbs(path, locale, supplied);
      assert.equal(items.length, 4);
      assert.equal(items[1].label, siteLinks.resources.label[locale]);
      assert.equal(items[3].label, "Published article title");
      assert.equal(items[3].href, undefined);
      assert.deepEqual(publicBreadcrumbs(path, locale, items), items);
    }
  }
  assert.equal(publicBreadcrumbs("/resources", "en").length, 2);
  for (const path of ["/contact", "/features", "/about", "/docs-other"]) assert.equal(publicBreadcrumbs(path, "en").length, 2);
});

test("footer links only directories, which in turn list every page", () => {
  const links = siteFooterGroups.flatMap(group => group.links);
  assert.ok(links.length <= 12, `${links.length} footer links`);
  for (const link of links) assert.match(link.href, /^\/[a-z-]+$/);
});

test("empty directories leave the header and footer; comparisons stand in for use cases", () => {
  assert.equal(visibleHeaderNavigation(null).length, headerNavigation.length);
  const header = visibleHeaderNavigation(new Set(["/docs", "/compare", "/blog"]));
  assert.deepEqual(header.map(item => item.link.href), ["/compare", "/docs", "/pricing"]);
  assert.equal(header[0].section, "/use-cases");
  assert.equal(header[0].label, undefined);
  assert.equal(header[0].groups?.[0].all, undefined, "no link to the empty use-case directory");
  const guides = visibleHeaderNavigation(new Set(["/resources", "/blog"])).find(item => item.link.href === "/resources");
  assert.deepEqual(guides?.groups?.[0].links?.map(link => link.href), ["/blog"]);
  const footer = visibleFooterGroups(new Set());
  assert.deepEqual(footer.map(group => group.links.map(link => link.href)), [["/examples", "/pricing"], ["/contact"]]);
  assert.equal(visibleFooterGroups(null).flatMap(group => group.links).length, siteFooterGroups.flatMap(group => group.links).length);
});
