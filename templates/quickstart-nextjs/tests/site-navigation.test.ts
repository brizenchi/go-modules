import assert from "node:assert/strict";
import test from "node:test";
import { activePublicNavigationHref, customerBreadcrumbs, customerFooterLinks, publicNavigation, siteLinks, siteFooterGroups } from "../content/navigation";
import { publicContentPaths } from "../lib/seo";

test("public navigation keeps one feature entry and an explicit home destination", () => {
  assert.deepEqual(publicNavigation.map(item => item.href), ["/", "/features", "/pricing", "/resources"]);
  assert.equal(activePublicNavigationHref("/"), "/");
  assert.equal(activePublicNavigationHref("/features/credits"), "/features");
  assert.equal(activePublicNavigationHref("/use-cases/subscription-saas"), "/features");
  assert.equal(activePublicNavigationHref("/docs/payments"), "/resources");
  assert.equal(activePublicNavigationHref("/blog/a-guide"), "/resources");
  for (const path of ["/account", "/admin", "/features-other", "/unknown"]) assert.equal(activePublicNavigationHref(path), "");
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

test("customer breadcrumbs follow the account hierarchy and preserve homepage language", () => {
  assert.deepEqual(customerBreadcrumbs("/account", "zh").map(item => item.href), ["/zh", "/account"]);
  assert.deepEqual(customerBreadcrumbs("/billing", "en").map(item => item.label), ["Home", "My account", "Subscription & billing"]);
  assert.deepEqual(customerBreadcrumbs("/credits", "zh").map(item => item.label), ["首页", "账户中心", "积分明细"]);
  assert.equal(customerBreadcrumbs("/referrals", "zh").at(-1)?.label, "推荐奖励");
});
