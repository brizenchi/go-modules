import type { ContentKind, NavigationSection, PublishedArticle, SiteNavigation } from "./api";
import { isEditorialPath } from "./content-discovery";
import { navigationSections } from "../content/navigation";

/** The backend's limits (internal/feature/operations/navigation.go). */
export const navigationLimits = { header: 6, menu: 10, footer: 4, links: 8, label: 40 } as const;

export type ArticleOption = { path: string; kind: ContentKind; title: string };

/** Articles published in any language, once each; the first language listed wins the title. */
export function articleOptions(rows: readonly PublishedArticle[]): ArticleOption[] {
  const options = new Map<string, ArticleOption>();
  for (const row of rows) {
    if (isEditorialPath(row.path) && !options.has(row.path)) options.set(row.path, { path: row.path, kind: row.kind, title: row.translation.title });
  }
  return [...options.values()].sort((a, b) => a.kind.localeCompare(b.kind) || a.title.localeCompare(b.title));
}

export function moveItem<T>(list: readonly T[], index: number, offset: -1 | 1): T[] {
  const target = index + offset;
  if (index < 0 || target < 0 || index >= list.length || target >= list.length) return [...list];
  const next = [...list];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

export function removeItem<T>(list: readonly T[], index: number): T[] {
  return list.filter((_, position) => position !== index);
}

const allSections = Object.keys(navigationSections) as NavigationSection[];

export function unusedHeaderSections(navigation: SiteNavigation): NavigationSection[] {
  return allSections.filter(section => !navigation.header.some(item => item.section === section));
}

export function unusedFooterSections(navigation: SiteNavigation): NavigationSection[] {
  return allSections.filter(section => !navigation.footer.some(group => group.links.includes(section)));
}

/** A new header item lists its section's pages when it can, otherwise it is a plain link. */
export function addHeaderItem(navigation: SiteNavigation, section: NavigationSection): SiteNavigation {
  if (navigation.header.length >= navigationLimits.header || navigation.header.some(item => item.section === section)) return navigation;
  const menu = navigationSections[section].menu ? "auto" : "none";
  return { ...navigation, header: [...navigation.header, { section, label: { en: "", zh: "" }, menu, items: [] }] };
}

const validLabel = (value: string, required: boolean) => value.trim().length <= navigationLimits.label && !/[\r\n\0]/.test(value) && (!required || value.trim().length > 0);

/** The same rules the backend enforces, so the editor can explain a problem before saving. */
export function navigationProblems(navigation: SiteNavigation): Array<"header" | "label" | "menu" | "footer-title" | "footer-links"> {
  const problems = new Set<"header" | "label" | "menu" | "footer-title" | "footer-links">();
  if (navigation.header.length > navigationLimits.header || new Set(navigation.header.map(item => item.section)).size !== navigation.header.length) problems.add("header");
  for (const item of navigation.header) {
    if (!validLabel(item.label.en, false) || !validLabel(item.label.zh, false)) problems.add("label");
    if (item.menu === "manual" && (!item.items.length || item.items.length > navigationLimits.menu || new Set(item.items).size !== item.items.length || !item.items.every(isEditorialPath))) problems.add("menu");
    if (item.menu !== "none" && !navigationSections[item.section].menu) problems.add("menu");
  }
  const links = navigation.footer.flatMap(group => group.links);
  if (navigation.footer.length > navigationLimits.footer || new Set(links).size !== links.length) problems.add("footer-links");
  for (const group of navigation.footer) {
    if (!validLabel(group.title.en, true) || !validLabel(group.title.zh, true)) problems.add("footer-title");
    if (!group.links.length || group.links.length > navigationLimits.links) problems.add("footer-links");
  }
  return [...problems];
}

/** Trimmed copy as the backend stores it; manual items are dropped from other menu modes. */
export function navigationPayload(navigation: SiteNavigation): SiteNavigation {
  return {
    version: 1,
    header: navigation.header.map(item => ({ section: item.section, label: { en: item.label.en.trim(), zh: item.label.zh.trim() }, menu: item.menu, items: item.menu === "manual" ? item.items : [] })),
    footer: navigation.footer.map(group => ({ title: { en: group.title.en.trim(), zh: group.title.zh.trim() }, links: group.links }))
  };
}
