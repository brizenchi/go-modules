import type { PublishedArticle } from "./api";
import type { GuideIndexEntry } from "./guide-index";
import { stripLocalePrefix } from "./locale";

export type DiscoveryEntry = GuideIndexEntry & {
  published_at?: string;
  position?: number;
  links?: string[];
};

export function contentReferences(markdown: string): string[] {
  return [...new Set([...markdown.matchAll(/\]\((\/[^\s)]+)(?:\s+"[^"]*")?\)/g)]
    .map(match => stripLocalePrefix(match[1].split(/[?#]/, 1)[0]))
    .filter(path => !path.startsWith("//") && !path.startsWith("/api/")))];
}

/** Send card metadata and reference paths to the browser, not every article body. */
export function discoveryEntry(article: PublishedArticle): DiscoveryEntry {
  return {
    id: article.id, path: article.path, kind: article.kind,
    published_at: article.published_at, position: article.position,
    translation: { title: article.translation.title, summary: article.translation.summary },
    links: article.references || contentReferences(article.translation.body)
  };
}

export function uniqueContent(entries: readonly DiscoveryEntry[]): DiscoveryEntry[] {
  const seen = new Set<string>();
  return entries.filter(entry => {
    if (!entry.path.startsWith(`/${entry.kind}/`) || entry.path.includes("?") || entry.path.includes("#") || seen.has(entry.path)) return false;
    seen.add(entry.path);
    return true;
  });
}

/** Directories that list published pages of these kinds; Guides gathers documentation and articles. */
const kindsByDirectory: Readonly<Record<string, readonly string[]>> = {
  "/features": ["features"], "/use-cases": ["use-cases"], "/compare": ["compare"],
  "/docs": ["docs"], "/blog": ["blog"], "/resources": ["docs", "blog"]
};
/** Directories that are themselves a single published page. */
const pageDirectories: readonly string[] = ["/about", "/updates"];

const editorialPath = /^\/(blog|docs|features|use-cases|compare)\/[a-z0-9]+(?:-[a-z0-9]+)*$/;
/** Paths an editor can publish under a directory (`/docs/setup`), as validated by the content API. */
export function isEditorialPath(path: string): boolean { return editorialPath.test(path); }

/** Links in published text may reach only pages and directories that exist in this language. */
export function linkablePath(published: ReadonlySet<string>, directories: ReadonlySet<string> | null): (href: string) => boolean {
  return href => {
    const path = stripLocalePrefix(href.split(/[?#]/, 1)[0]) || "/";
    if (isEditorialPath(path)) return published.has(path);
    if (isContentDirectory(path)) return !directories || directories.has(path);
    return true;
  };
}

/** Article kinds a directory lists; empty for single-page and code-owned sections. */
export function directoryKinds(path: string): readonly string[] {
  return Object.hasOwn(kindsByDirectory, path) ? kindsByDirectory[path] : [];
}

export function isContentDirectory(path: string): boolean {
  return Object.hasOwn(kindsByDirectory, path) || pageDirectories.includes(path);
}

/** A directory exists in a language only when it has something to show. Templates
 * differ: an unused section stays out of navigation, the sitemap and search results.
 * `publishedPages` is null without a CMS, where bundled pages always exist. */
export function availableDirectories(entries: readonly DiscoveryEntry[], publishedPages: ReadonlySet<string> | null): string[] {
  const kinds = new Set(uniqueContent(entries).map(entry => entry.kind));
  return [
    ...Object.entries(kindsByDirectory).filter(([, needs]) => needs.some(kind => kinds.has(kind as DiscoveryEntry["kind"]))).map(([path]) => path),
    ...pageDirectories.filter(path => !publishedPages || publishedPages.has(path))
  ];
}

export type NavigationEntry = { path: string; kind: DiscoveryEntry["kind"]; title: string; position?: number };
export type NavigationIndex = { entries: NavigationEntry[]; directories: string[] };

/** Header menus may list any published article, but need only its title and order. */
export function navigationEntries(entries: readonly DiscoveryEntry[]): NavigationEntry[] {
  return uniqueContent(entries).map(entry => ({ path: entry.path, kind: entry.kind, title: entry.translation.title, position: entry.position }));
}

export function latestGuides(entries: readonly DiscoveryEntry[]): DiscoveryEntry[] {
  return uniqueContent(entries)
    .filter(entry => (entry.kind === "docs" || entry.kind === "blog") && Number.isFinite(Date.parse(entry.published_at || "")))
    .sort((a, b) => Date.parse(b.published_at!) - Date.parse(a.published_at!) || (a.position ?? 0) - (b.position ?? 0) || a.path.localeCompare(b.path))
    .slice(0, 3);
}

export function relatedContent(entries: readonly DiscoveryEntry[], currentPath: string): DiscoveryEntry[] {
  const catalog = uniqueContent(entries);
  const current = catalog.find(entry => entry.path === currentPath);
  if (!current) return [];
  const score = (entry: DiscoveryEntry) => (current.links?.includes(entry.path) ? 4 : 0)
    + (entry.links?.includes(current.path) ? 2 : 0)
    + (entry.kind === current.kind ? 1 : 0);
  return catalog.filter(entry => entry.path !== current.path && score(entry) > 0)
    .sort((a, b) => score(b) - score(a) || (a.position ?? 0) - (b.position ?? 0) || a.path.localeCompare(b.path))
    .slice(0, 3);
}
