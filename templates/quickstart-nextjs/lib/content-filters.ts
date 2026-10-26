import type { ContentArticle, ContentArticleState, ContentListFilters, ContentTranslation } from "./api";
import { contentLocales, type ContentLocale } from "./locale";

export const noContentFilters: ContentListFilters = { query: "", kind: "", locale: "", status: "" };

const fields: Array<keyof ContentTranslation> = ["title", "summary", "body", "seo_title", "seo_description"];

/** Mirrors content.Article.State in the backend; a draft without a title or body is not a version. */
export function articleState(row: Pick<ContentArticle, "translations" | "published">, locale: ContentLocale): ContentArticleState {
  const live = row.published[locale];
  const draft = row.translations[locale];
  if (live && draft && fields.some(field => draft[field] !== live[field])) return "changed";
  if (live) return "published";
  if (draft && (draft.title.trim() || draft.body.trim())) return "draft";
  return "missing";
}

/** Languages the list shows for an article: always EN and Chinese, others once written or published. */
export function listedLanguages(row: Pick<ContentArticle, "translations" | "published">): ContentLocale[] {
  return contentLocales.filter(locale => locale === "en" || locale === "zh" || articleState(row, locale) !== "missing");
}

export function contentListQuery(page: number, filters: ContentListFilters): URLSearchParams {
  const params = new URLSearchParams({ page: String(page), limit: "20" });
  for (const key of ["query", "kind", "locale", "status"] as const) if (filters[key]) params.set(key, filters[key]);
  return params;
}

/** "Not written" is only meaningful for one language, so clearing the language clears it. */
export function withLanguage(filters: ContentListFilters, locale: ContentListFilters["locale"]): ContentListFilters {
  return { ...filters, locale, status: !locale && filters.status === "missing" ? "" : filters.status };
}
