"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";
import type { Locale } from "../lib/locale";

export const languageOptions: { locale: Locale; label: string; language: string }[] = [
  { locale: "en", label: "EN", language: "en" },
  { locale: "zh", label: "中文", language: "zh-CN" }
];

type Props = {
  locale: Locale;
  label: string;
  destinations?: Partial<Record<Locale, string>>;
  onSelect: (locale: Locale) => void;
};

/** Public options remain links in initial HTML, even while the disclosure is closed. */
export function LanguagePicker({ locale, label, destinations, onSelect }: Props) {
  const disclosure = useRef<HTMLDetailsElement>(null);
  const trigger = useRef<HTMLElement>(null);
  const current = languageOptions.find(option => option.locale === locale)!;

  useEffect(() => {
    const dismissOutside = (event: PointerEvent) => {
      if (event.target instanceof Node && !disclosure.current?.contains(event.target) && disclosure.current) disclosure.current.open = false;
    };
    document.addEventListener("pointerdown", dismissOutside);
    return () => document.removeEventListener("pointerdown", dismissOutside);
  }, []);

  function select(value: Locale) {
    onSelect(value);
    if (disclosure.current) disclosure.current.open = false;
    trigger.current?.focus();
  }

  return (
    <details className="language-picker" ref={disclosure} onKeyDown={event => {
      if (event.key === "Escape" && disclosure.current?.open) {
        event.preventDefault();
        disclosure.current.open = false;
        trigger.current?.focus();
      }
    }} onBlur={event => {
      if (!event.currentTarget.contains(event.relatedTarget) && disclosure.current) disclosure.current.open = false;
    }}>
      <summary ref={trigger} className="language-trigger" aria-label={`${label}: ${current.label}`}>
        <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3c-5 5-5 13 0 18M12 3c5 5 5 13 0 18" /></svg>
        <span lang={current.language}>{current.label}</span>
        <svg className="language-chevron" viewBox="0 0 24 24" aria-hidden="true"><path d="m8 10 4 4 4-4" /></svg>
      </summary>
      <nav className="language-options" aria-label={label}>
        {languageOptions.filter(option => !destinations || destinations[option.locale]).map(option => {
          const content = <><span>{option.label}</span><span aria-hidden="true">{option.locale === locale ? "✓" : ""}</span></>;
          return destinations ? (
            <Link key={option.locale} href={destinations[option.locale]!} hrefLang={option.language} lang={option.language} aria-current={option.locale === locale ? "page" : undefined} onClick={() => select(option.locale)}>{content}</Link>
          ) : (
            <button key={option.locale} type="button" lang={option.language} aria-pressed={option.locale === locale} onClick={() => select(option.locale)}>{content}</button>
          );
        })}
      </nav>
    </details>
  );
}
