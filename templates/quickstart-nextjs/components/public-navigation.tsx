"use client";

import { LocalizedLink as Link } from "./localized-link";
import { activePublicNavigationHref, publicNavigation } from "@/content/navigation";
import { useI18n } from "@/lib/i18n";

export function PublicNavigation({ pathname }: { pathname: string }) {
  const { t } = useI18n();
  const active = activePublicNavigationHref(pathname);
  return <nav className="main-nav" aria-label={t({ en: "Primary", zh: "主导航" })}>
    {publicNavigation.map(item => <Link key={item.href} href={item.href}
      className={`main-nav-link${active === item.href ? " active" : ""}`}
      aria-current={pathname === item.href ? "page" : active === item.href ? "true" : undefined}>{t(item.label)}</Link>)}
  </nav>;
}
