import { apiRequest, type ContentLocale, type ContentTranslation, type ContentTranslationJob, type ContentTranslationSettings } from "./api";

export function readContentTranslationSettings(token: string) { return apiRequest<ContentTranslationSettings>("/admin/content-translation-settings", { authToken: token }); }
export function readContentTranslation(token: string, jobId: string) { return apiRequest<ContentTranslationJob>(`/admin/content-translations/${encodeURIComponent(jobId)}`, { authToken: token }); }
export function createContentTranslation(token: string, articleId: string, input: { version: number; source_locale: ContentLocale; target_locale: ContentLocale; reason: string }, key: string) {
 return apiRequest<ContentTranslationJob>(`/admin/articles/${encodeURIComponent(articleId)}/translations`, { authToken: token, method: "POST", json: input, headers: { "Idempotency-Key": key } });
}

export function otherLocale(locale: ContentLocale): ContentLocale { return locale === "zh" ? "en" : "zh"; }

export function translationPhaseLabel(phase: string): { en: string; zh: string } {
 switch (phase) {
  case "queued": return { en: "Waiting", zh: "等待处理" };
  case "translating": return { en: "Translating", zh: "正在翻译" };
  case "reviewing": return { en: "Checking", zh: "正在检查" };
  case "repairing": return { en: "Fixing", zh: "正在修正" };
  case "rechecking": return { en: "Rechecking", zh: "正在复查" };
  default: return { en: "Finished", zh: "已完成" };
 }
}

export function isTerminalTranslationStatus(status: string): boolean { return status === "succeeded" || status === "failed"; }

// An empty source draft has nothing to translate; translating it would burn
// a model call for guaranteed garbage.
export function hasTranslatableContent(value: ContentTranslation): boolean {
 return value.title.trim() !== "" && value.summary.trim() !== "" && value.body.trim() !== "";
}

function lengthFields(value: ContentTranslation): string[] {
 const fields = [value.title, value.summary, value.body];
 if (value.seo_title.trim() !== "") fields.push(value.seo_title);
 if (value.seo_description.trim() !== "") fields.push(value.seo_description);
 return fields;
}

export function countZhCharacters(value: ContentTranslation): number {
 return lengthFields(value).reduce((total, field) => total + Array.from(field).filter(character => !/\s/u.test(character)).length, 0);
}

export function countEnWords(value: ContentTranslation): number {
 return lengthFields(value).reduce((total, field) => total + field.split(/\s+/).filter(Boolean).length, 0);
}

// The backend verdict stays authoritative; this only pre-flights the same
// product limit so the button explains itself before the request.
export function translationLengthState(source: ContentTranslation, locale: ContentLocale, settings: ContentTranslationSettings): { count: number; limit: number; over: boolean } {
 if (locale === "zh") { const count = countZhCharacters(source); return { count, limit: settings.max_zh_characters, over: count > settings.max_zh_characters }; }
 const count = countEnWords(source); return { count, limit: settings.max_en_words, over: count > settings.max_en_words };
}

// An empty check state means the draft was never checked; "stale" covers a
// manual edit after the check, "source-updated" a changed source draft.
export function translationCheckState(meta: { check_status: "passed" | "warning" } | undefined, draftEdited: boolean, sourceUpdated: boolean): "none" | "passed" | "warning" | "stale" | "source-updated" {
 if (!meta) return "none";
 if (draftEdited) return "stale";
 if (sourceUpdated) return "source-updated";
 return meta.check_status;
}

const contentPagePaths = ["/", "/features", "/use-cases", "/compare", "/resources", "/docs", "/blog", "/about", "/updates"];
const contentKinds = ["blog", "docs", "features", "use-cases", "compare"];

// Mirrors the backend kind/path rule so client feedback matches the server
// verdict exactly instead of a generic 400 after submission.
export function isValidContentPath(kind: string, path: string): boolean {
 if (kind === "page") return contentPagePaths.includes(path);
 if (!contentKinds.includes(kind)) return false;
 return new RegExp("^/" + kind + "/[a-z0-9]+(-[a-z0-9]+)*$").test(path);
}
