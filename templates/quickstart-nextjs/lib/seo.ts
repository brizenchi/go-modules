import type { Metadata } from "next";
import { articles } from "../content/articles";
import { siteConfig } from "../content/site";
import { documentation } from "../content/docs";
import { appEnv } from "./env";
import { marketingRoutes } from "../content/marketing";
import { localizeHref, localeFromPath, stripLocalePrefix, type Locale } from "./locale";

export const siteDescription = siteConfig.description.en;
export const siteDescriptions = siteConfig.description;
export const siteOrigin = new URL(appEnv.appUrl).origin;
export const publicPaths = ["/", "/pricing", "/docs", "/blog", "/updates", "/contact", "/privacy", "/terms", ...marketingRoutes.map(page => page.path)];
export const privatePaths = ["/account", "/dashboard", "/billing", "/referrals", "/credits", "/files", "/notes", "/admin", "/login", "/invite", "/oauth", "/api"];

export function canonicalURL(path: string): string {
  // Canonicals never carry invitation codes, checkout markers, or other queries.
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("\\")) throw new Error("Canonical path must be local.");
  const url = new URL(path, siteOrigin);
  if (url.origin !== siteOrigin) throw new Error("Canonical path must stay on the site origin.");
  url.search = "";
  url.hash = "";
  return url.toString();
}

export function publicMetadata(title: string, description: string, path: string, article?: { publishedAt?: string }): Metadata {
  const url = canonicalURL(path);
  const locale = localeFromPath(path);
  const basePath = stripLocalePrefix(path);
  const languages = {
    en: canonicalURL(localizeHref(basePath, "en")),
    "zh-CN": canonicalURL(localizeHref(basePath, "zh")),
    "x-default": canonicalURL(localizeHref(basePath, "en"))
  };
  const fullTitle = `${title} · ${appEnv.appName}`;
  return {
    title: fullTitle,
    description,
    alternates: { canonical: url, languages },
    openGraph: {
      title: fullTitle, description, url, siteName: appEnv.appName,
      type: article ? "article" : "website",
      locale: locale === "zh" ? "zh_CN" : "en_US",
      alternateLocale: [locale === "zh" ? "en_US" : "zh_CN"],
      images: [{ url: canonicalURL("/opengraph-image"), width: 1200, height: 630, alt: appEnv.appName }],
      ...(article?.publishedAt ? { publishedTime: article.publishedAt } : {})
    },
    twitter: { card: "summary_large_image", title: fullTitle, description, images: [canonicalURL("/opengraph-image")] }
  };
}

export function privateMetadata(title?: string): Metadata {
  return {
    ...(title ? { title } : {}),
    robots: { index: false, follow: false },
    alternates: { canonical: null, languages: {} },
    openGraph: null,
    twitter: null
  };
}

export function publicContentPaths(): string[] {
  return [...new Set([
    ...publicPaths,
    ...documentation.map(guide => `/docs/${guide.id}`),
    ...articles.map(article => `/blog/${article.slug}`)
  ])];
}

export function sitemapEntries(): Array<{ url: string; lastModified?: string; alternates: { languages: Record<string, string> } }> {
  const pages = [
    ...publicPaths.map((path) => ({ url: canonicalURL(path) })),
    ...documentation.map((guide) => ({ url: canonicalURL(`/docs/${guide.id}`) })),
    ...articles.map((article) => ({ url: canonicalURL(`/blog/${article.slug}`), lastModified: article.publishedAt }))
  ];
  return pages.flatMap(page => {
    const path = new URL(page.url).pathname;
    const languages = { en: page.url, "zh-CN": canonicalURL(localizeHref(path, "zh")) };
    return [page.url, languages["zh-CN"]].map(url => ({ ...page, url, alternates: { languages } }));
  });
}

const publicPageCopy: Record<string, { title: Record<Locale, string>; description: Record<Locale, string> }> = {
  "/": { title: siteConfig.homepageTitle, description: siteDescriptions },
  "/pricing": { title: { en: "Pricing", zh: "价格方案" }, description: { en: "Compare subscriptions, lifetime access, and credit packages in the example SaaS catalog. The starter template is free.", zh: "比较示例 SaaS 的周期订阅、终身权益和积分包。启动模板本身免费。" } },
  "/docs": { title: { en: "Template user guide", zh: "模板使用指南" }, description: { en: "Learn what the free SaaS template includes, try sign-in, payments and referrals, then configure your own website and administrator account.", zh: "了解免费 SaaS 模板的功能，体验登录、支付和邀请流程，再配置自己的网站和管理员账户。" } },
  "/blog": { title: { en: "Product guides", zh: "产品文章" }, description: { en: "Practical guides to configure your SaaS, understand invitation rewards, and support your users.", zh: "通过实用指南配置你的 SaaS、了解邀请奖励，并为用户提供支持。" } },
  "/updates": { title: { en: "Product updates", zh: "产品更新" }, description: { en: "Follow changes to the SaaS starter, with links to the features and guides.", zh: "查看 SaaS 启动模板的更新记录，以及相关功能与指南。" } },
  "/contact": { title: { en: "Contact and support", zh: "联系与支持" }, description: { en: "Find product guides and the support channels configured for this site.", zh: "查阅产品指南，联系本站提供的支持渠道。" } },
  "/privacy": { title: { en: "Privacy", zh: "隐私政策" }, description: { en: "Editable privacy information for the SaaS starter. Adapt it to your actual service.", zh: "SaaS 启动模板的隐私说明，请按实际服务完善后使用。" } },
  "/terms": { title: { en: "Terms", zh: "服务条款" }, description: { en: "Editable service terms covering accounts, subscriptions, credits, and support for your product.", zh: "涵盖账户、订阅、积分与支持的服务条款示例，请按实际产品调整。" } }
};

/** One metadata resolver keeps translated routes aligned with their content catalog. */
export function publicPageMetadata(path: string, locale: Locale): Metadata | null {
  const basePath = stripLocalePrefix(path).split(/[?#]/, 1)[0];
  const localizedPath = localizeHref(basePath, locale);
  const copy = publicPageCopy[basePath] || marketingRoutes.find(page => page.path === basePath);
  if (copy) return publicMetadata(copy.title[locale], copy.description[locale], localizedPath);
  const guide = documentation.find(item => `/docs/${item.id}` === basePath);
  if (guide) return publicMetadata(guide.title[locale], guide.summary[locale], localizedPath, {});
  const article = articles.find(item => `/blog/${item.slug}` === basePath);
  if (article) return publicMetadata(article.title[locale], article.summary[locale], localizedPath, { publishedAt: article.publishedAt });
  return null;
}
