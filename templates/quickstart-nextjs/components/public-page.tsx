import { discoveryIndex, isManagedPath, managedPagePaths, publishedArticle, websitePresentation } from "@/lib/public-content";
import { pagePresentation } from "@/lib/page-presentation";
import { appEnv } from "@/lib/env";
import type { Locale } from "@/lib/locale";
import { PublishedContentArticle, PublishedContentIndex } from "./published-content";
import { ManagedHome } from "./content/managed-home";
import { ManagedPage } from "./managed-page";
import { notFound } from "next/navigation";
import { HomeContent } from "./content/home";
import { ProductPricing } from "./content/pricing";
import { Documentation } from "./content/docs";
import { BlogIndex, BlogArticle } from "./content/blog";
import { GuideArticle } from "./content/guide-article";
import { ProductUpdates } from "./content/updates";
import { ContactContent } from "./content/contact";
import { PolicyContent } from "./content/policy";
import { MarketingIndex, MarketingDetail, AboutPageContent } from "./content/marketing";
import { marketingCategories, getMarketingPage } from "@/content/marketing";
import { findDocumentation } from "@/content/docs";
import { findArticle } from "@/content/articles";

export async function PublicPage({ path, locale="en" }: { path: string; locale?: Locale }) {
 if (isManagedPath(path)) {
  const resolved = await publishedArticle(path, locale);
  if (resolved.managed) {
   if (!resolved.article && !managedPagePaths.includes(path)) notFound();
   const page = resolved.article || pagePresentation(path, locale, await websitePresentation(), appEnv.appName);
   const entries = await discoveryIndex(locale);
   if (path === "/") return <ManagedHome page={page} entries={entries} identityOnly={!resolved.article} />;
   if (["/blog", "/docs", "/features", "/use-cases", "/compare", "/resources"].includes(path)) {
    return <PublishedContentIndex kind={path.slice(1) as "blog" | "docs" | "features" | "use-cases" | "compare" | "resources"} page={page} entries={entries} />;
   }
   if (!resolved.article || resolved.article.kind === "page") return <ManagedPage page={page} />;
   return <PublishedContentArticle article={resolved.article} entries={entries} />;
  }
 }

  if (path === "/") return <HomeContent entries={await discoveryIndex(locale)} />;
  if (path === "/pricing") return <ProductPricing />;
  if (path === "/docs") return <Documentation />;
  if (path === "/blog") return <BlogIndex />;
  if (path === "/updates") return <ProductUpdates />;
  if (path === "/contact") return <ContactContent />;
  if (path === "/privacy" || path === "/terms") return <PolicyContent kind={path === "/privacy" ? "privacy" : "terms"} />;
  if (path === "/about") return <AboutPageContent />;
  if (path === "/resources") {
    return <PublishedContentIndex kind="resources" entries={await discoveryIndex(locale)} />;
  }
  const category = marketingCategories.find(item => item.path === path);
  if (category) return <MarketingIndex kind={category.kind} />;
  const page = getMarketingPage(path);
  if (page) return <MarketingDetail page={page} entries={await discoveryIndex(locale)} />;
  const guide = path.startsWith("/docs/") ? findDocumentation(path.slice(6)) : undefined;
  if (guide) return <GuideArticle guide={guide} entries={await discoveryIndex(locale)} />;
  const article = path.startsWith("/blog/") ? findArticle(path.slice(6)) : undefined;
  if (article) return <BlogArticle article={article} entries={await discoveryIndex(locale)} />;
  notFound();
}
