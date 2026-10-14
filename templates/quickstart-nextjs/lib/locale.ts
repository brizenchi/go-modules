export type Locale = "en" | "zh";

export const DEFAULT_LOCALE: Locale = "en";

const publicRoots = new Set(["pricing", "docs", "blog", "updates", "contact", "privacy", "terms", "features", "use-cases", "compare", "examples", "about", "resources"]);

export function localeFromPath(path: string): Locale {
  return /^\/zh(?=\/|[?#]|$)/.test(path) ? "zh" : "en";
}

export function stripLocalePrefix(path: string): string {
  if (localeFromPath(path) !== "zh") return path;
  const suffix = path.slice(3);
  return suffix.startsWith("/") ? suffix : `/${suffix}`;
}

/** Only public content has translated routes; private workspaces keep their URLs. */
export function isPublicPath(path: string): boolean {
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("\\")) return false;
  const pathname = stripLocalePrefix(path).split(/[?#]/, 1)[0];
  return pathname === "/" || publicRoots.has(pathname.split("/")[1]);
}

export function localizeHref(href: string, locale: Locale): string {
  if (!isPublicPath(href)) return href;
  const base = stripLocalePrefix(href);
  if (locale === "en") return base;
  if (base === "/") return "/zh";
  return `/zh${base.startsWith("/?") || base.startsWith("/#") ? base.slice(1) : base}`;
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
