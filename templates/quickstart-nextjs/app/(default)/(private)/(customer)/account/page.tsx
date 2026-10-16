"use client";

import { InterfaceIcon } from "@/components/interface-icon";
import { SignOutButton } from "@/components/sign-out-button";
import { SiteShell } from "@/components/site-shell";
import { ConsoleError, ConsoleGate, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { StatusBadge } from "@/components/status-badge";
import { usePageReturn } from "@/components/use-page-return";
import { getMembership } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { formatDate } from "@/lib/format";
import styles from "./page.module.css";

export default function AccountPage() {
 const { t, locale } = useI18n();
 const { session, ready } = useConsoleSession();
 const token = session?.token || "";
 const membership = useConsoleResource(token, getMembership);
 usePageReturn(membership.refresh);
 const plan = membership.data?.plan;
 const planLabel = plan === "free" ? t({ en: "Free", zh: "免费版" })
  : plan === "starter" ? t({ en: "Starter", zh: "入门版" })
  : plan === "pro" ? t({ en: "Pro", zh: "专业版" }) : plan === "premium" ? t({ en: "Premium", zh: "高级版" }) : plan === "lifetime" ? t({ en: "Lifetime", zh: "终身版" }) : "—";
 return <SiteShell eyebrow={t({ en: "Account", zh: "账户" })} title={t({ en: "My account", zh: "账户中心" })}
  description={t({ en: "Your identity and membership.", zh: "查看账户身份与会员等级。" })}>
  <ConsoleGate session={session} ready={ready}>
   {session ? <section className={styles.account} aria-labelledby="account-identity">
    <div className={styles.identityHeading}><span className={styles.icon}><InterfaceIcon name="account" /></span><h2 id="account-identity">{t({ en: "Your account", zh: "账户信息" })}</h2></div>
    <p className={styles.email}>{session.user.email}</p>
    <p className={styles.note}>{t({ en: "Account notices are sent to this email.", zh: "账户通知将发送至此邮箱。" })}</p>
    <div className={styles.membership}><span>{t({ en: "Membership", zh: "会员等级" })}</span><div aria-live="polite"><StatusBadge tone={plan && plan !== "free" && membership.data?.active ? "success" : "neutral"}>{membership.loading && !membership.data ? t({ en: "Loading…", zh: "正在读取…" }) : planLabel}</StatusBadge>{membership.data?.valid_until ? <small>{t({ en: "Until", zh: "有效期至" })} {formatDate(membership.data.valid_until, locale)}</small> : null}</div></div>
    <ConsoleError error={membership.error} />
    <div className={styles.actions}><SignOutButton session={session} className="button" /></div>
   </section> : null}
  </ConsoleGate>
 </SiteShell>;
}
