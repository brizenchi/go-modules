"use client";

import { usePathname } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { LocalizedLink as Link } from "./localized-link";
import { ApiError, logout } from "@/lib/api";
import { activeAdminSection, adminPrimarySection, adminNavigationGroups, adminSectionHref, adminSectionIcons, adminSectionNames } from "@/lib/admin-navigation";
import { getOperatorAuthorization } from "@/lib/operations-api";
import { useSitePresentation } from "./site-presentation";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import { interfaceLocale } from "@/lib/locale";
import { ConsoleError, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { LanguagePicker } from "./language-picker";
import { AdminSignInPanel } from "./admin-sign-in-panel";
import { InterfaceIcon } from "./interface-icon";
import { NavigationLabel } from "./navigation-label";
import { PendingLabel } from "./pending-label";
import { ResponsiveDialog } from "./responsive-dialog";
import { BrandMark } from "./brand-mark";
import styles from "./admin-shell.module.css";

async function verifyAdminAccess(token: string) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 8000);
  try { return await getOperatorAuthorization(token, controller.signal); } finally { clearTimeout(timer); }
}

function AdminIdentity({ email, name, busy, onSignOut }: { email: string; name: string; busy: boolean; onSignOut: () => void }) {
  const { t } = useI18n();
  const pathname = usePathname();
  const menu = useRef<HTMLDetailsElement>(null);
  useEffect(() => { if (menu.current) menu.current.open = false; }, [pathname, email]);
  useEffect(() => {
    const outside = (event: PointerEvent) => {
      if (menu.current?.open && event.target instanceof Node && !menu.current.contains(event.target)) menu.current.open = false;
    };
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, []);
  return <details ref={menu} className={styles.identityMenu} onBlur={(event) => {
    if (!event.currentTarget.contains(event.relatedTarget)) event.currentTarget.open = false;
  }} onKeyDown={(event) => {
    if (event.key === "Escape") { event.preventDefault(); event.currentTarget.open = false; event.currentTarget.querySelector("summary")?.focus(); }
  }}>
    <summary aria-label={t({ en: "Administrator account", zh: "管理员账户" })}>
      <span className={styles.avatar} aria-hidden="true">{(name || email).slice(0, 2).toUpperCase()}</span>
      <span className={styles.identity}>{name || t({ en: "Administrator", zh: "管理员" })}</span><InterfaceIcon className={styles.chevron} name="chevron" />
    </summary>
    <div className={styles.identityPopover}>
      <div className={styles.identityDetails}><strong>{t({ en: "Administrator", zh: "管理员" })}</strong><span>{email}</span></div>
      <Link href="/"><InterfaceIcon name="arrow" />{t({ en: "View website", zh: "访问网站" })}</Link>
      <button type="button" onClick={onSignOut} disabled={busy}><InterfaceIcon name="logout" />{busy ? <PendingLabel>{t({ en: "Signing out…", zh: "正在退出…" })}</PendingLabel> : t({ en: "Sign out", zh: "退出登录" })}</button>
    </div>
  </details>;
}

// Authorization remains in this persistent administrator layout and the API.
export function AdminShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
 const presentation=useSitePresentation();const siteName=presentation?.brand_name||appEnv.appName;
  const { t, locale, setLocale } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const isAdmin = session?.user.role === "admin";
  const access = useConsoleResource(isAdmin ? token : "", verifyAdminAccess);
  const action = useConsoleAction(token);
  const [navOpen, setNavOpen] = useState(false);
  const current = adminPrimarySection(activeAdminSection(pathname));
  useEffect(() => { setNavOpen(false); }, [pathname]);
  useEffect(() => {
    const desktop = window.matchMedia("(min-width: 901px)");
    const close = () => { if (desktop.matches) setNavOpen(false); };
    desktop.addEventListener("change", close);
    return () => desktop.removeEventListener("change", close);
  }, []);
  function signOut() { void action.run(() => logout(token), () => {}); }
  const languageSwitch = <LanguagePicker locale={locale} label={t({ en: "Choose language", zh: "选择语言" })} onSelect={value => setLocale(interfaceLocale(value))} />;
  const brand = <Link className={styles.brand} href="/" aria-label={`${siteName} · ${t({ en: "Home", zh: "首页" })}`}>
    <BrandMark className={styles.brandMark} />
    <strong>{siteName}</strong><span className={styles.tag}>{t({ en: "Admin", zh: "管理后台" })}</span>
  </Link>;
  const header = <header className={styles.topbar}><div className={styles.headerInner}>{brand}<div className={styles.account}>{languageSwitch}{ready && isAdmin && access.data ? <AdminIdentity email={session?.user.email || ""} name={session?.user.username || ""} busy={action.busy} onSignOut={signOut} /> : null}</div></div></header>;

  if (!ready || (isAdmin && !access.data && !access.error)) return <div className={styles.entry}>{header}<main className={styles.entryLoading} aria-busy="true"><PendingLabel>{t({ en: "Checking administrator access…", zh: "正在验证管理员权限…" })}</PendingLabel><div className="button-row"><Link className="button" href="/">{t({ en: "Back to website", zh: "返回网站" })}</Link>{token ? <button type="button" className="button" onClick={signOut}>{t({ en: "Sign out", zh: "退出登录" })}</button> : null}</div></main></div>;
  const unavailable = isAdmin && access.error && !(access.error instanceof ApiError && access.error.status === 403);
  if (!isAdmin || access.error) return <div className={styles.entry}>{header}
    <main className={styles.loginCard}>
      <span className={styles.eyebrow}>{t({ en: "Website administration", zh: "网站管理" })}</span>
      <h1>{!session ? t({ en: "Administrator sign-in", zh: "管理员登录" }) : unavailable ? t({ en: "Administration is temporarily unavailable", zh: "管理后台暂时无法连接" }) : t({ en: "Administrator access required", zh: "需要管理员权限" })}</h1>
      <p>{!session ? t({ en: "Sign in with your administrator account to manage this website.", zh: "使用管理员账号登录，管理网站的用户、订阅与服务配置。" }) : unavailable ? t({ en: "We couldn't connect to the administration service. Check your connection and try again.", zh: "暂时无法连接管理服务，请检查网络后重试。" }) : t({ en: "This account cannot access website management. Your personal account is available separately.", zh: "当前账号没有网站管理权限，可前往账户中心管理个人账户。" })}</p>
      {!session ? <AdminSignInPanel /> : <div className="button-row"><Link className="button primary" href="/account">{t({ en: "My account", zh: "前往账户中心" })}</Link><button className="button" type="button" disabled={action.busy} onClick={signOut}>{t({ en: "Use another account", zh: "切换账号" })}</button></div>}
      <ConsoleError error={access.error} retry={isAdmin ? () => void access.refresh() : undefined} /><ConsoleError error={action.error} />
      <Link className={styles.publicLink} href="/">{t({ en: "Back to website", zh: "返回网站" })}<span aria-hidden="true"> ↗</span></Link>
    </main>
  </div>;

  function navigation() {
    return <nav className={styles.navigation} aria-label={t({ en: "Administrator navigation", zh: "后台导航" })}>
      {adminNavigationGroups.map(group => <div className={styles.navGroup} key={group.id}>{adminNavigationGroups.length > 1 ? <h2 className={styles.navLabel}>{t(group.label)}</h2> : null}<ul>
        {group.sections.map(section => <li key={section}><Link href={adminSectionHref(section)} aria-current={section === current ? "page" : undefined} onNavigate={() => { if (section === current) setNavOpen(false); }}>
          <InterfaceIcon name={adminSectionIcons[section]} /><NavigationLabel label={t(adminSectionNames[section])} className={styles.navLinkContent} statusClassName={styles.navigationStatus} />
        </Link></li>)}
      </ul></div>)}
    </nav>;
  }
  return <div className={styles.shell}>
    <a className="skip-content" href="#admin-content">{t({ en: "Skip to content", zh: "跳到正文" })}</a>{header}
    <div className={styles.container}><div className={styles.breadcrumb}><button type="button" className={styles.mobileTrigger} aria-label={t({ en: "Open administrator navigation", zh: "打开后台导航" })} aria-haspopup="dialog" aria-expanded={navOpen} onClick={() => setNavOpen(true)}><InterfaceIcon name="menu" /></button><Link href="/admin">{t({ en: "Administration", zh: "管理后台" })}</Link>{current && current !== "overview" ? <><span aria-hidden="true">/</span><span>{t(adminSectionNames[current])}</span></> : null}</div>
      <div className={styles.grid}><aside className={styles.sidebar}>{navigation()}<Link className={styles.supportLink} href="/docs/admin">{t({ en: "Administration guide", zh: "后台使用指南" })}<span aria-hidden="true">↗</span></Link></aside>
        <main id="admin-content" tabIndex={-1} className={styles.content}><ConsoleError error={action.error} />{children}</main>
      </div>
    </div>
    <ResponsiveDialog open={navOpen} onClose={() => setNavOpen(false)} title={t({ en: "Administration", zh: "管理后台" })} closeLabel={t({ en: "Close navigation", zh: "关闭导航" })} presentation="side">{navigation()}</ResponsiveDialog>
  </div>;
}
