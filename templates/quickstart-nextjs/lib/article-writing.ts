import type { ArticleImageSuggestion } from "./api";
import { contentLocales, isContentLocale, type ContentLocale } from "./locale";

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

/** One suggested image in every language it was translated into. */
export type ImagePlanItem = Partial<Record<WritingLanguage, ArticleImageSuggestion>>;
export type ImagePlan = { source: WritingLanguage; items: ImagePlanItem[] };

/** Pairs the source suggestions with their translations; a language whose list does not match is left out. */
export function imagePlan(source: WritingLanguage, images: readonly ArticleImageSuggestion[], translated: Partial<Record<WritingLanguage, readonly ArticleImageSuggestion[]>>): ImagePlan {
  const matching = (Object.entries(translated) as Array<[WritingLanguage, readonly ArticleImageSuggestion[] | undefined]>).filter(([language, list]) => language !== source && list?.length === images.length);
  return { source, items: images.map((image, index) => ({ [source]: image, ...Object.fromEntries(matching.map(([language, list]) => [language, list?.[index]])) })) };
}

/** Inserts one uploaded image into each language's body, under that language's heading for it. */
export function insertPlannedImage(bodies: Partial<Record<WritingLanguage, string>>, item: ImagePlanItem, source: WritingLanguage, url: string): Partial<Record<WritingLanguage, string>> {
  const position = headingIndex(bodies[source] ?? "", item[source]?.section ?? "");
  const next: Partial<Record<WritingLanguage, string>> = {};
  for (const language of writingLanguages) {
    const image = item[language]; const body = bodies[language];
    if (image && body !== undefined) next[language] = insertImage(body, imageMarkdown(image.alt, url), image.section, position);
  }
  return next;
}

/** Suggestions kept in session storage while an article is edited; anything malformed is ignored. */
export function parseImagePlan(raw: string | null): ImagePlan | null {
  try {
    const value: unknown = raw ? JSON.parse(raw) : null;
    if (!value || typeof value !== "object" || Array.isArray(value)) return null;
    const { source, items } = value as { source?: unknown; items?: unknown };
    const suggestion = (image: unknown) => !!image && typeof image === "object" && ["section", "alt", "description"].every(key => typeof (image as Record<string, unknown>)[key] === "string");
    if (!isContentLocale(source) || !Array.isArray(items)) return null;
    const valid = items.every(item => item && typeof item === "object" && suggestion((item as Record<string, unknown>)[source]) && Object.entries(item as Record<string, unknown>).every(([language, image]) => isContentLocale(language) && suggestion(image)));
    return valid ? { source, items: items as ImagePlanItem[] } : null;
  } catch {
    return null;
  }
}

const headingLine = /^#{2,6}\s+/;
function headingIndex(body: string, section: string): number {
  return section.trim() ? body.split("\n").filter(line => headingLine.test(line)).findIndex(line => line.replace(headingLine, "").trim() === section.trim()) : -1;
}

/** Markdown for an uploaded image; brackets and line breaks cannot break out of the alt text. */
export function imageMarkdown(alt: string, url: string): string {
  return `![${alt.replace(/[[\]\r\n]+/g, " ").replace(/\s+/g, " ").trim()}](${url})`;
}

/** Puts the image right below the "## section" heading it was suggested for; otherwise below the
 * heading at position (counting headings from 0) when given, or at the end. */
export function insertImage(body: string, image: string, section: string, position = -1): string {
  const lines = body.split("\n");
  const headings = lines.flatMap((line, index) => headingLine.test(line) ? [index] : []);
  const named = section.trim() ? headings.find(index => lines[index].replace(headingLine, "").trim() === section.trim()) : undefined;
  const heading = named ?? (position >= 0 ? headings[position] : undefined) ?? -1;
  if (heading < 0) return insertImageAt(body, image, body.length);
  lines.splice(heading + 1, 0, "", image);
  return lines.join("\n");
}

/** Inserts the image as its own paragraph at a cursor offset. */
export function insertImageAt(body: string, image: string, offset: number): string {
  const position = Math.max(0, Math.min(offset, body.length));
  const before = body.slice(0, position).trimEnd();
  const after = body.slice(position).trimStart();
  return `${before ? `${before}\n\n` : ""}${image}\n${after ? `\n${after}` : ""}`;
}
