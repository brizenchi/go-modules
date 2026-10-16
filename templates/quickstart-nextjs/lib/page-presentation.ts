import type { PublishedArticle, WebsiteSettings } from "./api";
import type { Locale } from "./locale";

export type PagePresentation = Pick<PublishedArticle, "path" | "locales" | "translation">;
export const pageLabels = {
 "/": { en: "Home", zh: "首页" },
 "/features": { en: "Features", zh: "功能介绍" },
 "/use-cases": { en: "Use cases", zh: "使用场景" },
 "/compare": { en: "Comparisons", zh: "方案比较" },
 "/resources": { en: "Guides", zh: "指南" },
 "/docs": { en: "Documentation", zh: "使用文档" },
 "/blog": { en: "Articles", zh: "产品文章" },
 "/about": { en: "About", zh: "关于我们" },
 "/updates": { en: "Updates", zh: "更新日志" }
};
export const managedPagePaths = Object.keys(pageLabels);

/** Structural routes outlive their editorial copy. Before a page is published,
 * use only saved site identity and navigation labels: never a seed article, an
 * invented publication date or another language's draft. */
export function pagePresentation(path: string, locale: Locale, site: WebsiteSettings | null, defaultBrand: string): PagePresentation {
 const title = path === "/" ? (locale === "zh" ? site?.headline_zh : site?.headline_en) || site?.brand_name || defaultBrand
  : pageLabels[path as keyof typeof pageLabels]?.[locale] || defaultBrand;
 const summary = path === "/" ? (locale === "zh" ? site?.description_zh : site?.description) || "" : "";
 return { path, locales: ["en", "zh"], translation: { title, summary, body: "", seo_title: "", seo_description: "" } };
}
