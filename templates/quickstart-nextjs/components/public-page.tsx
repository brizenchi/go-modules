import { publishedArticle, publishedIndex } from "@/lib/public-content";
import type { Locale } from "@/lib/locale";
import { PublishedContentArticle, PublishedContentIndex } from "./published-content";
import { notFound } from "next/navigation";
import HomePage from "@/app/(default)/(public)/page";
import { ProductPricing } from "./content/pricing";
import { Documentation } from "./content/docs";
import { BlogIndex, BlogArticle } from "./content/blog";
import { GuideArticle } from "./content/guide-article";
import { ProductUpdates } from "./content/updates";
import { ContactContent } from "./content/contact";
import { PolicyContent } from "./content/policy";
import { MarketingIndex, MarketingDetail, AboutPageContent, ResourcesPageContent } from "./content/marketing";
import { marketingCategories, getMarketingPage } from "@/content/marketing";
import { findDocumentation } from "@/content/docs";
import { findArticle } from "@/content/articles";

export async function PublicPage({ path, locale="en" }: { path: string; locale?: Locale }) {
 if (/^\/(blog|docs|features|use-cases|compare)\/.+/.test(path)) {
  const resolved=await publishedArticle(path,locale);
  if(resolved.managed){if(!resolved.article)notFound();return <PublishedContentArticle article={resolved.article}/>;}
 }
 if (["/blog","/docs","/features","/use-cases","/compare","/resources"].includes(path)) {
  const entries=await publishedIndex(locale);
  if(entries)return <PublishedContentIndex kind={path.slice(1) as "blog"|"docs"|"features"|"use-cases"|"compare"|"resources"} entries={entries}/>;
 }

  if (path === "/") return <HomePage />;
  if (path === "/pricing") return <ProductPricing />;
  if (path === "/docs") return <Documentation />;
  if (path === "/blog") return <BlogIndex />;
  if (path === "/updates") return <ProductUpdates />;
  if (path === "/contact") return <ContactContent />;
  if (path === "/privacy" || path === "/terms") return <PolicyContent kind={path === "/privacy" ? "privacy" : "terms"} />;
  if (path === "/about") return <AboutPageContent />;
  if (path === "/resources") return <ResourcesPageContent />;
  const category = marketingCategories.find(item => item.path === path);
  if (category) return <MarketingIndex kind={category.kind} />;
  const page = getMarketingPage(path);
  if (page) return <MarketingDetail page={page} />;
  const guide = path.startsWith("/docs/") ? findDocumentation(path.slice(6)) : undefined;
  if (guide) return <GuideArticle guide={guide} />;
  const article = path.startsWith("/blog/") ? findArticle(path.slice(6)) : undefined;
  if (article) return <BlogArticle article={article} />;
  notFound();
}
