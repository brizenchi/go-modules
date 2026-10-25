"use client";

import { LocalizedLink as Link } from "./localized-link";
import { activePublicNavigationHref, headerMenu, visibleHeaderNavigation } from "@/content/navigation";
import { useNavigationIndex, usePublishedDirectories } from "./navigation-index";
import { useI18n } from "@/lib/i18n";

/** Menus are plain links in the initial HTML; hover and keyboard focus only reveal them. */
export function PublicNavigation({ pathname }: { pathname: string }) {
  const { t, locale } = useI18n();
  const entries = useNavigationIndex()?.entries ?? [];
  const items = visibleHeaderNavigation(usePublishedDirectories());
  const active = activePublicNavigationHref(pathname);
  return <nav className="main-nav" aria-label={t({ en: "Primary", zh: "主导航" })}>
    {items.map(item => {
      const href = item.link.href;
      const current = active === item.section;
      const groups = headerMenu(item, entries, locale);
      const trigger = <Link href={href}
        className={`main-nav-link${current ? " active" : ""}`}
        aria-current={pathname === href ? "page" : current ? "true" : undefined}>{t(item.label || item.link.label)}</Link>;
      return <div className={`main-nav-item${groups.length ? " has-menu" : ""}`} key={href}>
        {trigger}
        {groups.length ? <div className="main-nav-menu"><div className="main-nav-menu-card">
          {groups.map(group => <div className="main-nav-menu-group" key={group.title}>
            <span className="main-nav-menu-title">{group.title}</span>
            <ul>{group.links.map(link => <li key={link.href}><Link href={link.href} aria-current={pathname === link.href ? "page" : undefined}>{link.label}</Link></li>)}</ul>
            {group.all ? <Link className="main-nav-menu-all" href={group.all.href}>{group.all.label}<span aria-hidden="true"> →</span></Link> : null}
          </div>)}
        </div></div> : null}
      </div>;
    })}
  </nav>;
}
