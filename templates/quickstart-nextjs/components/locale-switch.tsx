"use client";

import { usePathname } from "next/navigation";
import { LanguagePicker } from "@/components/language-picker";
import { useI18n } from "@/lib/i18n";
import { isPublicPath, localizeHref } from "@/lib/locale";

export function LocaleSwitch() {
  const { locale, setLocale, t } = useI18n();
  const path = usePathname() || "/";
  return <LanguagePicker
    locale={locale}
    label={t({ en: "Choose language", zh: "选择语言" })}
    destinations={isPublicPath(path) ? { en: localizeHref(path, "en"), zh: localizeHref(path, "zh") } : undefined}
    onSelect={setLocale}
  />;
}
