"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { ApiError, logout } from "@/lib/api";
import { clearSessionIfToken } from "@/lib/auth";
import { activeAdminSection, adminSectionHref, adminSectionNames, adminSections } from "@/lib/admin-navigation";
import { getOperatorOverview } from "@/lib/operations-api";
import { getPublicSiteSettings, publicSiteSettingsFallback, SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { useI18n } from "@/lib/i18n";
import { ConsoleError, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { SignInPanel } from "./sign-in-panel";
import styles from "./admin-shell.module.css";

async function verifyAdminAccess(token: string) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 8000);
  try { return await getOperatorOverview(token, controller.signal); } finally { clearTimeout(timer); }
}

// This layout is mounted only by app/admin/layout.tsx. It never uses the
// customer SiteShell or fetches the administrator's personal billing summary.
export function AdminShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { t, locale, setLocale } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const isAdmin = session?.user.role === "admin";
  const access = useConsoleResource(isAdmin ? token : "", verifyAdminAccess);
  const action = useConsoleAction(token);
  const [brand, setBrand] = useState(publicSiteSettingsFallback.brand_name);
  const current = activeAdminSection(pathname);

  useEffect(() => {
    let controller: AbortController | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const sync = () => {
      controller?.abort();
      if (timer) clearTimeout(timer);
      const request = new AbortController();
      controller = request;
      timer = setTimeout(() => request.abort(), 8000);
      void getPublicSiteSettings(request.signal).then((settings) => {
        if (!request.signal.aborted) setBrand(settings.brand_name);
      }).catch(() => {}).finally(() => { if (controller === request && timer) clearTimeout(timer); });
    };
    sync();
    window.addEventListener(SITE_SETTINGS_EVENT, sync);
    return () => { controller?.abort(); if (timer) clearTimeout(timer); window.removeEventListener(SITE_SETTINGS_EVENT, sync); };
  }, []);

  function signOut() {
    void action.run(async () => {
      clearSessionIfToken(token);
      await logout(token);
    }, () => {});
  }
  const languageSwitch = <div className={styles.languages} aria-label={t({ en: "Language", zh: "语言" })}>
    {(["en", "zh"] as const).map((value) => <button type="button" key={value} aria-pressed={locale === value} onClick={() => setLocale(value)}>{value === "en" ? "EN" : "中文"}</button>)}
  </div>;

  if (!ready || (isAdmin && !access.data && !access.error)) return <div className={styles.entry}><div className={styles.entryBar}><span className={styles.tag}>ADMIN</span>{languageSwitch}</div><p role="status">{t({ en: "Checking administrator access…", zh: "正在验证管理员权限…" })}</p><div className={styles.loadingActions}><Link className="button" href="/">{t({ en: "Back to website", zh: "返回网站首页" })}</Link>{token ? <button type="button" className="button" onClick={signOut}>{t({ en: "Sign out", zh: "退出登录" })}</button> : null}</div></div>;

  const unavailable = isAdmin && access.error && !(access.error instanceof ApiError && access.error.status === 403);
  if (!isAdmin || access.error) return <div className={styles.entry}>
    <header className={styles.entryBar}><Link href="/">{brand}</Link><span className={styles.tag}>ADMIN</span>{languageSwitch}</header>
    <main className={styles.loginCard}>
      <span className={styles.eyebrow}>{t({ en: "Website administration", zh: "网站管理员后台" })}</span>
      <h1>{!session ? t({ en: "Administrator sign-in", zh: "管理员登录" }) : unavailable ? t({ en: "Administration is temporarily unavailable", zh: "管理后台暂时无法连接" }) : t({ en: "Administrator access required", zh: "需要网站管理员权限" })}</h1>
      <p>{!session ? t({ en: "Sign in with your administrator account to manage this website.", zh: "请使用网站管理员账号登录，管理全站用户、订阅和网站配置。" }) : unavailable ? t({ en: "We couldn't connect to the administration service. Try again when your connection is available.", zh: "暂时无法连接管理服务，请检查网络后重试。" }) : t({ en: "This account cannot open website management. You can manage your own subscription in your account center.", zh: "当前账号无法进入网站管理。你可以在用户中心管理自己的订阅和账户。" })}</p>
      {!session ? <SignInPanel compact returnTo={pathname} /> : <div className="button-row"><Link className="button primary" href="/account">{t({ en: "Go to my account", zh: "返回用户中心" })}</Link><button className="button" type="button" disabled={action.busy} onClick={signOut}>{t({ en: "Use another account", zh: "切换登录账号" })}</button></div>}
      <ConsoleError error={access.error} retry={isAdmin ? () => void access.refresh() : undefined} />
      <ConsoleError error={action.error} />
      <Link className={styles.publicLink} href="/">{t({ en: "Back to website", zh: "返回网站首页" })} ↗</Link>
    </main>
  </div>;

  return <div className={styles.shell}>
    <aside className={styles.sidebar}>
      <Link className={styles.brand} href="/admin"><span className={styles.brandMark} aria-hidden="true">A</span><span><strong>{brand}</strong><small>{t({ en: "Website administration", zh: "网站管理员后台" })}</small></span></Link>
      <span className={styles.navLabel}>{t({ en: "SITE MANAGEMENT", zh: "全站管理" })}</span>
      <nav className={styles.navigation} aria-label={t({ en: "Administrator navigation", zh: "网站管理员导航" })}>
        {adminSections.map((section, index) => <Link key={section} href={adminSectionHref(section)} aria-current={section === current ? "page" : undefined}><span className={styles.navIndex} aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>{t(adminSectionNames[section])}</Link>)}
      </nav>
      <div className={styles.sidebarFoot}><span className={styles.tag}>ADMIN</span><p>{t({ en: "Manage the entire website", zh: "管理整个网站的业务与配置" })}</p><Link href="/">{t({ en: "View website", zh: "访问网站" })} ↗</Link></div>
    </aside>
    <div className={styles.main}>
      <header className={styles.topbar}><div><span className={styles.tag}>ADMIN</span><strong>{current ? t(adminSectionNames[current]) : t({ en: "Website administration", zh: "网站管理员后台" })}</strong></div><div className={styles.account}>{languageSwitch}<span className={styles.identity} title={session?.user.email}>{session?.user.email}</span><button type="button" className={styles.logout} onClick={signOut} disabled={action.busy}>{t({ en: "Sign out", zh: "退出登录" })}</button></div></header>
      <main className={styles.content}><ConsoleError error={action.error} />{children}</main>
    </div>
  </div>;
}
