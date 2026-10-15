import type { MetadataRoute } from "next";
import { sitemapEntries, canonicalURL } from "@/lib/seo";
import { publishedIndex } from "@/lib/public-content";
import { localizeHref, stripLocalePrefix } from "@/lib/locale";
export const revalidate=30;
export default async function sitemap():Promise<MetadataRoute.Sitemap>{
 const [en,zh]=await Promise.all([publishedIndex("en"),publishedIndex("zh")]);
 if(en===null||zh===null)return sitemapEntries();
 const entries:MetadataRoute.Sitemap=sitemapEntries().filter(entry=>!/^\/(blog|docs|features|use-cases|compare)\/.+/.test(stripLocalePrefix(new URL(entry.url).pathname)));
 for(const [locale,rows] of [["en",en],["zh",zh]] as const)for(const row of rows){entries.push({url:canonicalURL(localizeHref(row.path,locale)),lastModified:row.published_at,alternates:{languages:Object.fromEntries(row.locales.map(language=>[language==="zh"?"zh-CN":"en",canonicalURL(localizeHref(row.path,language))]))}});}
 return Array.from(new Map(entries.map(entry=>[entry.url,entry])).values());
}
