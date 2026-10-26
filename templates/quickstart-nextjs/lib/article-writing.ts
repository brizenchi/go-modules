import type { ContentArticle, ContentArticleState, ContentKind } from "./api";
import { articleState } from "./content-filters";
import { contentLocales, type ContentLocale } from "./locale";

/** Languages articles can be written and translated in (the content languages). */
export const writingLanguages = contentLocales;
export type WritingLanguage = ContentLocale;
export const languageNames: Record<WritingLanguage, { en: string; zh: string }> = {
  en: { en: "English", zh: "英文" },
  zh: { en: "Chinese", zh: "中文" },
  ja: { en: "Japanese", zh: "日文" },
  ko: { en: "Korean", zh: "韩文" },
  es: { en: "Spanish", zh: "西班牙文" },
  fr: { en: "French", zh: "法文" },
  de: { en: "German", zh: "德文" }
};

export function translationTargets(source: WritingLanguage): WritingLanguage[] {
  return writingLanguages.filter(language => language !== source);
}

/** AI writing drafts one language and translates it into the others: the open
 * language when it is ticked, otherwise the first ticked one. */
export function writingPlan(selected: readonly WritingLanguage[], current: WritingLanguage): { source: WritingLanguage; targets: WritingLanguage[] } | null {
  const chosen = writingLanguages.filter(language => selected.includes(language));
  const source = chosen.includes(current) ? current : chosen[0];
  return source ? { source, targets: chosen.filter(language => language !== source) } : null;
}

/** Runs task over items, at most limit at a time. Each task handles its own errors. */
export async function mapLimit<T>(items: readonly T[], limit: number, task: (item: T) => Promise<void>): Promise<void> {
  let next = 0;
  const worker = async () => { while (next < items.length) await task(items[next++]); };
  await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
}

const articlePath = /^\/(blog|docs|features|use-cases|compare)\/[a-z0-9]+(?:-[a-z0-9]+)*$/;

/** Mirrors the backend: the section, then a lowercase slug, up to 200 characters. Pages keep fixed paths. */
export function validArticlePath(kind: ContentKind, path: string): boolean {
  return kind === "page" || (path.length <= 200 && articlePath.test(path) && path.startsWith(`/${kind}/`));
}

export type PublishOption = { language: WritingLanguage; state: ContentArticleState; complete: boolean };

/** Every language with its state; publishing needs a title, summary and body. */
export function publishOptions(row: Pick<ContentArticle, "translations" | "published">): PublishOption[] {
  return writingLanguages.map(language => {
    const draft = row.translations[language];
    return { language, state: articleState(row, language), complete: !!(draft?.title.trim() && draft.summary.trim() && draft.body.trim()) };
  });
}

/** A complete language that is new or changed; an unchanged live language has nothing to publish. */
export function canPublish(option: PublishOption): boolean {
  return option.complete && (option.state === "draft" || option.state === "changed");
}

/** Markdown for an uploaded image; brackets and line breaks cannot break out of the alt text. */
export function imageMarkdown(alt: string, url: string): string {
  return `![${cleanAlt(alt)}](${url})`;
}

/** Inserts the image as its own paragraph at a cursor offset. */
export function insertImageAt(body: string, image: string, offset: number): string {
  return placeImage(body, image, offset).body;
}

/** Inserts an image at the cursor and reports where its alt text sits, so the editor can select it. */
export function insertImageAtCursor(body: string, alt: string, url: string, offset: number): { body: string; altStart: number; altEnd: number } {
  const placed = placeImage(body, imageMarkdown(alt, url), offset);
  const altStart = placed.start + 2;
  return { body: placed.body, altStart, altEnd: altStart + cleanAlt(alt).length };
}

const genericName = /^(image|img|photo|picture|pasted image|screenshot|screen shot|截屏|截图|屏幕截图|图片)?[\s\d.:_-]*(at)?[\s\d.:_-]*$/i;

/** A file name as alt text, or "" when it only says the file is a screenshot, photo or number. */
export function defaultAlt(fileName: string): string {
  const name = fileName.replace(/\.[a-z0-9]+$/i, "").replace(/[_-]+/g, " ").replace(/\s+/g, " ").trim();
  return genericName.test(name) ? "" : name;
}

function cleanAlt(alt: string): string {
  return alt.replace(/[[\]\r\n]+/g, " ").replace(/\s+/g, " ").trim();
}

function placeImage(body: string, image: string, offset: number): { body: string; start: number } {
  const position = Math.max(0, Math.min(offset, body.length));
  const before = body.slice(0, position).trimEnd();
  const after = body.slice(position).trimStart();
  const prefix = before ? `${before}\n\n` : "";
  return { body: `${prefix}${image}\n${after ? `\n${after}` : ""}`, start: prefix.length };
}
