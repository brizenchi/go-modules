import type { ContentKind, PublishedArticle } from "./api";

export type GuideIndexKind = Exclude<ContentKind, "page"> | "resources";
export type GuideCategory = "all" | "docs" | "blog";
export type GuideIndexEntry = Pick<PublishedArticle, "id" | "path" | "kind"> & { translation: Pick<PublishedArticle["translation"], "title" | "summary"> };

/** The learning archive does not repeat product capability/marketing pages. */
export function filterGuideIndex<T extends GuideIndexEntry>(entries: readonly T[], kind: GuideIndexKind, category: GuideCategory, query: string): T[] {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  return entries.filter(entry => {
    if (kind === "resources") {
      if (entry.kind !== "docs" && entry.kind !== "blog") return false;
      if (category !== "all" && entry.kind !== category) return false;
    } else if (entry.kind !== kind) return false;
    const text = `${entry.translation.title} ${entry.translation.summary}`.toLowerCase();
    return terms.every(term => text.includes(term));
  });
}
