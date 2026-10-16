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
    .filter(path => !path.startsWith("//")))];
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
