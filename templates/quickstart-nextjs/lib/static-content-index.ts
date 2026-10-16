import { documentation } from "../content/docs";
import { articles } from "../content/articles";
import { marketingPages } from "../content/marketing";
import type { DiscoveryEntry } from "./content-discovery";
import type { Locale } from "./locale";

/** Used only when this deployment has no managed content API. */
export function staticContentIndex(locale: Locale): DiscoveryEntry[] {
  return [
    ...documentation.map((guide, position): DiscoveryEntry => ({
      id: `docs:${guide.id}`, path: `/docs/${guide.id}`, kind: "docs", position,
      translation: { title: guide.title[locale], summary: guide.summary[locale] },
      links: guide.links?.map(link => link.href)
    })),
    ...articles.map((article, position): DiscoveryEntry => ({
      id: `blog:${article.slug}`, path: `/blog/${article.slug}`, kind: "blog", position,
      published_at: article.publishedAt,
      translation: { title: article.title[locale], summary: article.summary[locale] }
    })),
    ...marketingPages.flatMap((page, position): DiscoveryEntry[] => page.kind === "examples" ? [] : [{
      id: page.path, path: page.path, kind: page.kind, position,
      translation: { title: page.title[locale], summary: page.description[locale] },
      links: [page.primaryCta.href, ...page.related.map(link => link.href)]
    }])
  ];
}
