"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { useI18n } from "@/lib/i18n";
import { getPublicSiteSettings, publicSiteSettingsFallback, SITE_SETTINGS_EVENT } from "@/lib/site-settings";
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
      eyebrow={t({ en: "Contact", zh: "联系支持" })}
      title={t({ en: "A little help goes a long way.", zh: "有问题，从这里找到帮助。" })}
      description={t({ en: "Find a guide or reach the support channel provided by this site.", zh: "查阅使用指南，或通过本站提供的支持渠道联系我们。" })}
      sideTitle={t({ en: "Before you send", zh: "描述问题时" })} showEnvironment={false}
      sideBody={<p>{t({ en: "Tell us which page you were using and what you expected to happen. Keep passwords, login codes, and full card details out of your message.", zh: "请说明在哪个页面操作，以及原本期望的结果。不要在消息里附上密码、登录验证码或完整银行卡信息。" })}</p>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Contact", zh: "联系支持" }) }]}
      toc={[{ id: "support", label: t({ en: "Support channels", zh: "支持渠道" }) }]}
    >
      <section id="support">
        <div className={styles.contacts}>
          <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Self service", zh: "自助帮助" })}</span><h2>{t({ en: "Start with the guides", zh: "先看看使用指南" })}</h2><p>{t({ en: "Setup instructions, invitation rules, and a clear explanation of the starter’s main flows.", zh: "查看配置说明、邀请规则，以及模板主要使用流程。" })}</p><Link className="button" href="/docs">{t({ en: "Read documentation", zh: "查看文档" })} ↗</Link></article>
          {settings.support_email ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Email", zh: "邮件支持" })}</span><h2>{t({ en: "Write to support", zh: "发送支持邮件" })}</h2><p>{t({ en: "Use the email address provided by the site operator.", zh: "通过站点运营者提供的邮箱联系支持。" })}</p><a className="button primary" href={`mailto:${settings.support_email}`}>{settings.support_email} ↗</a></article> : null}
          {settings.support_url ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Help center", zh: "帮助中心" })}</span><h2>{t({ en: "Open the support page", zh: "打开支持页面" })}</h2><p>{t({ en: "Continue to the help channel configured for this site.", zh: "前往本站配置的帮助渠道，继续处理你的问题。" })}</p><a className="button primary" href={settings.support_url} target="_blank" rel="noopener noreferrer">{t({ en: "Visit support", zh: "前往支持页面" })} ↗</a></article> : null}
          {!hasSupport ? <article className={styles.contactCard}><span className="panel-kicker">{t({ en: "Direct support", zh: "直接联系" })}</span><h2>{state === "loading" ? t({ en: "Finding support channels…", zh: "正在获取支持渠道…" }) : t({ en: "Support details are not available yet", zh: "暂无可用的直接支持渠道" })}</h2><p role="status">{state === "loading" ? t({ en: "Checking the site’s contact information.", zh: "正在查询本站的联系信息。" }) : state === "unavailable" ? t({ en: "Contact information could not be loaded. Try again later or browse the guides in the meantime.", zh: "暂时无法获取联系信息。请稍后重试，或先查阅使用指南。" }) : t({ en: "The site operator has not published a support email or help page. You can still browse the documentation.", zh: "站点运营者尚未公布支持邮箱或帮助页面，你仍然可以查阅文档。" })}</p>{state === "unavailable" ? <button className="button" type="button" onClick={() => window.dispatchEvent(new Event(SITE_SETTINGS_EVENT))}>{t({ en: "Try again", zh: "重试" })}</button> : null}</article> : null}
        </div>
        <p className={styles.supportHint}>{t({ en: "Questions about how your information is handled? Read the privacy page before contacting support.", zh: "想了解信息如何使用和处理？可以先阅读隐私说明。" })} <Link href="/privacy">{t({ en: "Privacy →", zh: "隐私说明 →" })}</Link></p>
      </section>
    </SiteShell>
  );
}
