import assert from "node:assert/strict";
import test from "node:test";
import { activePublicNavigationHref, customerBreadcrumbs, customerFooterLinks, publicNavigation, siteLinks, siteFooterGroups, siteLegalLinks } from "../content/navigation";
import { publicContentPaths } from "../lib/seo";

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
