import assert from "node:assert/strict";
import test from "node:test";
import { activePublicNavigationHref, customerBreadcrumbs, customerFooterLinks, guideDirectories, guideNavigation, publicBreadcrumbs, publicNavigation, siteLinks, siteFooterGroups, siteLegalLinks } from "../content/navigation";
import { publicContentPaths } from "../lib/seo";
import { breadcrumbStructuredData } from "../lib/structured-data";

test("public navigation keeps one feature entry and an explicit home destination", () => {
  assert.deepEqual(publicNavigation.map(item => item.href), ["/", "/features", "/pricing", "/resources"]);
  assert.equal(activePublicNavigationHref("/"), "/");
  assert.equal(activePublicNavigationHref("/features/credits"), "/features");
  assert.equal(activePublicNavigationHref("/use-cases/subscription-saas"), "");
  assert.equal(activePublicNavigationHref("/docs/payments"), "/resources");
  assert.equal(activePublicNavigationHref("/blog/a-guide"), "/resources");
  assert.equal(activePublicNavigationHref("/compare/starter-vs-from-scratch"), "");
  assert.equal(activePublicNavigationHref("/contact"), "");
  assert.equal(activePublicNavigationHref("/about"), "");
  for (const path of ["/account", "/admin", "/features-other", "/unknown"]) assert.equal(activePublicNavigationHref(path), "");
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
      assert.equal(activePublicNavigationHref(`${prefix}${directory.href}`), "/resources");
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
