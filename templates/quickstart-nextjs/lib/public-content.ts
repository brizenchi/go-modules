import type { Metadata } from "next";
import { apiRequest, type PublishedArticle, type WebsiteSettings } from "./api";
import { appEnv } from "./env";
import { canonicalURL, publicMetadata, publicPageMetadata } from "./seo";
import { localizeHref, type Locale } from "./locale";
import type { PageResult } from "./operations-api";
import { discoveryEntry, type DiscoveryEntry } from "./content-discovery";
import { staticContentIndex } from "./static-content-index";

const publicRead = { cache: "force-cache" as const, next: { revalidate: 30 } };
export const managedPagePaths = ["/", "/features", "/use-cases", "/compare", "/resources", "/docs", "/blog", "/about", "/updates"];
export function isManagedPath(path: string): boolean {
 return managedPagePaths.includes(path) || /^\/(blog|docs|features|use-cases|compare)\/[a-z0-9]+(?:-[a-z0-9]+)*$/.test(path);
}
/** Only an explicitly unconfigured preview can use bundled content. A configured
 * CMS must never resurrect missing or unpublished content from frontend files. */
export async function publishedIndex(locale: Locale): Promise<PublishedArticle[] | null> {
 if (!appEnv.apiConfigured) return null;
 const items: PublishedArticle[] = [];
 for (let page = 1; ; page++) {
   const result = await apiRequest<PageResult<PublishedArticle>>(`/site/articles?locale=${locale}&limit=100&page=${page}`, publicRead);
   items.push(...result.items); if (items.length >= result.total || result.items.length === 0) return items;
 }
}

export async function discoveryIndex(locale: Locale): Promise<DiscoveryEntry[]> {
 const entries = await publishedIndex(locale);
 return entries === null ? staticContentIndex(locale) : entries.map(discoveryEntry);
}
/** Enumerate the live build snapshot; dynamicParams still permits later publications. */
export async function publishedPaths(locale: Locale, fallback: string[]): Promise<string[]> {
 const rows = await publishedIndex(locale);
 if (rows === null) return fallback;
 return [...new Set([...fallback.filter(path => !isManagedPath(path)), ...rows.map(row => row.path)])];
}
export async function publishedArticle(path: string, locale: Locale): Promise<{ managed: boolean; article: PublishedArticle | null }> {
 if (!appEnv.apiConfigured) return { managed: false, article: null };
 const data = await apiRequest<{ article: PublishedArticle | null }>(`/site/articles/resolve?${new URLSearchParams({ path, locale })}`, publicRead); return { managed: true, article: data.article };
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
   const article = resolved.article; if (!article) return null;
   const text = article.translation; const title = text.seo_title || text.title; const description = text.seo_description || text.summary;
   const languages = {...Object.fromEntries(article.locales.map(language => [language === "zh" ? "zh-CN" : "en", canonicalURL(localizeHref(path, language))])), "x-default":canonicalURL(localizeHref(path,article.locales.includes("en")?"en":article.locales[0]))};
   const base = publicMetadata(title, description, localizeHref(path, locale), article.kind === "page" ? undefined : { publishedAt: article.published_at });
   metadata = { ...base, alternates: { canonical: canonicalURL(localizeHref(path, locale)), languages }, openGraph: { ...base.openGraph, alternateLocale: article.locales.filter(value => value !== locale).map(value => value === "zh" ? "zh_CN" : "en_US") } };
  }
 }
 if (!metadata) return null;
 return applyWebsiteMetadata(metadata, await websitePresentation(), false, locale);
}
