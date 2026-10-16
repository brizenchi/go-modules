"use client";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { getBusinessSettings, saveBusinessSettings, type BusinessSettings } from "@/lib/operations-api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { ConsoleError, ConsoleGate, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { AdminPageHeader } from "./admin-page-header";
import { SiteLogoField } from "./site-logo-field";
import { safeLogoURL } from "@/lib/site-branding";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";
const defaults: BusinessSettings = { brand_name: "", logo_url: "", description: "", description_zh: "", headline_en: "", headline_zh: "", sender_email: "", sender_name: "", support_email: "", support_url: "", export_credit_cost: 1 };
const keys = ["brand_name", "logo_url", "description", "description_zh", "sender_email", "sender_name", "support_email", "support_url"] as const;
export function WebsiteConsole() {
 const { t } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const loader = useCallback((current: string) => getBusinessSettings(current), []); const resource = useConsoleResource(token, loader);
 const [logoBusy, setLogoBusy] = useState(false);
 const [settings, setSettings] = useState<BusinessSettings | null>(null); const [reason, setReason] = useState(""); const action = useConsoleAction(token); const intent = useRef({ payload: "", key: "" });
 useEffect(() => { if (resource.data) setSettings({ ...defaults, ...resource.data }); }, [resource.data]);
 function submit(event: FormEvent) {
  event.preventDefault(); if (!settings || logoBusy || (settings.logo_url?.trim() && !safeLogoURL(settings.logo_url))) return;
  const payload = { ...Object.fromEntries(keys.map(key => [key, settings[key] || ""])), export_credit_cost: settings.export_credit_cost, reason: reason.trim() } as BusinessSettings & { reason: string };
  const serialized = JSON.stringify(payload); if (intent.current.payload !== serialized) intent.current = { payload: serialized, key: newIntentKey() };
  void action.run(() => saveBusinessSettings(token, payload, intent.current.key), () => { setReason(""); intent.current = { payload: "", key: "" }; window.dispatchEvent(new Event(SITE_SETTINGS_EVENT)); action.setMessage(t({ en: "Settings saved. Public branding refreshes after subsequent requests, usually within one minute. Sender changes apply after an API restart.", zh: "设置已保存。官网品牌将在后续访问触发缓存刷新，通常一分钟内生效；发件设置在后端重启后生效。" })); void resource.refresh(); });
 }
 function field(key: typeof keys[number], label: { en: string; zh: string }, multiline = false) {
  const props = { id: `website-${key}`, value: String(settings?.[key] || ""), onChange: (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => settings && setSettings({ ...settings, [key]: event.target.value }), maxLength: key === "brand_name" || key === "sender_name" ? 100 : key.includes("headline") ? 160 : key === "support_url" ? 1024 : multiline ? 500 : 255, required: key === "brand_name" };
  return <div className="field"><label htmlFor={props.id}>{t(label)}</label>{multiline ? <textarea {...props} rows={3} /> : <input {...props} type={key.endsWith("email") ? "email" : key === "support_url" ? "url" : "text"} pattern={key === "support_url" ? "https://.+" : undefined} />}</div>;
 }
 return <><AdminPageHeader section="settings" onRefresh={() => void resource.refresh()} busy={resource.loading} /><ConsoleGate session={session} ready={ready} admin><ConsoleError error={resource.error} retry={() => void resource.refresh()} /><ConsoleError error={action.error} />{action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
  {settings ? <form className={styles.websiteForm} onSubmit={submit}><fieldset disabled={action.busy} className={styles.websiteSection}><legend>{t({ en: "Website identity", zh: "网站资料" })}</legend><p>{t({ en: "Used by the public header, footer and site identity. Page copy and search descriptions are managed in Content.", zh: "用于官网页头、页脚与站点身份。页面文案和搜索描述在内容管理中维护。" })}</p>{field("brand_name", { en: "Website name", zh: "网站名称" })}<SiteLogoField token={token} value={settings.logo_url || ""} brand={settings.brand_name} disabled={action.busy} onChange={logo_url => setSettings(current => current ? { ...current, logo_url } : current)} onBusyChange={setLogoBusy} /><div className={styles.editorMeta}>{field("description", { en: "Service introduction · EN", zh: "服务介绍 · EN" }, true)}{field("description_zh", { en: "Service introduction · 中文", zh: "服务介绍 · 中文" }, true)}</div><p><Link href="/admin/articles">{t({ en: "Edit and publish homepage copy in Content → Pages & introductions.", zh: "首页文案请到「内容管理 → 页面与栏目介绍」编辑并发布。" })}</Link></p></fieldset>
   <fieldset disabled={action.busy} className={styles.websiteSection}><legend>{t({ en: "Help & support", zh: "帮助与支持" })}</legend><div className={styles.editorMeta}>{field("support_email", { en: "Support email", zh: "客服邮箱" })}{field("support_url", { en: "Help page (HTTPS)", zh: "支持页面（HTTPS）" })}</div></fieldset>
   <fieldset disabled={action.busy} className={styles.websiteSection}><legend>{t({ en: "Email delivery", zh: "邮件发送" })}</legend><p>{t({ en: "Set the verified sender address and name. Provider credentials stay in deployment configuration. Restart the API after changing the sender.", zh: "填写服务商已验证的发件邮箱与名称，服务商密钥由部署配置管理。修改发件设置后需重启后端。" })}</p><div className={styles.editorMeta}>{field("sender_email", { en: "Sender email", zh: "发件邮箱" })}{field("sender_name", { en: "Sender name", zh: "发件人名称" })}</div></fieldset>
   <div className={styles.editorSave}><div className="field"><label htmlFor="website-reason">{t({ en: "Change reason", zh: "修改原因" })}</label><input id="website-reason" required minLength={3} maxLength={500} value={reason} onChange={event => setReason(event.target.value)} disabled={action.busy} /></div><button type="submit" className="button primary" disabled={action.busy || logoBusy || Boolean(settings.logo_url?.trim() && !safeLogoURL(settings.logo_url)) || Array.from(reason.trim()).length < 3} aria-busy={action.busy}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "保存中…" })}</PendingLabel> : t({ en: "Save settings", zh: "保存设置" })}</button></div>
  </form> : null}
 </ConsoleGate></>;
}
