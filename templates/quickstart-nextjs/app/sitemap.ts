import type { MetadataRoute } from "next";
import { exposedSitemapEntries, sitemapEntries, canonicalURL } from "@/lib/seo";
import { publishedDirectories, publishedIndex, isManagedPath } from "@/lib/public-content";
import { localizeHref, stripLocalePrefix } from "@/lib/locale";
export const revalidate=30;
export default async function sitemap():Promise<MetadataRoute.Sitemap>{
 const [en,zh,enDirectories,zhDirectories]=await Promise.all([publishedIndex("en"),publishedIndex("zh"),publishedDirectories("en"),publishedDirectories("zh")]);
 const directories={en:enDirectories,zh:zhDirectories};
 if(en===null||zh===null)return exposedSitemapEntries(sitemapEntries(),directories);
 const entries:MetadataRoute.Sitemap=sitemapEntries().filter(entry=>!isManagedPath(stripLocalePrefix(new URL(entry.url).pathname)));
 for(const [locale,rows] of [["en",en],["zh",zh]] as const)for(const row of rows){
  const fallback=row.locales.includes("en")?"en":row.locales[0];
  // Same alternates as the page's hreflang tags, including x-default.
  const languages={...Object.fromEntries(row.locales.map(language=>[language==="zh"?"zh-CN":"en",canonicalURL(localizeHref(row.path,language))])),"x-default":canonicalURL(localizeHref(row.path,fallback))};
  entries.push({url:canonicalURL(localizeHref(row.path,locale)),lastModified:row.published_at,alternates:{languages}});
 }
 return exposedSitemapEntries(Array.from(new Map(entries.map(entry=>[entry.url,entry])).values()),directories);
}
