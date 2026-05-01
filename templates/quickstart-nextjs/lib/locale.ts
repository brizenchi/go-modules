export type Locale = "en" | "zh";

export const DEFAULT_LOCALE: Locale = "en";

export function normalizeLocale(value?: string | null): Locale {
  const raw = (value || "").trim().toLowerCase();
  if (raw.startsWith("zh")) {
    return "zh";
  }
  return DEFAULT_LOCALE;
}

export function pickLocaleText<T>(locale: Locale, values: { en: T; zh: T }): T {
  return locale === "zh" ? values.zh : values.en;
}

export function humanizeSegment(segment: string): string {
  return segment
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
