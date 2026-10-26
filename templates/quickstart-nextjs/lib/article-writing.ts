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

/** Markdown for an uploaded image; brackets and line breaks cannot break out of the alt text. */
export function imageMarkdown(alt: string, url: string): string {
  return `![${alt.replace(/[[\]\r\n]+/g, " ").replace(/\s+/g, " ").trim()}](${url})`;
}

/** Puts the image right below the "## section" heading it was suggested for, or at the end. */
export function insertImage(body: string, image: string, section: string): string {
  const lines = body.split("\n");
  const heading = section.trim() ? lines.findIndex(line => /^#{2,6}\s+/.test(line) && line.replace(/^#{2,6}\s+/, "").trim() === section.trim()) : -1;
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
