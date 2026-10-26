import { apiRequest, type ContentArticle, type ContentSave, type ContentImportResult, type PublishedArticle } from "./api";
import type { PageResult } from "./operations-api";
import { articleOptions, type ArticleOption } from "./navigation-editor";
export function listContent(token: string, page = 1, query = "") { return apiRequest<PageResult<ContentArticle>>(`/admin/articles?${new URLSearchParams({ page: String(page), limit: "20", query })}`, { authToken: token }); }
export function readContent(token: string, id: string) { return apiRequest<ContentArticle>(`/admin/articles/${encodeURIComponent(id)}`, { authToken: token }); }
export function saveContent(token: string, id: string | null, value: ContentSave, key: string) { return apiRequest<ContentArticle>(`/admin/articles${id ? `/${encodeURIComponent(id)}` : ""}`, { authToken: token, method: id ? "PUT" : "POST", json: value, headers: { "Idempotency-Key": key } }); }
export function publishContent(token: string, id: string, version: number, locale: "en" | "zh", action: "publish" | "unpublish", reason: string, key: string) { return apiRequest<ContentArticle>(`/admin/articles/${encodeURIComponent(id)}/publications`, { authToken: token, method: "POST", json: { version, locale, action, reason }, headers: { "Idempotency-Key": key } }); }
export function importContent(token: string, apply = false, reason = "", key = "") { return apiRequest<ContentImportResult>("/admin/articles/import", { authToken: token, method: "POST", json: { apply, reason }, headers: key ? { "Idempotency-Key": key } : {} }); }

async function publishedArticles(locale: "en" | "zh"): Promise<PublishedArticle[]> {
  const items: PublishedArticle[] = [];
  for (let page = 1; ; page++) {
    const result = await apiRequest<PageResult<PublishedArticle>>(`/site/articles?locale=${locale}&limit=100&page=${page}`);
    items.push(...result.items);
    if (items.length >= result.total || result.items.length === 0) return items;
  }
}
/** Published articles in either language, offered when choosing menu entries. */
export async function publishedArticleOptions(): Promise<ArticleOption[]> {
  const [en, zh] = await Promise.all([publishedArticles("en"), publishedArticles("zh")]);
  return articleOptions(en, zh);
}
