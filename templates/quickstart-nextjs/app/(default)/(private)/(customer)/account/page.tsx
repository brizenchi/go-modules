"use client";

import { useEffect, useState } from "react";
import { LocalizedLink as Link } from "@/components/localized-link";
import { InterfaceIcon } from "@/components/interface-icon";
import { AccountDialog } from "@/components/account-dialog";
import { SignInDialog } from "@/components/sign-in-dialog";
import { SiteShell } from "@/components/site-shell";
import { readSession, SESSION_EVENT, type AuthSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { accountServiceItems } from "@/lib/workspace";
import styles from "./page.module.css";

export default function AccountPage() {
  const { t } = useI18n();
  const [session, setSession] = useState<AuthSession | null>(null);
  const [ready, setReady] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
  const [signInOpen, setSignInOpen] = useState(false);
  useEffect(() => {
    const sync = () => { setSession(readSession()); setReady(true); };
    sync();
    window.addEventListener(SESSION_EVENT, sync); window.addEventListener("storage", sync);
    return () => { window.removeEventListener(SESSION_EVENT, sync); window.removeEventListener("storage", sync); };
  }, []);
  return <SiteShell eyebrow={t({ en: "Account", zh: "账户中心" })}
    title={t({ en: "My account", zh: "账户中心" })}
    description={t({ en: "Manage your subscription, credits and rewards.", zh: "管理你的订阅、积分与推荐奖励。" })}>
    {!ready ? <p className={styles.message} role="status">{t({ en: "Loading your account…", zh: "正在加载账户…" })}</p>
      : !session ? <section className={styles.guest}>
        <h2>{t({ en: "Sign in to continue", zh: "登录后开始使用" })}</h2>
        <p>{t({ en: "Access your subscription, credits and rewards.", zh: "查看你的订阅、积分和推荐奖励。" })}</p>
        <button className="button primary" type="button" onClick={() => setSignInOpen(true)}>{t({ en: "Sign in", zh: "登录" })}</button>
      </section> : <div className={styles.layout}>
        <section className={styles.tools} aria-labelledby="workspace-tools">
          <div className={styles.sectionHead}><h2 id="workspace-tools">{t({ en: "Account services", zh: "账户服务" })}</h2><span>{t({ en: "Your subscriptions and usage", zh: "套餐与使用记录" })}</span></div>
          {accountServiceItems.map((item) => <Link className={styles.row} href={item.href} key={item.href}>
            <span className={styles.toolIcon}><InterfaceIcon name={item.icon} /></span>
            <span className={styles.rowCopy}><strong>{t(item.label)}</strong><small>{t(item.description)}</small></span>
            <InterfaceIcon name="arrow" className={styles.arrow} />
          </Link>)}
        </section>
        <aside className={styles.account} aria-labelledby="workspace-account">
          <span className={styles.accountIcon}><InterfaceIcon name="account" /></span><h2 id="workspace-account">{t({ en: "Your account", zh: "我的账户" })}</h2>
          <p className={styles.email}>{session.user.email}</p>
          <button className={styles.textButton} type="button" onClick={() => setAccountOpen(true)}>{t({ en: "Account information", zh: "账户信息" })}<span aria-hidden="true"> →</span></button>
          <p className={styles.message}>{t({ en: "View your sign-in email or sign out from account information.", zh: "在账户信息中查看登录邮箱或退出登录。" })}</p>
          <Link className={styles.pricing} href="/pricing">{t({ en: "Explore plans", zh: "查看可选套餐" })}<span aria-hidden="true"> ↗</span></Link>
        </aside>
      </div>}
    <AccountDialog open={accountOpen} session={session} onClose={() => setAccountOpen(false)} />
    <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} />
  </SiteShell>;
}
