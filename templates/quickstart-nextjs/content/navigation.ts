import { humanizeSegment, localizeHref, stripLocalePrefix, type Locale } from "../lib/locale";
import { isContentDirectory, type NavigationEntry } from "../lib/content-discovery";
import { siteConfig } from "./site";

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
  examples: link("/examples", "Live demo", "在线体验"),
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

export type HeaderMenuGroup = { title: Copy; kind?: NavigationEntry["kind"]; links?: readonly SiteLink[]; all?: SiteLink };
/** `fallback` takes the item's place when its own directory is empty (Comparisons without use cases). */
export type HeaderItem = { link: SiteLink; label?: Copy; fallback?: SiteLink; groups?: readonly HeaderMenuGroup[] };
export type ResolvedMenuGroup = { title: string; links: Array<{ href: string; label: string }>; all?: { href: string; label: string } };

/** Header: each directory opens a menu of its published pages; the logo is the home link. */
export const headerNavigation: readonly HeaderItem[] = [
  { link: siteLinks.features, groups: [
    { title: { en: "Capabilities", zh: "核心能力" }, kind: "features", all: { ...siteLinks.features, label: { en: "All features", zh: "全部功能" } } }
  ] },
  { link: siteLinks.useCases, fallback: siteLinks.compare, groups: [
    { title: { en: "By business model", zh: "按收费方式" }, kind: "use-cases", all: { ...siteLinks.useCases, label: { en: "All use cases", zh: "全部场景" } } },
    { title: { en: "Comparisons", zh: "方案比较" }, kind: "compare", all: { ...siteLinks.compare, label: { en: "All comparisons", zh: "全部比较" } } }
  ] },
  { link: siteLinks.docs, label: { en: "Docs", zh: "文档" } },
  { link: siteLinks.pricing },
  { link: siteLinks.resources, groups: [
    { title: { en: "Learn and follow", zh: "学习与更新" }, links: [siteLinks.blog, siteLinks.updates], all: { ...siteLinks.resources, label: { en: "All guides", zh: "指南总览" } } }
  ] }
];
export const publicNavigation = headerNavigation.map(item => item.link);
/** The header call to action; the account menu beside it owns sign-in. */
export const headerAction = siteLinks.examples;
export const sourceRepository = siteConfig.repositoryURL ? { href: siteConfig.repositoryURL, label: { en: "GitHub", zh: "GitHub" } } : null;
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
/** Footer links reach directories only; directories and the header link each page. */
export const siteFooterGroups: Array<{ title: Copy; links: SiteLink[] }> = [
  { title: { en: "Product", zh: "产品" }, links: [siteLinks.features, siteLinks.useCases, siteLinks.compare, siteLinks.examples, siteLinks.pricing] },
  { title: { en: "Guides", zh: "指南" }, links: [...guideDirectories] },
  { title: { en: "About & support", zh: "关于与支持" }, links: [siteLinks.about, siteLinks.help] }
];

/** Shows every link until the published directories are known (`null`). */
export function directoryFilter(directories: ReadonlySet<string> | null): (href: string) => boolean {
  return href => !directories || !isContentDirectory(href) || directories.has(href);
}

/** Header items whose directory has something to show; empty sections disappear. */
export function visibleHeaderNavigation(directories: ReadonlySet<string> | null): Array<HeaderItem & { section: string }> {
  const shown = directoryFilter(directories);
  return headerNavigation.flatMap(item => {
    const link = shown(item.link.href) ? item.link : item.fallback && shown(item.fallback.href) ? item.fallback : null;
    if (!link) return [];
    return [{
      ...item, link, section: item.link.href, label: link === item.link ? item.label : undefined,
      groups: item.groups?.map(group => ({ ...group, links: group.links?.filter(entry => shown(entry.href)), all: group.all && shown(group.all.href) ? group.all : undefined }))
    }];
  });
}

export function visibleFooterGroups(directories: ReadonlySet<string> | null): Array<{ title: Copy; links: SiteLink[] }> {
  const shown = directoryFilter(directories);
  return siteFooterGroups.map(group => ({ ...group, links: group.links.filter(entry => shown(entry.href)) })).filter(group => group.links.length > 0);
}

/** Menus list only pages published in this language, so the header never links to a 404. */
export function headerMenu(item: HeaderItem, entries: readonly NavigationEntry[], locale: Locale, limit = 8): ResolvedMenuGroup[] {
  return (item.groups || []).map(group => {
    const links = group.kind
      ? entries.filter(entry => entry.kind === group.kind)
        .sort((a, b) => (a.position ?? 0) - (b.position ?? 0) || a.path.localeCompare(b.path))
        .slice(0, limit)
        .map(entry => ({ href: entry.path, label: entry.title }))
      : (group.links || []).map(link => ({ href: link.href, label: link.label[locale] }));
    return { title: group.title[locale], links, ...(group.all ? { all: { href: group.all.href, label: group.all.label[locale] } } : {}) };
  }).filter(group => group.links.length > 0);
}

export function activePublicNavigationHref(path: string): string {
  const pathname = stripLocalePrefix(path.split(/[?#]/, 1)[0]);
  const within = (prefix: string) => pathname === prefix || pathname.startsWith(`${prefix}/`);
  if (within("/features")) return "/features";
  if (within("/use-cases") || within("/compare")) return "/use-cases";
  if (within("/docs")) return "/docs";
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
