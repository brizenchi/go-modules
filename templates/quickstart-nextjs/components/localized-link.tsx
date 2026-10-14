"use client";

import Link from "next/link";
import type { ComponentProps } from "react";
import { useI18n } from "@/lib/i18n";
import { localizeHref } from "@/lib/locale";

export function LocalizedLink({ href, ...props }: ComponentProps<typeof Link>) {
  const { locale } = useI18n();
  const destination = typeof href === "string" ? localizeHref(href, locale) : { ...href, pathname: href.pathname ? localizeHref(href.pathname, locale) : href.pathname };
  return <Link {...props} href={destination} />;
}
