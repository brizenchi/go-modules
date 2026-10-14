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
  DEFAULT_LOCALE,
  isPublicPath,
  localizeHref,
  type Locale,
  normalizeLocale,
  pickLocaleText,
  resolvePageLocale
} from "./locale";

const STORAGE_KEY = "go-modules.quickstart-nextjs.locale";

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

function applyDocumentLanguage(locale: Locale): void {
  if (typeof document === "undefined") {
    return;
  }
  document.documentElement.lang = locale === "zh" ? "zh-CN" : "en";
}

export function LocaleProvider({ children, initialLocale = DEFAULT_LOCALE }: { children: React.ReactNode; initialLocale?: Locale }) {
  const pathname = usePathname() || (initialLocale === "zh" ? "/zh" : "/");
  const publicPage = isPublicPath(pathname);
  const [preference, setLocaleState] = useState<Locale>(initialLocale);
  const locale = resolvePageLocale(pathname, preference);

  useEffect(() => {
    if (publicPage) {
      applyDocumentLanguage(resolvePageLocale(pathname));
      return;
    }
    let stored: string | null = null;
    try { stored = window.localStorage.getItem(STORAGE_KEY); } catch { /* A browser can disable storage. */ }
    const next = normalizeLocale(stored || (typeof navigator !== "undefined" ? navigator.language : DEFAULT_LOCALE));
    setLocaleState(next);
    applyDocumentLanguage(next);
  }, [pathname, publicPage]);

  const value = useMemo<I18nContextValue>(() => ({
    locale,
    setLocale(next: Locale) {
      const normalized = normalizeLocale(next);
      if (typeof window !== "undefined") {
        try { window.localStorage.setItem(STORAGE_KEY, normalized); } catch { /* Language selection remains usable without storage. */ }
        if (publicPage) {
          const destination = localizeHref(`${window.location.pathname}${window.location.search}${window.location.hash}`, normalized);
          if (destination !== `${window.location.pathname}${window.location.search}${window.location.hash}`) window.location.assign(destination);
          return;
        }
      }
      setLocaleState(normalized);
      applyDocumentLanguage(normalized);
    },
    t<T>(values: LocaleText<T>): T {
      return pickLocaleText(locale, values);
    }
  }), [locale, publicPage]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const value = useContext(I18nContext);
  if (!value) {
    throw new Error("useI18n must be used within LocaleProvider");
  }
  return value;
}
