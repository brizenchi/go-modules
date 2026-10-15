import { localizeHref, stripLocalePrefix, type Locale } from "../lib/locale";

type Copy = { en: string; zh: string };
export type SiteLink = { href: string; label: Copy };
const link = (href: string, en: string, zh: string): SiteLink => ({ href, label: { en, zh } });

/** Shared names for the header, footer, account navigation and breadcrumbs. */
export const siteLinks = {
  home: link("/", "Home", "首页"),
  features: link("/features", "Features", "功能介绍"),
  useCases: link("/use-cases", "Use cases", "使用场景"),
  pricing: link("/pricing", "Pricing", "价格"),
  resources: link("/resources", "Guides", "指南"),
  docs: link("/docs", "Documentation", "使用文档"),
  blog: link("/blog", "Articles", "产品文章"),
  updates: link("/updates", "Updates", "更新日志"),
  examples: link("/examples", "Try the demo", "体验演示"),
  compare: link("/compare", "Comparisons", "方案比较"),
  about: link("/about", "About", "关于我们"),
  help: link("/contact", "Help & support", "帮助与支持"),
  privacy: link("/privacy", "Privacy policy", "隐私政策"),
  terms: link("/terms", "Terms of service", "服务条款"),
  account: link("/account", "My account", "账户中心"),
  billing: link("/billing", "Subscription & billing", "订阅与账单"),
  referrals: link("/referrals", "Referral rewards", "推荐奖励"),
  credits: link("/credits", "Credit history", "积分明细")
};

export const publicNavigation = [siteLinks.home, siteLinks.features, siteLinks.pricing, siteLinks.resources];
export const customerFooterLinks = [siteLinks.help, siteLinks.privacy, siteLinks.terms];
export const siteFooterGroups: Array<{ title: Copy; links: SiteLink[] }> = [
  { title: { en: "Product", zh: "产品" }, links: [siteLinks.features, siteLinks.useCases, siteLinks.pricing, siteLinks.examples] },
  { title: siteLinks.resources.label, links: [siteLinks.resources, siteLinks.docs, siteLinks.blog, siteLinks.updates, siteLinks.compare] },
  { title: { en: "Support & legal", zh: "支持与条款" }, links: [siteLinks.help, siteLinks.about, siteLinks.privacy, siteLinks.terms] }
];

export function activePublicNavigationHref(path: string): string {
  const pathname = stripLocalePrefix(path.split(/[?#]/, 1)[0]);
  if (pathname === "/") return "/";
  const within = (prefix: string) => pathname === prefix || pathname.startsWith(`${prefix}/`);
  if (["/features", "/use-cases"].some(within)) return "/features";
  if (within("/pricing")) return "/pricing";
  if (["/resources", "/docs", "/blog", "/updates", "/examples", "/compare"].some(within)) return "/resources";
  return "";
}

export function siteLinkForPath(path: string): SiteLink | undefined {
  if (!path) return undefined;
  return Object.values(siteLinks).find(item => item.href === stripLocalePrefix(path.split(/[?#]/, 1)[0]));
}

export function customerBreadcrumbs(path: string, locale: Locale): Array<{ href: string; label: string }> {
  const current = [siteLinks.account, siteLinks.billing, siteLinks.referrals, siteLinks.credits].find(item => item.href === path.split(/[?#]/, 1)[0]) || siteLinks.account;
  const items = current === siteLinks.account ? [siteLinks.home, current] : [siteLinks.home, siteLinks.account, current];
  return items.map(item => ({ href: localizeHref(item.href, locale), label: item.label[locale] }));
}
