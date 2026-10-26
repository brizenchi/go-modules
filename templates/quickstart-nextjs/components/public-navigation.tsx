"use client";

import { LocalizedLink as Link } from "./localized-link";
import { normalizeNavigation, resolveHeader } from "@/content/navigation";
import { useNavigationIndex } from "./navigation-index";
import { useSitePresentation } from "./site-presentation";
import { useI18n } from "@/lib/i18n";

/** Header chosen in admin (or the defaults). Menus are plain links in the initial
 * HTML; hover and keyboard focus only reveal them. */
export function PublicNavigation({ pathname }: { pathname: string }) {
  const { t, locale } = useI18n();
  const items = resolveHeader(normalizeNavigation(useSitePresentation()?.navigation), useNavigationIndex(), locale, pathname);
  return <nav className="main-nav" aria-label={t({ en: "Primary", zh: "主导航" })}>
    {items.map(item => <div className={`main-nav-item${item.links.length ? " has-menu" : ""}`} key={item.href}>
      <Link href={item.href} className={`main-nav-link${item.current ? " active" : ""}`}
        aria-current={pathname === item.href ? "page" : item.current ? "true" : undefined}>{item.label}</Link>
      {item.links.length ? <div className="main-nav-menu"><div className="main-nav-menu-card"><div className="main-nav-menu-group">
        <ul>{item.links.map(link => <li key={link.href}><Link href={link.href} aria-current={link.current ? "page" : undefined}>{link.label}</Link></li>)}</ul>
        {item.all ? <Link className="main-nav-menu-all" href={item.all.href}>{item.all.label}<span aria-hidden="true"> →</span></Link> : null}
      </div></div></div> : null}
    </div>)}
  </nav>;
}
