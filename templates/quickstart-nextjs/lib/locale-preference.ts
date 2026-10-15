import { DEFAULT_LOCALE, normalizeLocale, type Locale } from "./locale";

export const LOCALE_STORAGE_KEY = "go-modules.quickstart-nextjs.locale";

type LocaleStorage = Pick<Storage, "getItem" | "setItem">;

export function browserLocaleStorage(): LocaleStorage | null {
  try { return typeof window === "undefined" ? null : window.localStorage; }
  catch { return null; }
}

/** Browser/OS language never overrides the English default or a saved choice. */
export function readLocalePreference(storage: LocaleStorage | null, fallback: Locale = DEFAULT_LOCALE): Locale {
  try {
    const saved = storage?.getItem(LOCALE_STORAGE_KEY);
    return saved ? normalizeLocale(saved) : fallback;
  } catch { return fallback; }
}

export function saveLocalePreference(locale: Locale, storage: LocaleStorage | null): void {
  try { storage?.setItem(LOCALE_STORAGE_KEY, normalizeLocale(locale)); }
  catch { /* Keep the in-memory choice when the browser blocks persistence. */ }
}
