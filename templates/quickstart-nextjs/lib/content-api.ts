import { apiRequest, type ArticleBrief, type ArticleDraft, type ArticleTranslations, type ContentArticle, type ContentSave, type ContentImportResult, type ContentTranslation, type PublishedArticle, type SiteImage } from "./api";
import type { PageResult } from "./operations-api";
import { articleOptions, type ArticleOption } from "./navigation-editor";
import { contentLocales, type ContentLocale } from "./locale";
export function listContent(token: string, page = 1, query = "") { return apiRequest<PageResult<ContentArticle>>(`/admin/articles?${new URLSearchParams({ page: String(page), limit: "20", query })}`, { authToken: token }); }
export function readContent(token: string, id: string) { return apiRequest<ContentArticle>(`/admin/articles/${encodeURIComponent(id)}`, { authToken: token }); }
export function saveContent(token: string, id: string | null, value: ContentSave, key: string) { return apiRequest<ContentArticle>(`/admin/articles${id ? `/${encodeURIComponent(id)}` : ""}`, { authToken: token, method: id ? "PUT" : "POST", json: value, headers: { "Idempotency-Key": key } }); }
export function publishContent(token: string, id: string, version: number, locale: ContentLocale, action: "publish" | "unpublish", reason: string, key: string) { return apiRequest<ContentArticle>(`/admin/articles/${encodeURIComponent(id)}/publications`, { authToken: token, method: "POST", json: { version, locale, action, reason }, headers: { "Idempotency-Key": key } }); }
export function importContent(token: string, apply = false, reason = "", key = "") { return apiRequest<ContentImportResult>("/admin/articles/import", { authToken: token, method: "POST", json: { apply, reason }, headers: key ? { "Idempotency-Key": key } : {} }); }

async function publishedArticles(locale: ContentLocale): Promise<PublishedArticle[]> {
  const items: PublishedArticle[] = [];
  for (let page = 1; ; page++) {
    const result = await apiRequest<PageResult<PublishedArticle>>(`/site/articles?locale=${locale}&limit=100&page=${page}`);
    items.push(...result.items);
    if (items.length >= result.total || result.items.length === 0) return items;
  }
}
/** Published articles in either language, offered when choosing menu entries. */
export async function publishedArticleOptions(): Promise<ArticleOption[]> {
  return articleOptions((await Promise.all(contentLocales.map(publishedArticles))).flat());
}
/** Asks the configured AI model for an article; the editor saves it as a draft. */
export function draftArticle(token: string, brief: ArticleBrief) {
  return apiRequest<ArticleDraft>("/admin/articles/drafts", { authToken: token, method: "POST", json: brief });
}
/** Translates one language's draft; links and images are kept. */
export function translateArticle(token: string, from: ContentLocale, to: ContentLocale[], translation: ContentTranslation) {
  return apiRequest<ArticleTranslations>("/admin/articles/translations", { authToken: token, method: "POST", json: { from, to, translation } });
}
export function uploadSiteImage(token: string, file: File) {
  const body = new FormData();
  body.append("file", file);
  return apiRequest<SiteImage>("/admin/site-images", { authToken: token, method: "POST", body });
}
