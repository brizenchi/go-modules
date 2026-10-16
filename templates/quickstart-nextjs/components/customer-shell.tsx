"use client";

import { useEffect, useState, type ReactNode } from "react";
import { NavigationLabel } from "./navigation-label";
import { LocalizedLink as Link } from "@/components/localized-link";
import { Breadcrumbs } from "./breadcrumbs";
import { customerBreadcrumbs, customerFooterLinks, siteLinks } from "@/content/navigation";
import { InterfaceIcon } from "@/components/interface-icon";
import { ResponsiveDialog } from "@/components/responsive-dialog";
import { workspaceNavItems } from "@/lib/workspace";
import { useI18n } from "@/lib/i18n";
import styles from "./customer-shell.module.css";

type CustomerShellProps = {
  brand: string;
  activeHref: string;
  accountMenu: ReactNode;
  localeControl: ReactNode;
  children: ReactNode;
};

export function CustomerShell({ brand, activeHref, accountMenu, localeControl, children }: CustomerShellProps) {
  const { t, locale } = useI18n();
  const [navOpen, setNavOpen] = useState(false);
  useEffect(() => {
    const desktop = window.matchMedia("(min-width: 761px)");
    const closeOnDesktop = () => { if (desktop.matches) setNavOpen(false); };
    desktop.addEventListener("change", closeOnDesktop);
    return () => desktop.removeEventListener("change", closeOnDesktop);
  }, []);

  useEffect(() => { setNavOpen(false); }, [activeHref]);

  function navigation() {
    return <nav className={styles.navigation} aria-label={t({ en: "Account navigation", zh: "账户导航" })}>
      {workspaceNavItems.map(item => <Link key={item.href} href={item.href}
        className={`${styles.navItem}${activeHref === item.href ? ` ${styles.active}` : ""}`}
        aria-current={activeHref === item.href ? "page" : undefined}
        onNavigate={() => { if (activeHref === item.href) setNavOpen(false); }}>
        <InterfaceIcon name={item.icon} /><NavigationLabel label={t(item.label)} className={styles.navLinkContent} statusClassName={styles.navigationStatus} />
      </Link>)}
    </nav>;
  }

  return <div className={styles.shell}>
    <a className="skip-content" href="#main-content">{t({ en: "Skip to content", zh: "跳到正文" })}</a>
    <header className={styles.header}>
      <div className={styles.headerInner}>
        <Link className={styles.brand} href="/" aria-label={`${brand} · ${t({ en: "Home", zh: "首页" })}`}>
          <span className={styles.brandMark}><svg viewBox="0 0 28 28" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><path d="M6 8.5 14 4l8 4.5v9L14 22l-8-4.5v-9Z" /><path d="m10 11 4-2.3 4 2.3v4.5L14 18l-4-2.5z" /></svg></span>
          <strong>{brand}</strong>
        </Link>
        <div className={styles.tools}>{localeControl}{accountMenu}</div>
      </div>
    </header>
    <div className={styles.container}>
      <div className={styles.breadcrumb}>
        <button type="button" className={styles.mobileTrigger} aria-label={t({ en: "Open account navigation", zh: "打开账户导航" })} aria-haspopup="dialog" aria-expanded={navOpen} onClick={() => setNavOpen(true)}><InterfaceIcon name="menu" /></button>
        <Breadcrumbs items={customerBreadcrumbs(activeHref, locale)} path={activeHref} label={t({ en: "Breadcrumb", zh: "面包屑导航" })} />
      </div>
      <div className={styles.grid}>
        <aside className={styles.sidebar}>
          <p className={styles.navLabel}>{t({ en: "Account", zh: "个人账户" })}</p>
          {navigation()}
          <Link className={styles.support} href={siteLinks.help.href}>{t(siteLinks.help.label)}<span aria-hidden="true">↗</span></Link>
        </aside>
        <main id="main-content" tabIndex={-1} className={`workspace-page ${styles.content}`}>
          {children}
        </main>
      </div>
    </div>
    <footer className={styles.footer}><span>© {new Date().getFullYear()} {brand}</span><nav aria-label={t({ en: "Account support and policies", zh: "账户帮助与条款" })}>{customerFooterLinks.map(item => <Link key={item.href} href={item.href}>{t(item.label)}</Link>)}</nav></footer>
    <ResponsiveDialog open={navOpen} onClose={() => setNavOpen(false)} title={t({ en: "My account", zh: "账户中心" })}
      closeLabel={t({ en: "Close navigation", zh: "关闭导航" })} presentation="side">
      {(dismiss) => <>{navigation()}<Link className={styles.support} href={siteLinks.help.href} onClick={dismiss}>{t(siteLinks.help.label)}<span aria-hidden="true">↗</span></Link></>}
    </ResponsiveDialog>
  </div>;
}

/** Page-owned content inside the persistent customer route layout. */
export function CustomerPage({ title, description, children, actions, context, contextTitle }: {
  title: string;
  description: string;
  children: ReactNode;
  actions?: ReactNode;
  context?: ReactNode;
  contextTitle?: string;
}) {
  return <>
    <div className={styles.pageHeader}>
      <div className={styles.heading}><h1>{title}</h1></div>
      <p>{description}</p>
      {actions ? <div className={styles.actions}>{actions}</div> : null}
    </div>
    {context ? <aside className={styles.context}>{contextTitle ? <h2>{contextTitle}</h2> : null}{context}</aside> : null}
    {children}
  </>;
}
