"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";
import { usePathname } from "next/navigation";
import {
  contentLocaleFromPath,
  DEFAULT_LOCALE,
  isPublicPath,
  languageTags,
  localeFromPath,
  type Locale,
  normalizeLocale,
  pickLocaleText,
  resolvePageLocale
} from "./locale";
import { browserLocaleStorage, LOCALE_STORAGE_KEY, readLocalePreference, saveLocalePreference } from "./locale-preference";

type LocaleText<T> = {
  en: T;
  zh: T;
};

type I18nContextValue = {
  locale: Locale;
  setLocale: (next: Locale) => void;
  t: <T>(values: LocaleText<T>) => T;
};

const I18nContext = createContext<I18nContextValue | null>(null);

/** A public page keeps the language of its content; private pages follow the interface. */
function applyDocumentLanguage(locale: Locale, pathname: string, publicPage: boolean): void {
  if (typeof document === "undefined") {
    return;
  }
  document.documentElement.lang = languageTags[publicPage ? contentLocaleFromPath(pathname) : locale];
}

export function LocaleProvider({ children, initialLocale = DEFAULT_LOCALE }: { children: React.ReactNode; initialLocale?: Locale }) {
  const pathname = usePathname() || (initialLocale === "zh" ? "/zh" : "/");
  const publicPage = isPublicPath(pathname);
  const [preference, setLocaleState] = useState<Locale>(initialLocale);
  const locale = resolvePageLocale(pathname, preference);

  useEffect(() => {
    const storage = browserLocaleStorage();
    if (publicPage) {
      // Indexed public URLs retain their own language and carry it into the
      // account/admin area, including visits through a direct translated link.
      const current = localeFromPath(pathname);
      saveLocalePreference(current, storage);
      setLocaleState(current);
    } else {
      setLocaleState(current => readLocalePreference(storage, current));
    }

    const sync = (event: StorageEvent) => {
      if (!publicPage && (event.key === LOCALE_STORAGE_KEY || event.key === null)) {
        setLocaleState(readLocalePreference(browserLocaleStorage()));
      }
    };
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, [pathname, publicPage]);

  useEffect(() => { applyDocumentLanguage(locale, pathname, publicPage); }, [locale, pathname, publicPage]);

  const value = useMemo<I18nContextValue>(() => ({
    locale,
    setLocale(next: Locale) {
      const normalized = normalizeLocale(next);
      saveLocalePreference(normalized, browserLocaleStorage());
      setLocaleState(normalized);
    },
    t<T>(values: LocaleText<T>): T {
      return pickLocaleText(locale, values);
    }
  }), [locale]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const value = useContext(I18nContext);
  if (!value) {
    throw new Error("useI18n must be used within LocaleProvider");
  }
  return value;
}
