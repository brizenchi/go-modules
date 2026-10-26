/** Interface languages: every label, button and code-owned page exists in these. */
export type Locale = "en" | "zh";

export const DEFAULT_LOCALE: Locale = "en";

/** Languages articles are published in. Pages in a language without an interface
 * translation show the article in that language and the interface in English. */
export const contentLocales = ["en", "zh", "ja", "ko", "es", "fr", "de"] as const;
export type ContentLocale = typeof contentLocales[number];

/** BCP 47 tags for html lang and hreflang. */
export const languageTags: Record<ContentLocale, string> = { en: "en", zh: "zh-CN", ja: "ja", ko: "ko", es: "es", fr: "fr", de: "de" };
export const openGraphLocales: Record<ContentLocale, string> = { en: "en_US", zh: "zh_CN", ja: "ja_JP", ko: "ko_KR", es: "es_ES", fr: "fr_FR", de: "de_DE" };
/** Each language named in itself, for language choices. */
export const nativeLanguageNames: Record<ContentLocale, string> = { en: "English", zh: "中文", ja: "日本語", ko: "한국어", es: "Español", fr: "Français", de: "Deutsch" };

const localePrefix = new RegExp(`^/(${contentLocales.filter(locale => locale !== "en").join("|")})(?=/|[?#]|$)`);

export function isContentLocale(value: unknown): value is ContentLocale {
  return typeof value === "string" && (contentLocales as readonly string[]).includes(value);
}

export function isInterfaceLocale(value: ContentLocale): value is Locale {
  return value === "en" || value === "zh";
}

/** The interface language that accompanies an article language. */
export function interfaceLocale(locale: ContentLocale): Locale {
  return locale === "zh" ? "zh" : "en";
}

export function contentLocaleFromPath(path: string): ContentLocale {
  return (localePrefix.exec(path)?.[1] as ContentLocale | undefined) || "en";
}

const publicRoots = new Set(["pricing", "docs", "blog", "updates", "contact", "privacy", "terms", "features", "use-cases", "compare", "examples", "about", "resources"]);

export function localeFromPath(path: string): Locale {
  return interfaceLocale(contentLocaleFromPath(path));
}

export function stripLocalePrefix(path: string): string {
  const prefix = localePrefix.exec(path);
  if (!prefix) return path;
  const suffix = path.slice(prefix[0].length);
  return suffix.startsWith("/") ? suffix : `/${suffix}`;
}

/** Only public content has translated routes; private workspaces keep their URLs. */
export function isPublicPath(path: string): boolean {
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("\\")) return false;
  const pathname = stripLocalePrefix(path).split(/[?#]/, 1)[0];
  return pathname === "/" || publicRoots.has(pathname.split("/")[1]);
}

/** A public URL in an interface language. A link already in another article
 * language (/ja/...) is kept, so article pages can point to each other. */
export function localizeHref(href: string, locale: Locale): string {
  if (!isPublicPath(href)) return href;
  if (!isInterfaceLocale(contentLocaleFromPath(href))) return href;
  return contentHref(href, locale);
}

/** A public URL in any article language. */
export function contentHref(href: string, locale: ContentLocale): string {
  if (!isPublicPath(href)) return href;
  const base = stripLocalePrefix(href);
  if (locale === "en") return base;
  if (base === "/") return `/${locale}`;
  return `/${locale}${base.startsWith("/?") || base.startsWith("/#") ? base.slice(1) : base}`;
}

/** Persisted preferences are for workspaces, never for a public page's indexed language. */
export function resolvePageLocale(path: string, preference?: string | null): Locale {
  return isPublicPath(path) ? localeFromPath(path) : normalizeLocale(preference);
}

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
