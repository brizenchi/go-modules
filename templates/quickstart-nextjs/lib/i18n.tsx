"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";
import {
  DEFAULT_LOCALE,
  type Locale,
  normalizeLocale,
  pickLocaleText
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

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    const stored = typeof window !== "undefined" ? window.localStorage.getItem(STORAGE_KEY) : null;
    const next = normalizeLocale(stored || (typeof navigator !== "undefined" ? navigator.language : DEFAULT_LOCALE));
    setLocaleState(next);
    applyDocumentLanguage(next);
  }, []);

  const value = useMemo<I18nContextValue>(() => ({
    locale,
    setLocale(next: Locale) {
      const normalized = normalizeLocale(next);
      setLocaleState(normalized);
      if (typeof window !== "undefined") {
        window.localStorage.setItem(STORAGE_KEY, normalized);
      }
      applyDocumentLanguage(normalized);
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
