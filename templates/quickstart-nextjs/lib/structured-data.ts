import { canonicalURL } from "./seo";

export type BreadcrumbItem = { href?: string; label: string };
export type ArticleSchemaType = "Article" | "TechArticle" | "BlogPosting";

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

const organizationID = () => `${canonicalURL("/")}#organization`;
const publisherOf = (name?: string) => name ? { publisher: { "@type": "Organization", "@id": organizationID(), name } } : {};

/** Who publishes the site and what it is; emitted once, on each language's homepage. */
export function siteStructuredData(site: {
  name: string;
  description: string;
  path: string;
  language: string;
  sameAs?: string[];
  application?: { category: string; operatingSystem: string; price: string; priceCurrency: string };
}) {
  const url = canonicalURL(site.path);
  const organization = { "@id": organizationID() };
  return {
    "@context": "https://schema.org",
    "@graph": [
      { "@type": "Organization", ...organization, name: site.name, url: canonicalURL("/"), ...(site.sameAs?.length ? { sameAs: site.sameAs } : {}) },
      { "@type": "WebSite", "@id": `${url}#website`, url, name: site.name, description: site.description, inLanguage: site.language, publisher: organization },
      ...(site.application ? [{
        "@type": "SoftwareApplication", "@id": `${url}#application`, name: site.name, url, description: site.description,
        applicationCategory: site.application.category, operatingSystem: site.application.operatingSystem,
        offers: { "@type": "Offer", price: site.application.price, priceCurrency: site.application.priceCurrency },
        publisher: organization
      }] : [])
    ]
  };
}

/** Kinds that read as guides or posts keep an article type; product pages use webPageStructuredData. */
export function articleSchemaType(kind: string): ArticleSchemaType {
  return kind === "docs" ? "TechArticle" : kind === "blog" ? "BlogPosting" : "Article";
}

export function articleStructuredData(article: {
  title: string;
  description: string;
  path: string;
  language: string;
  type?: ArticleSchemaType;
  publisher?: string;
  publishedAt?: string;
  modifiedAt?: string;
}) {
  const url = canonicalURL(article.path);
  return {
    "@context": "https://schema.org",
    "@type": article.type ?? "Article",
    "@id": `${url}#article`,
    url,
    mainEntityOfPage: { "@type": "WebPage", "@id": url },
    headline: article.title,
    description: article.description,
    inLanguage: article.language,
    ...publisherOf(article.publisher),
    ...(article.publishedAt ? { datePublished: article.publishedAt } : {}),
    ...(article.modifiedAt ? { dateModified: article.modifiedAt } : {})
  };
}

/** Feature, use-case and comparison pages describe the product; they are not articles. */
export function webPageStructuredData(page: { title: string; description: string; path: string; language: string; publisher?: string; modifiedAt?: string }) {
  const url = canonicalURL(page.path);
  return {
    "@context": "https://schema.org",
    "@type": "WebPage",
    "@id": `${url}#webpage`,
    url,
    name: page.title,
    description: page.description,
    inLanguage: page.language,
    ...publisherOf(page.publisher),
    ...(page.modifiedAt ? { dateModified: page.modifiedAt } : {})
  };
}

/** Only questions answered visibly on the same page belong here. */
export function faqStructuredData(items: Array<{ question: string; answer: string }>) {
  return {
    "@context": "https://schema.org",
    "@type": "FAQPage",
    mainEntity: items.map(item => ({ "@type": "Question", name: item.question, acceptedAnswer: { "@type": "Answer", text: item.answer } }))
  };
}

export function serializeStructuredData(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
