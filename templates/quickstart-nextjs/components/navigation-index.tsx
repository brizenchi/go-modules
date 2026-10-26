"use client";
import { createContext, useContext, useMemo } from "react";
import { directoryFilter } from "@/content/navigation";
import { linkablePath, type DiscoveryEntry, type NavigationIndex } from "@/lib/content-discovery";
import { useI18n } from "@/lib/i18n";
import { contentHref, isInterfaceLocale, stripLocalePrefix, type ContentLocale, type Locale } from "@/lib/locale";

type NavigationIndexes = Partial<Record<Locale, NavigationIndex>>;
const NavigationIndexContext = createContext<NavigationIndexes>({});

/** Published pages and directories per language, loaded by the layouts that render the public header. */
export function NavigationIndexProvider({ value, children }: { value: NavigationIndexes; children: React.ReactNode }) {
  return <NavigationIndexContext.Provider value={value}>{children}</NavigationIndexContext.Provider>;
}
/** `null` when this page did not load the index; callers then show every directory. */
export function useNavigationIndex(): NavigationIndex | null {
  const { locale } = useI18n();
  return useContext(NavigationIndexContext)[locale] ?? null;
}
export function usePublishedDirectories(): ReadonlySet<string> | null {
  const index = useNavigationIndex();
  return useMemo(() => index ? new Set(index.directories) : null, [index]);
}
/** Whether a link may be shown: empty directories are hidden everywhere. */
export function useDirectoryFilter(): (href: string) => boolean {
  const directories = usePublishedDirectories();
  return useMemo(() => directoryFilter(directories), [directories]);
}
/** Internal links in published text that still have a destination in this language. */
export function useLinkablePath(entries: readonly Pick<DiscoveryEntry, "path">[]): (href: string) => boolean {
  const directories = usePublishedDirectories();
  return useMemo(() => linkablePath(new Set(entries.map(entry => entry.path)), directories), [entries, directories]);
}
/** Links in an article: to its own language where that page is published there,
 * otherwise to the interface version, and never to a page that does not exist. */
export function useArticleLinks(entries: readonly Pick<DiscoveryEntry, "path">[], locale: ContentLocale): { linkable: (href: string) => boolean; hrefFor?: (href: string) => string } {
  const index = useNavigationIndex();
  const directories = usePublishedDirectories();
  return useMemo(() => {
    const own = new Set(entries.map(entry => entry.path));
    if (isInterfaceLocale(locale)) return { linkable: linkablePath(own, directories) };
    const fallback = linkablePath(new Set((index?.entries ?? []).map(entry => entry.path)), directories);
    const base = (href: string) => stripLocalePrefix(href.split(/[?#]/, 1)[0]) || "/";
    return { linkable: href => own.has(base(href)) || fallback(href), hrefFor: href => own.has(base(href)) ? contentHref(href, locale) : href };
  }, [entries, locale, index, directories]);
}
