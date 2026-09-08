import { canonicalURL } from "./seo";

export type BreadcrumbItem = { href?: string; label: string };

export function breadcrumbStructuredData(items: BreadcrumbItem[], path: string) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((item, index) => ({
      "@type": "ListItem",
      position: index + 1,
      name: item.label,
      item: canonicalURL(index === items.length - 1 ? path : item.href ?? path)
    }))
  };
}

export function articleStructuredData(article: {
  title: string;
  description: string;
  path: string;
  language: string;
  publishedAt?: string;
}) {
  const url = canonicalURL(article.path);
  return {
    "@context": "https://schema.org",
    "@type": "Article",
    "@id": `${url}#article`,
    url,
    mainEntityOfPage: { "@type": "WebPage", "@id": url },
    headline: article.title,
    description: article.description,
    inLanguage: article.language,
    ...(article.publishedAt ? { datePublished: article.publishedAt } : {})
  };
}

export function serializeStructuredData(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
