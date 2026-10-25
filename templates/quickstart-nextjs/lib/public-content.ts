import type { Metadata } from "next";
import { apiRequest, ApiError, type PublishedArticle, type WebsiteSettings } from "./api";
import { appEnv } from "./env";
import { canonicalURL, publicMetadata, publicPageMetadata } from "./seo";
import { localizeHref, type Locale } from "./locale";
import type { PageResult } from "./operations-api";
import { availableDirectories, discoveryEntry, isContentDirectory, isEditorialPath, navigationEntries, type DiscoveryEntry, type NavigationIndex } from "./content-discovery";
import { staticContentIndex } from "./static-content-index";
import { managedPagePaths, pagePresentation } from "./page-presentation";
import { siteStructuredData } from "./structured-data";
import { siteConfig } from "../content/site";
export { managedPagePaths } from "./page-presentation";

const publicRead = { cache: "force-cache" as const, next: { revalidate: 30 } };
export function isManagedPath(path: string): boolean {
 return managedPagePaths.includes(path) || isEditorialPath(path);
}
/** Unconfigured previews and older APIs without CMS routes retain compatibility.
 * A CMS returning a successful empty response never resurrects bundled articles. */
export async function publishedIndex(locale: Locale): Promise<PublishedArticle[] | null> {
 if (!appEnv.apiConfigured) return null;
 const items: PublishedArticle[] = [];
 try {
  for (let page = 1; ; page++) {
   const result = await apiRequest<PageResult<PublishedArticle>>(`/site/articles?locale=${locale}&limit=100&page=${page}`, publicRead);
   items.push(...result.items); if (items.length >= result.total || result.items.length === 0) return items;
  }
 } catch (error) {
  if (error instanceof ApiError && error.status === 404) return null;
  throw error;
 }
}

async function siteCatalog(locale: Locale): Promise<{ entries: DiscoveryEntry[]; pages: Set<string> | null }> {
 const rows = await publishedIndex(locale);
 return rows === null
  ? { entries: staticContentIndex(locale), pages: null }
  : { entries: rows.map(discoveryEntry), pages: new Set(rows.filter(row => row.kind === "page").map(row => row.path)) };
}
export async function discoveryIndex(locale: Locale): Promise<DiscoveryEntry[]> {
 return (await siteCatalog(locale)).entries;
}
/** Directories with something to show in this language (see availableDirectories). */
export async function publishedDirectories(locale: Locale): Promise<Set<string>> {
 const { entries, pages } = await siteCatalog(locale);
 return new Set(availableDirectories(entries, pages));
}
/** The homepage identifies the site with the same description its search result uses. */
export async function homeStructuredData(locale: Locale) {
 const [metadata, site] = await Promise.all([managedMetadata("/", locale), websitePresentation()]);
 return siteStructuredData({
  name: site?.brand_name || appEnv.appName,
  description: typeof metadata?.description === "string" && metadata.description ? metadata.description : siteConfig.description[locale],
  path: localizeHref("/", locale), language: locale === "zh" ? "zh-CN" : "en",
  sameAs: siteConfig.repositoryURL ? [siteConfig.repositoryURL] : [],
  application: siteConfig.application
 });
}
export async function navigationIndex(locale: Locale): Promise<NavigationIndex> {
 const { entries, pages } = await siteCatalog(locale);
 return { entries: navigationEntries(entries), directories: availableDirectories(entries, pages) };
}
/** Enumerate the live build snapshot; dynamicParams still permits later publications. */
export async function publishedPaths(locale: Locale, fallback: string[]): Promise<string[]> {
 const rows = await publishedIndex(locale);
 if (rows === null) return fallback;
 return [...new Set([...fallback.filter(path => managedPagePaths.includes(path) || !isManagedPath(path)), ...rows.map(row => row.path)])];
}
export async function publishedArticle(path: string, locale: Locale): Promise<{ managed: boolean; article: PublishedArticle | null }> {
 if (!appEnv.apiConfigured) return { managed: false, article: null };
 try {
  const data = await apiRequest<{ article: PublishedArticle | null }>(`/site/articles/resolve?${new URLSearchParams({ path, locale })}`, publicRead);
  return { managed: true, article: data.article };
 } catch (error) {
  // Missing content is represented by 200 {article:null}. A route-level 404
  // instead means an older backend/preview without this endpoint.
  if (error instanceof ApiError && error.status === 404) return { managed: false, article: null };
  throw error;
 }
}
export async function websitePresentation(): Promise<WebsiteSettings | null> {
 if (!appEnv.apiConfigured) return null;
 const value = await apiRequest<WebsiteSettings>("/site/settings", publicRead);
 return value.presentation_version === 1 ? value : null;
}
export function applyWebsiteMetadata(metadata: Metadata, site: WebsiteSettings | null, home = false, locale: Locale = "en"): Metadata {
 if (!site) return metadata;
 const brand = site.brand_name || appEnv.appName;
 const original = typeof metadata.title === "string" ? metadata.title : appEnv.appName;
 const title = home ? `${(locale === "zh" ? site.headline_zh : site.headline_en) || brand} · ${brand}` : original.replace(` · ${appEnv.appName}`, ` · ${brand}`);
 const description = home ? (locale === "zh" ? site.description_zh : site.description) || metadata.description : metadata.description;
 return { ...metadata, title, description, openGraph: { ...metadata.openGraph, title, description: description || undefined, siteName: brand }, twitter: { ...metadata.twitter, title, description: description || undefined } };
}
export async function managedMetadata(path: string, locale: Locale): Promise<Metadata | null> {
 let metadata = publicPageMetadata(path, locale);
 if (isManagedPath(path)) {
  const resolved = await publishedArticle(path, locale);
  if (resolved.managed) {
   const article = resolved.article;
   if (!article) {
    if (!managedPagePaths.includes(path)) return null;
    const site = await websitePresentation();
    const page = pagePresentation(path, locale, site, appEnv.appName);
    const base = publicMetadata(page.translation.title, page.translation.summary, localizeHref(path, locale));
    // Keep navigation usable while preventing empty shells from being indexed.
    return { ...applyWebsiteMetadata(base, site), robots: { index: false, follow: true } };
   }
   const text = article.translation; const title = text.seo_title || text.title; const description = text.seo_description || text.summary;
   const languages = {...Object.fromEntries(article.locales.map(language => [language === "zh" ? "zh-CN" : "en", canonicalURL(localizeHref(path, language))])), "x-default":canonicalURL(localizeHref(path,article.locales.includes("en")?"en":article.locales[0]))};
   const base = publicMetadata(title, description, localizeHref(path, locale), article.kind === "page" ? undefined : { publishedAt: article.published_at });
   metadata = { ...base, alternates: { canonical: canonicalURL(localizeHref(path, locale)), languages }, openGraph: { ...base.openGraph, alternateLocale: article.locales.filter(value => value !== locale).map(value => value === "zh" ? "zh_CN" : "en_US") } };
  }
 }
 if (!metadata) return null;
 metadata = applyWebsiteMetadata(metadata, await websitePresentation(), false, locale);
 return isContentDirectory(path) ? withDirectoryAvailability(metadata, path, locale) : metadata;
}
/** An empty directory stays reachable but leaves search results, and no language version points to it. */
async function withDirectoryAvailability(metadata: Metadata, path: string, locale: Locale): Promise<Metadata> {
 const [en, zh] = await Promise.all([publishedDirectories("en"), publishedDirectories("zh")]);
 const visible = { en: en.has(path), zh: zh.has(path) };
 if (visible.en && visible.zh) return metadata;
 const languages = Object.fromEntries(Object.entries(metadata.alternates?.languages || {}).filter(([language]) =>
  language === "x-default" ? visible.en || visible.zh : language === "zh-CN" ? visible.zh : visible.en));
 if (!visible.en && visible.zh && "x-default" in languages) languages["x-default"] = canonicalURL(localizeHref(path, "zh"));
 return { ...metadata, alternates: { ...metadata.alternates, languages }, ...(visible[locale] ? {} : { robots: { index: false, follow: true } }) };
}
