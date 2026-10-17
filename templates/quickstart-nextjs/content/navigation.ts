import { humanizeSegment, localizeHref, stripLocalePrefix, type Locale } from "../lib/locale";

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
export const productNavigation = [siteLinks.features, siteLinks.useCases, siteLinks.compare, siteLinks.examples];
export const guideDirectories = [
  { ...siteLinks.docs, description: { en: "Set up your site and connect the services you need.", zh: "配置站点，接入产品需要的服务。" } },
  { ...siteLinks.blog, description: { en: "Practical answers to product and account questions.", zh: "围绕产品与账户问题，查阅实用文章。" } },
  { ...siteLinks.updates, description: { en: "See what changed and where to use it.", zh: "了解最近改了什么，以及在哪里使用。" } }
];
export const guideNavigation: readonly SiteLink[] = [
  { ...siteLinks.resources, label: { en: "All guides", zh: "指南总览" } },
  ...guideDirectories
];
export const customerFooterLinks = [siteLinks.help, siteLinks.privacy, siteLinks.terms];
export const siteLegalLinks = [siteLinks.privacy, siteLinks.terms];
export const siteFooterGroups: Array<{ title: Copy; links: SiteLink[] }> = [
  { title: { en: "Product", zh: "产品" }, links: [siteLinks.features, siteLinks.useCases, siteLinks.pricing, siteLinks.compare, siteLinks.examples] },
  { title: { en: "Guides", zh: "指南" }, links: [...guideNavigation] },
  { title: { en: "About & support", zh: "关于与支持" }, links: [siteLinks.about, siteLinks.help] }
];

export function activePublicNavigationHref(path: string): string {
  const pathname = stripLocalePrefix(path.split(/[?#]/, 1)[0]);
  if (pathname === "/") return "/";
  const within = (prefix: string) => pathname === prefix || pathname.startsWith(`${prefix}/`);
  if (within("/features")) return "/features";
  if (within("/pricing")) return "/pricing";
  if (guideNavigation.some(item => within(item.href))) return siteLinks.resources.href;
  return "";
}

export function siteLinkForPath(path: string): SiteLink | undefined {
  if (!path) return undefined;
  return Object.values(siteLinks).find(item => item.href === stripLocalePrefix(path.split(/[?#]/, 1)[0]));
}

export type PublicBreadcrumb = { href?: string; label: string };

/** The guide hub is the editorial parent, independent of the stable route URLs. */
export function publicBreadcrumbs(path: string, locale: Locale, supplied?: readonly PublicBreadcrumb[]): PublicBreadcrumb[] {
  const pathname = stripLocalePrefix(path.split(/[?#]/, 1)[0]).replace(/\/+$/, "") || "/";
  let current = "";
  const defaults: PublicBreadcrumb[] = pathname === "/" ? [{ label: siteLinks.home.label[locale] }] : [
    { href: "/", label: siteLinks.home.label[locale] },
    ...pathname.split("/").filter(Boolean).map(segment => {
      current += `/${segment}`;
      return { href: current, label: humanizeSegment(segment) };
    })
  ];
  const items = (supplied?.length ? supplied : defaults).map((item, index, source) => {
    const known = siteLinkForPath(item.href || (index === source.length - 1 ? pathname : ""));
    return { label: known?.label[locale] || item.label, href: item.href ? localizeHref(item.href, locale) : undefined };
  });
  if (!guideDirectories.some(item => pathname === item.href || pathname.startsWith(`${item.href}/`))) return items;
  const trail = items.filter(item => !item.href || !["/", siteLinks.resources.href].includes(stripLocalePrefix(item.href)));
  return [
    { href: localizeHref("/", locale), label: siteLinks.home.label[locale] },
    { href: localizeHref(siteLinks.resources.href, locale), label: siteLinks.resources.label[locale] },
    ...trail
  ];
}

export function customerBreadcrumbs(path: string, locale: Locale): Array<{ href: string; label: string }> {
  const current = [siteLinks.account, siteLinks.billing, siteLinks.referrals, siteLinks.credits].find(item => item.href === path.split(/[?#]/, 1)[0]) || siteLinks.account;
  const items = [siteLinks.home, current];
  return items.map(item => ({ href: localizeHref(item.href, locale), label: item.label[locale] }));
}
