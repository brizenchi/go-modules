import { humanizeSegment, localizeHref, stripLocalePrefix, type Locale } from "../lib/locale";
import { directoryKinds, isContentDirectory, isEditorialPath, type NavigationIndex } from "../lib/content-discovery";
import type { FooterNavigationGroup, HeaderNavigationItem, NavigationSection, SiteNavigation } from "../lib/api";

type Copy = { en: string; zh: string };
export type SiteLink = { href: string; label: Copy };
const link = (href: string, en: string, zh: string): SiteLink => ({ href, label: { en, zh } });

/** Shared names for the header, footer, account navigation and breadcrumbs. */
export const siteLinks = {
  home: link("/", "Home", "首页"),
  features: link("/features", "Features", "功能介绍"),
  useCases: link("/use-cases", "Use cases", "使用场景"),
  pricing: link("/pricing", "Pricing", "价格"),
  resources: link("/resources", "Resources", "资源"),
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

/** Each section a header item or footer link may name, and whether its directory lists articles. */
export const navigationSections: Readonly<Record<NavigationSection, { link: SiteLink; menu: boolean; headerLabel?: Copy }>> = {
  home: { link: siteLinks.home, menu: false },
  features: { link: siteLinks.features, menu: true },
  "use-cases": { link: siteLinks.useCases, menu: true },
  compare: { link: siteLinks.compare, menu: true },
  docs: { link: siteLinks.docs, menu: true, headerLabel: { en: "Docs", zh: "文档" } },
  blog: { link: siteLinks.blog, menu: true },
  resources: { link: siteLinks.resources, menu: true },
  updates: { link: siteLinks.updates, menu: false },
  pricing: { link: siteLinks.pricing, menu: false },
  examples: { link: siteLinks.examples, menu: false },
  about: { link: siteLinks.about, menu: false },
  contact: { link: siteLinks.help, menu: false }
};

const noLabel: Copy = { en: "", zh: "" };
/** Used until an administrator saves menus: one level of links, no dropdowns. */
export const defaultNavigation: SiteNavigation = {
  version: 1,
  header: (["home", "features", "use-cases", "compare", "pricing"] as const).map(section => ({ section, label: noLabel, menu: "none" as const, items: [] })),
  footer: [
    { title: { en: "Product", zh: "产品" }, links: ["features", "use-cases", "compare", "examples", "pricing"] },
    { title: { en: "About & support", zh: "关于与支持" }, links: ["about", "contact"] }
  ]
};

const isSection = (value: unknown): value is NavigationSection => typeof value === "string" && Object.hasOwn(navigationSections, value);
const record = (value: unknown): Record<string, unknown> => value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
const copyOf = (value: unknown): Copy => {
  const copy = record(value);
  return { en: typeof copy.en === "string" ? copy.en.trim() : "", zh: typeof copy.zh === "string" ? copy.zh.trim() : "" };
};

/** Saved menus are read defensively: unknown entries are dropped, and nothing saved means the defaults. */
export function normalizeNavigation(value: unknown): SiteNavigation {
  const raw = record(value);
  if (raw.version !== 1) return defaultNavigation;
  const sections = new Set<string>();
  const header: HeaderNavigationItem[] = [];
  for (const item of Array.isArray(raw.header) ? raw.header : []) {
    const entry = record(item);
    if (!isSection(entry.section) || sections.has(entry.section)) continue;
    sections.add(entry.section);
    const menuCapable = navigationSections[entry.section].menu;
    const items = [...new Set((Array.isArray(entry.items) ? entry.items : []).filter((path): path is string => typeof path === "string" && isEditorialPath(path)))];
    const menu = menuCapable && entry.menu === "manual" && items.length ? "manual" : menuCapable && entry.menu === "auto" ? "auto" : "none";
    header.push({ section: entry.section, label: copyOf(entry.label), menu, items: menu === "manual" ? items : [] });
  }
  const linked = new Set<string>();
  const footer: FooterNavigationGroup[] = [];
  for (const group of Array.isArray(raw.footer) ? raw.footer : []) {
    const entry = record(group);
    const title = copyOf(entry.title);
    const links: NavigationSection[] = [];
    for (const link of Array.isArray(entry.links) ? entry.links : []) {
      if (isSection(link) && !linked.has(link)) { linked.add(link); links.push(link); }
    }
    if (links.length && (title.en || title.zh)) footer.push({ title, links });
  }
  return { version: 1, header, footer };
}

export type ResolvedLink = { href: string; label: string; current: boolean };
export type ResolvedHeaderItem = ResolvedLink & { links: ResolvedLink[]; all?: { href: string; label: string } };

/** Shows every link until the published directories are known (`null`). */
export function directoryFilter(directories: ReadonlySet<string> | null): (href: string) => boolean {
  return href => !directories || !isContentDirectory(href) || directories.has(href);
}

const within = (pathname: string, href: string) => pathname === href || pathname.startsWith(`${href}/`);

/** The header in one language: empty sections disappear and menus link only pages published in it.
 * Without the published index (`null`) every section shows, without menus. */
export function resolveHeader(navigation: SiteNavigation, index: NavigationIndex | null, locale: Locale, path: string, limit = 10): ResolvedHeaderItem[] {
  const pathname = stripLocalePrefix(path.split(/[?#]/, 1)[0]);
  const shown = directoryFilter(index ? new Set(index.directories) : null);
  const entries = index?.entries ?? [];
  const titles = new Map(entries.map(entry => [entry.path, entry.title]));
  return navigation.header.flatMap(item => {
    const section = navigationSections[item.section];
    const href = section.link.href;
    if (!shown(href)) return [];
    const kinds = directoryKinds(href);
    const pages = item.menu === "manual"
      ? item.items.flatMap(page => titles.has(page) ? [{ path: page, title: titles.get(page)! }] : [])
      : item.menu === "auto"
        ? entries.filter(entry => kinds.includes(entry.kind)).sort((a, b) => (a.position ?? 0) - (b.position ?? 0) || a.path.localeCompare(b.path))
        : [];
    const links = pages.slice(0, limit).map(page => ({ href: page.path, label: page.title, current: pathname === page.path }));
    return [{
      href, label: item.label[locale] || (section.headerLabel || section.link.label)[locale],
      current: within(pathname, href) || links.some(link => link.current), links,
      ...(links.length ? { all: { href, label: locale === "zh" ? "查看全部" : "View all" } } : {})
    }];
  });
}

/** Footer columns in one language; links to empty sections and then empty columns are left out. */
export function resolveFooter(navigation: SiteNavigation, directories: ReadonlySet<string> | null, locale: Locale): Array<{ title: string; links: Array<{ href: string; label: string }> }> {
  const shown = directoryFilter(directories);
  return navigation.footer.map(group => ({
    title: group.title[locale] || group.title.en || group.title.zh,
    links: group.links.map(key => navigationSections[key].link).filter(link => shown(link.href)).map(link => ({ href: link.href, label: link.label[locale] }))
  })).filter(group => group.links.length > 0);
}

export const productNavigation = [siteLinks.features, siteLinks.useCases, siteLinks.compare];
export const guideDirectories = [
  { ...siteLinks.docs, description: { en: "Set up your site and connect the services you need.", zh: "配置站点，接入产品需要的服务。" } },
  { ...siteLinks.blog, description: { en: "Practical answers to product and account questions.", zh: "围绕产品与账户问题，查阅实用文章。" } },
  { ...siteLinks.updates, description: { en: "See what changed and where to use it.", zh: "了解最近改了什么，以及在哪里使用。" } }
];
export const guideNavigation: readonly SiteLink[] = [
  { ...siteLinks.resources, label: { en: "All resources", zh: "资源总览" } },
  ...guideDirectories
];
export const customerFooterLinks = [siteLinks.help, siteLinks.privacy, siteLinks.terms];
export const siteLegalLinks = [siteLinks.privacy, siteLinks.terms];

export function siteLinkForPath(path: string): SiteLink | undefined {
  if (!path) return undefined;
  return Object.values(siteLinks).find(item => item.href === stripLocalePrefix(path.split(/[?#]/, 1)[0]));
}

export type PublicBreadcrumb = { href?: string; label: string };

/** Two levels below home: the section, then the page. Navigation changes never move a page. */
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
  return items;
}

export function customerBreadcrumbs(path: string, locale: Locale): Array<{ href: string; label: string }> {
  const current = [siteLinks.account, siteLinks.billing, siteLinks.referrals, siteLinks.credits].find(item => item.href === path.split(/[?#]/, 1)[0]) || siteLinks.account;
  const items = [siteLinks.home, current];
  return items.map(item => ({ href: localizeHref(item.href, locale), label: item.label[locale] }));
}
