import type { Metadata } from "next";
import { apiRequest, ApiError, type PublishedArticle, type WebsiteSettings } from "./api";
import { appEnv } from "./env";
import { canonicalURL, publicMetadata, publicPageMetadata } from "./seo";
import { localizeHref, type Locale } from "./locale";
import type { PageResult } from "./operations-api";

const publicRead = { cache: "force-cache" as const, next: { revalidate: 30 } };
/** No configured API means a self-contained static template preview. An older
 * API may omit the new endpoints; other runtime failures are not treated as an
 * instruction to resurrect unpublished content from starter files. */
export async function publishedIndex(locale: Locale): Promise<PublishedArticle[] | null> {
 if (!appEnv.apiConfigured) return null;
 const items: PublishedArticle[] = [];
 try {
  for (let page = 1; ; page++) {
   const result = await apiRequest<PageResult<PublishedArticle>>(`/site/articles?locale=${locale}&limit=100&page=${page}`, publicRead);
   items.push(...result.items); if (items.length >= result.total || result.items.length === 0) return items;
  }
 } catch (error) { if (error instanceof ApiError && error.status === 404) return null; throw error; }
}
/** Enumerate the live build snapshot; dynamicParams still permits later publications. */
export async function publishedPaths(locale: Locale, fallback: string[]): Promise<string[]> {
 const rows = await publishedIndex(locale);
 if (rows === null) return fallback;
 return [...new Set([...fallback.filter(path => !/^\/(blog|docs|features|use-cases|compare)\/.+/.test(path)), ...rows.map(row => row.path)])];
}
export async function publishedArticle(path: string, locale: Locale): Promise<{ managed: boolean; article: PublishedArticle | null }> {
 if (!appEnv.apiConfigured) return { managed: false, article: null };
 try { const data = await apiRequest<{ article: PublishedArticle | null }>(`/site/articles/resolve?${new URLSearchParams({ path, locale })}`, publicRead); return { managed: true, article: data.article }; }
 catch (error) { if (error instanceof ApiError && error.status === 404) return { managed: false, article: null }; throw error; }
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
 if (/^\/(blog|docs|features|use-cases|compare)\/.+/.test(path)) {
  const resolved = await publishedArticle(path, locale);
  if (resolved.managed) {
   const article = resolved.article; if (!article) return null;
   const text = article.translation; const title = text.seo_title || text.title; const description = text.seo_description || text.summary;
   const languages = {...Object.fromEntries(article.locales.map(language => [language === "zh" ? "zh-CN" : "en", canonicalURL(localizeHref(path, language))])), "x-default":canonicalURL(localizeHref(path,article.locales.includes("en")?"en":article.locales[0]))};
   const base = publicMetadata(title, description, localizeHref(path, locale), { publishedAt: article.published_at });
   metadata = { ...base, alternates: { canonical: canonicalURL(localizeHref(path, locale)), languages }, openGraph: { ...base.openGraph, alternateLocale: article.locales.filter(value => value !== locale).map(value => value === "zh" ? "zh_CN" : "en_US") } };
  }
 }
 if (!metadata) return null;
 return applyWebsiteMetadata(metadata, await websitePresentation(), path === "/", locale);
}
