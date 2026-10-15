"use client";

import { usePathname } from "next/navigation";
import { useArticleLanguages } from "./article-language-context";
import { LanguagePicker } from "@/components/language-picker";
import { useI18n } from "@/lib/i18n";
import { isPublicPath, localizeHref } from "@/lib/locale";

export function LocaleSwitch() {
  const { locale, setLocale, t } = useI18n();
  const path = usePathname() || "/";
 const languages=useArticleLanguages();
  return <LanguagePicker
    locale={locale}
    label={t({ en: "Choose language", zh: "选择语言" })}
    destinations={isPublicPath(path) ? Object.fromEntries((languages || ["en", "zh"]).map(language => [language, localizeHref(path, language as "en" | "zh")])) : undefined}
    onSelect={setLocale}
  />;
}
