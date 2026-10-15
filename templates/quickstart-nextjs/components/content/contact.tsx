"use client";

import { LocalizedLink as Link } from "@/components/localized-link";
import { useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { useI18n } from "@/lib/i18n";
import { getPublicSiteSettings, publicSiteSettingsFallback, SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { siteLinks } from "@/content/navigation";
import styles from "./content.module.css";

export function ContactContent() {
  const { t } = useI18n();
  const [settings, setSettings] = useState(publicSiteSettingsFallback);
  const [state, setState] = useState<"loading" | "ready" | "unavailable">("loading");
  useEffect(() => {
    let controller: AbortController | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let generation = 0;
    const load = () => {
      controller?.abort();
      clearTimeout(timer);
      const current = ++generation;
      controller = new AbortController();
      timer = setTimeout(() => controller?.abort(), 8000);
      setState("loading");
      void getPublicSiteSettings(controller.signal).then((value) => {
        if (current !== generation) return;
        setSettings(value);
        setState("ready");
      }).catch(() => {
        if (current === generation) setState("unavailable");
      }).finally(() => { if (current === generation) clearTimeout(timer); });
    };
    load();
    window.addEventListener(SITE_SETTINGS_EVENT, load);
    return () => { generation++; controller?.abort(); clearTimeout(timer); window.removeEventListener(SITE_SETTINGS_EVENT, load); };
  }, []);
  const hasSupport = Boolean(settings.support_email || settings.support_url);
  return (
    <SiteShell
      eyebrow={t({ en: "YOUR ACCOUNT", zh: "账户服务" })}
      title={t(siteLinks.help.label)}
      description={t({ en: "Manage common account questions or contact this site’s support team.", zh: "处理账户、订阅与积分问题，或通过本站提供的渠道联系支持。" })}
      sideTitle={t({ en: "Before you send", zh: "描述问题时" })} showEnvironment={false}
      sideBody={<p>{t({ en: "Tell us which page you were using and what you expected to happen. Keep passwords, login codes, and full card details out of your message.", zh: "请说明在哪个页面操作，以及原本期望的结果。不要在消息里附上密码、登录验证码或完整银行卡信息。" })}</p>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t(siteLinks.help.label) }]}
      toc={[{ id: "account-help", label: t({ en: "Account help", zh: "账户帮助" }) }, { id: "support", label: t({ en: "Support channels", zh: "支持渠道" }) }]}
    >
      <section id="account-help" className={styles.accountHelp}>
        <h2>{t({ en: "How can we help?", zh: "需要哪方面的帮助？" })}</h2>
        <div className={styles.helpLinks}>{[siteLinks.account, siteLinks.billing, siteLinks.referrals, siteLinks.credits].map(item => <Link key={item.href} href={item.href}>{t(item.label)}<span aria-hidden="true">→</span></Link>)}</div>
        <details><summary>{t({ en: "Where can I manage my subscription and invoices?", zh: "在哪里管理订阅和发票？" })}</summary><p>{t({ en: "Open Subscription & billing to see your plan and billing options. If your subscription supports it, open billing management there to update payment details or download invoices.", zh: "进入订阅与账单，查看当前套餐与可用操作。支持账单管理的订阅可从该页更新支付方式、下载发票。" })}</p></details>
        <details><summary>{t({ en: "How do I check a credit balance or reward?", zh: "如何查询积分余额和推荐奖励？" })}</summary><p>{t({ en: "Your account shows the available balance. Credit history records changes; Referral rewards shows attributed invitations and their activation status.", zh: "账户中心显示可用积分；积分明细记录每次变动。推荐奖励页可查看邀请归因和激活状态。" })}</p></details>
        <details><summary>{t({ en: "Payment completed, but access has not updated?", zh: "支付完成后，会员或积分还没有更新？" })}</summary><p>{t({ en: "Return to your account and refresh the payment status. If it remains unchanged, contact support with the payment reference and account email. Include the request ID when an error displays one.", zh: "返回账户并刷新支付状态。若仍未更新，请通过下方渠道提供支付编号和账户邮箱；错误提示中有请求编号时，也请一并提供。" })}</p></details>
      </section>
      <section id="support">
        <div className={styles.contacts}>
          {settings.support_email ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Email", zh: "邮件支持" })}</span><h2>{t({ en: "Write to support", zh: "发送支持邮件" })}</h2><p>{t({ en: "Use the email address provided by the site operator.", zh: "通过站点运营者提供的邮箱联系支持。" })}</p><a className="button primary" href={`mailto:${settings.support_email}`}>{settings.support_email} ↗</a></article> : null}
          {settings.support_url ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Help center", zh: "帮助中心" })}</span><h2>{t({ en: "Open the support page", zh: "打开支持页面" })}</h2><p>{t({ en: "Continue to the help channel configured for this site.", zh: "前往本站配置的帮助渠道，继续处理你的问题。" })}</p><a className="button primary" href={settings.support_url} target="_blank" rel="noopener noreferrer">{t({ en: "Visit support", zh: "前往支持页面" })} ↗</a></article> : null}
          {!hasSupport ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Direct support", zh: "直接联系" })}</span><h2>{state === "loading" ? t({ en: "Finding support channels…", zh: "正在获取支持渠道…" }) : t({ en: "Support details are not available yet", zh: "暂无可用的直接支持渠道" })}</h2><p role="status">{state === "loading" ? t({ en: "Checking the site’s contact information.", zh: "正在查询本站的联系信息。" }) : state === "unavailable" ? t({ en: "Contact information could not be loaded. Try again later or use the account links above.", zh: "暂时无法获取联系信息。请稍后重试，或使用上方的账户入口。" }) : t({ en: "The site operator has not published a support email or help page. You can still manage your account above.", zh: "站点运营者尚未公布支持邮箱或帮助页面，你仍然可以通过上方入口管理账户。" })}</p>{state === "unavailable" ? <button className="button" type="button" onClick={() => window.dispatchEvent(new Event(SITE_SETTINGS_EVENT))}>{t({ en: "Try again", zh: "重试" })}</button> : null}</article> : null}
        </div>
        <p className={styles.supportHint}>{t({ en: "Questions about how your information is handled? Read the privacy page before contacting support.", zh: "想了解信息如何使用和处理？可以先阅读隐私说明。" })} <Link href="/privacy">{t({ en: "Privacy →", zh: "隐私说明 →" })}</Link></p>
      </section>
    </SiteShell>
  );
}
