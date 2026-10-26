"use client";

import { usePathname } from "next/navigation";
import { useArticleLanguages } from "./article-language-context";
import { LanguagePicker } from "@/components/language-picker";
import { isEditorialPath } from "@/lib/content-discovery";
import { useI18n } from "@/lib/i18n";
import { contentHref, contentLocaleFromPath, interfaceLocale, isInterfaceLocale, isPublicPath, stripLocalePrefix, type ContentLocale } from "@/lib/locale";

/** Public pages offer the languages they are published in; only articles have
 * routes in languages without an interface translation. */
export function LocaleSwitch() {
  const { locale, setLocale, t } = useI18n();
  const path = usePathname() || "/";
  const languages = useArticleLanguages();
  const publicPage = isPublicPath(path);
  const article = isEditorialPath(stripLocalePrefix(path.split(/[?#]/, 1)[0]));
  const available = (languages || ["en", "zh"] as ContentLocale[]).filter(language => article || isInterfaceLocale(language));
  return <LanguagePicker
    locale={publicPage ? contentLocaleFromPath(path) : locale}
    label={t({ en: "Choose language", zh: "选择语言" })}
    destinations={publicPage ? Object.fromEntries(available.map(language => [language, contentHref(path, language)])) : undefined}
    onSelect={value => setLocale(interfaceLocale(value))}
  />;
}
