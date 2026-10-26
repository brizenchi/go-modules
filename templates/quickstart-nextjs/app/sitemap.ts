import type { MetadataRoute } from "next";
import { exposedSitemapEntries, languageAlternates, sitemapEntries, canonicalURL } from "@/lib/seo";
import { publishedDirectories, publishedIndex, isManagedPath } from "@/lib/public-content";
import { isEditorialPath } from "@/lib/content-discovery";
import { contentHref, contentLocales, isInterfaceLocale, stripLocalePrefix } from "@/lib/locale";
export const revalidate=30;
export default async function sitemap():Promise<MetadataRoute.Sitemap>{
 const [indexes,enDirectories,zhDirectories]=await Promise.all([Promise.all(contentLocales.map(locale=>publishedIndex(locale))),publishedDirectories("en"),publishedDirectories("zh")]);
 const directories={en:enDirectories,zh:zhDirectories};
 if(indexes.some(rows=>rows===null))return exposedSitemapEntries(sitemapEntries(),directories);
 const entries:MetadataRoute.Sitemap=sitemapEntries().filter(entry=>!isManagedPath(stripLocalePrefix(new URL(entry.url).pathname)));
 contentLocales.forEach((locale,index)=>{for(const row of indexes[index]||[]){
  // Pages and directories exist in the interface languages; articles in every language.
  if(!isInterfaceLocale(locale)&&!isEditorialPath(row.path))continue;
  entries.push({url:canonicalURL(contentHref(row.path,locale)),lastModified:row.published_at,alternates:{languages:languageAlternates(row.path,row.locales)}});
 }});
 return exposedSitemapEntries(Array.from(new Map(entries.map(entry=>[entry.url,entry])).values()),directories);
}
