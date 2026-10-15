"use client";

import Link from "next/link";
import { Suspense, useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useSearchParams } from "next/navigation";
import adminStyles from "./admin-shell.module.css";
import { Panel } from "./ui";
import { ConsoleError, ConsoleGate, consoleStyles as styles, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { useI18n } from "@/lib/i18n";
import { formatDate } from "@/lib/format";
import { getBusinessSettings, getOperatorOverview, grantCredits, listCreditTransactions, listOperator, refundCredits, retryReferralReward, saveBusinessSettings, type BusinessSettings, type PageResult } from "@/lib/operations-api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { adminSections, adminSectionNames, adminSectionDescriptions, adminSectionHref } from "@/lib/admin-navigation";
import { IntegrationsConsole } from "./integrations-console";
import { AdminPageHeader } from "./admin-page-header";
import { DataTable, type DataColumn } from "./data-table";
import { StatusBadge } from "./status-badge";
import { ResponsiveDialog } from "./responsive-dialog";
import { PendingLabel } from "./pending-label";
import { CustomerPageLoading } from "./customer-page-loading";
import { InterfaceIcon } from "./interface-icon";
import { adminTag } from "@/lib/admin-presentation";

export const operatorSections = adminSections;
export type OperatorSection = typeof operatorSections[number];
type Row = Record<string, unknown>;
const names = adminSectionNames;
const columns: Partial<Record<OperatorSection, Array<[string, { en: string; zh: string }]>>> = {
  users: [["email", { en: "Email", zh: "邮箱" }], ["id", { en: "Account ID", zh: "账号 ID" }], ["role", { en: "Role", zh: "角色" }], ["credits", { en: "Credits", zh: "积分" }], ["created_at", { en: "Joined", zh: "注册时间" }]],
  orders: [["livemode", { en: "Environment", zh: "环境" }], ["id", { en: "Payment reference", zh: "支付编号" }], ["user_id", { en: "Account", zh: "账号" }], ["amount", { en: "Amount (minor units)", zh: "金额（最小单位）" }], ["status", { en: "Status", zh: "状态" }], ["created_at", { en: "Recorded", zh: "记录时间" }]],
  subscriptions: [["user_id", { en: "Account", zh: "账号" }], ["plan", { en: "Plan", zh: "套餐" }], ["status", { en: "Status", zh: "状态" }], ["provider_subscription_id", { en: "Subscription", zh: "订阅编号" }], ["period_end", { en: "Period end", zh: "本期截止" }]],
  referrals: [["referrer_id", { en: "Inviter", zh: "邀请人" }], ["referee_id", { en: "Invitee", zh: "受邀人" }], ["status", { en: "Status", zh: "状态" }], ["reward_credits", { en: "Reward", zh: "奖励积分" }], ["expires_at", { en: "Deadline", zh: "奖励截止" }]],
  credits: [["id", { en: "Transaction", zh: "流水编号" }], ["user_id", { en: "Account", zh: "账号" }], ["kind", { en: "Type", zh: "类型" }], ["amount", { en: "Credits", zh: "积分变动" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]],
  audit: [["status", { en: "Outcome", zh: "结果" }], ["actor_id", { en: "Operator", zh: "操作人" }], ["action", { en: "Action", zh: "操作" }], ["target_id", { en: "Target", zh: "对象" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]]
};

function OperatorConsoleInner({ section }: { section: Exclude<OperatorSection, "integrations"> }) {
  const { t, locale } = useI18n();
  const params = useSearchParams();
  const { session, ready } = useConsoleSession();
  const token = session?.user.role === "admin" ? session.token : "";
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [reason, setReason] = useState("");
  const reasonLength = Array.from(reason.trim()).length;
  const validReason = reasonLength >= (section === "settings" || section === "referrals" ? 3 : 1) && reasonLength <= 500;
  const [target, setTarget] = useState("");
  const [amount, setAmount] = useState("50");
  const [expiry, setExpiry] = useState("");
  const [refundID, setRefundID] = useState("");
  const [creditAction, setCreditAction] = useState<"grant" | "refund" | null>(null);
  const [selectedReferral, setSelectedReferral] = useState<number | null>(null);
  const [settings, setSettings] = useState<BusinessSettings | null>(null);
  const action = useConsoleAction(token);
  const intent = useRef({ payload: "", key: "" });
  function intentFor(payload: unknown) {
    const serialized = JSON.stringify(payload);
    if (intent.current.payload !== serialized || !intent.current.key) intent.current = { payload: serialized, key: newIntentKey() };
    return intent.current.key;
  }
  useEffect(() => { setPage(1); setSearch(""); setQuery(""); setStatus(""); setReason(""); setSelectedReferral(null); setCreditAction(null); setSettings(null); setTarget(""); setRefundID(""); setExpiry(""); intent.current = { payload: "", key: "" }; }, [section, token]);
  useEffect(() => { if (section === "credits") { const userID = params.get("user_id") || ""; setTarget(userID); setQuery(userID); setSearch(userID); } }, [params, section, token]);
  const loader = useCallback(async (currentToken: string): Promise<unknown> => {
    if (section === "overview") return getOperatorOverview(currentToken);
    if (section === "settings") return getBusinessSettings(currentToken);
    if (section === "credits") return listCreditTransactions(currentToken, page, query, true);
    return listOperator<Row>(currentToken, section, page, query, status);
  }, [page, query, section, status]);
  const resource = useConsoleResource(token, loader);
  useEffect(() => { if (section === "settings" && resource.data) setSettings(resource.data as BusinessSettings); }, [resource.data, section]);
  const result = resource.data && section !== "overview" && section !== "settings" ? resource.data as PageResult<Row> : null;
  const overview = section === "overview" ? resource.data as Record<string, number> | null : null;
  function success(message: string) { action.setMessage(message); setReason(""); intent.current = { payload: "", key: "" }; setSelectedReferral(null); setCreditAction(null); void resource.refresh(); }
  function submitSearch(event: FormEvent) { event.preventDefault(); setPage(1); setQuery(search.trim()); }
  function grant(event: FormEvent) {
    event.preventDefault();
    const payload = { user_id: target.trim(), amount: Number(amount), expires_at: expiry ? new Date(expiry).toISOString() : undefined, reason: reason.trim() };
    void action.run(() => grantCredits(token, { ...payload, idempotency_key: intentFor({ operation: "grant", ...payload }) }), () => success(t({ en: "Credits granted and recorded.", zh: "积分已发放，操作已记录。" })));
  }
  function refund(event: FormEvent) {
    event.preventDefault();
    const payload = { transaction_id: Number(refundID), reason: reason.trim() };
    void action.run(() => refundCredits(token, { ...payload, idempotency_key: intentFor({ operation: "refund", ...payload }) }), () => success(t({ en: "Credit refund recorded.", zh: "积分退回已记录。" })));
  }
  function retryReward(event: FormEvent) {
    event.preventDefault(); if (selectedReferral === null || !validReason) return;
    const id = selectedReferral;
    void action.run(() => retryReferralReward(token, id, reason.trim(), intentFor({ operation: "reward", id, reason: reason.trim() })), () => success(t({ en: "Invitation callback completed. Rewards depend on the site’s rules.", zh: "邀请回调已执行，是否发奖取决于本站规则。" })));
  }
  function saveSettings(event: FormEvent) {
    event.preventDefault(); if (!settings || !validReason || !settings.brand_name.trim()) return;
    const payload = { ...settings, reason: reason.trim() };
    void action.run(() => saveBusinessSettings(token, payload, intentFor(payload)), () => { window.dispatchEvent(new Event(SITE_SETTINGS_EVENT)); success(t({ en: "Site settings saved.", zh: "网站配置已保存。" })); });
  }
  function cell(row: Row, key: string) {
    const value = row[key];
    const tag = adminTag(section, key, value);
    if (tag) return <StatusBadge tone={tag.tone}>{t(tag.label)}</StatusBadge>;
    if (key === "amount" && section === "orders") return typeof value === "number" ? `${value} ${String(row.currency || "").toUpperCase()}` : t({ en: "Unknown", zh: "未知" });
    if (value === null || value === undefined || value === "") return "—";
    if (key.endsWith("_at") || key === "period_end") return formatDate(String(value), locale === "zh" ? "zh-CN" : "en-US");
    if (["user_id", "referrer_id", "referee_id"].includes(key)) {
      const email = row[key === "user_id" ? "email" : key === "referrer_id" ? "referrer_email" : "referee_email"];
      return <span className={adminStyles.cellIdentity}>{typeof email === "string" && email ? <span>{email}</span> : null}<small title={String(value)}>{String(value)}</small></span>;
    }
    if (key === "id" || key === "actor_id" || key === "target_id" || key === "provider_subscription_id") return <span className={adminStyles.mono} title={String(value)}>{String(value)}</span>;
    return String(value);
  }
  const tableColumns: DataColumn<Row>[] = (columns[section] || []).map(([key, label]) => ({ key, heading: t(label), cell: row => cell(row, key), align: ["amount", "credits", "reward_credits"].includes(key) ? "end" : "start" }));
  if (section === "users") tableColumns.push({ key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <Link className={`button ${adminStyles.tableAction}`} href={`/admin/credits?user_id=${encodeURIComponent(String(row.id))}`}>{t({ en: "Manage credits", zh: "管理积分" })}</Link> });
  if (section === "referrals") tableColumns.push({ key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <button className={`button ${adminStyles.tableAction}`} type="button" disabled={row.status !== "activated" || action.busy} onClick={() => { setSelectedReferral(Number(row.id)); setReason(""); }}>{t({ en: "Reconcile reward", zh: "核对奖励" })}</button> });
  const searchLabel = section === "credits" ? { en: "Account ID", zh: "账号 ID" }
    : section === "audit" ? { en: "Operator, action, target or reason", zh: "操作人、操作、对象或原因" }
    : section === "referrals" ? { en: "Email, account ID or referral code", zh: "邮箱、账号 ID 或邀请码" }
    : section === "users" ? { en: "Email, username or account ID", zh: "邮箱、用户名或账号 ID" }
    : { en: "Email or reference ID", zh: "邮箱或关联编号" };
  const dialogOpen = creditAction !== null || selectedReferral !== null;
  const pendingSubmit = (label: { en: string; zh: string }) => action.busy ? <PendingLabel>{t({ en: "Processing…", zh: "处理中…" })}</PendingLabel> : t(label);
  const settingField = (key: "brand_name" | "description" | "support_email" | "support_url", label: { en: string; zh: string }) => settings ? <div className="field" key={key}><label htmlFor={`setting-${key}`}>{t(label)}</label><input id={`setting-${key}`} type={key === "support_email" ? "email" : key === "support_url" ? "url" : "text"} required={key === "brand_name"} pattern={key === "support_url" ? "https://.+" : undefined} maxLength={key === "brand_name" ? 100 : key === "description" ? 500 : key === "support_url" ? 1024 : 255} value={settings[key]} onChange={(event) => setSettings({ ...settings, [key]: event.target.value })} /></div> : null;
  return <>
    <AdminPageHeader section={section} onRefresh={() => void resource.refresh()} busy={resource.loading} disabled={!token || action.busy} actions={section === "credits" ? <><button type="button" className="button" disabled={action.busy} onClick={() => { setCreditAction("refund"); setReason(""); }}>{t({ en: "Return credits", zh: "退回积分" })}</button><button type="button" className="button primary" disabled={action.busy} onClick={() => { setCreditAction("grant"); setReason(""); }}>{t({ en: "Grant credits", zh: "发放积分" })}</button></> : undefined} />
    <ConsoleGate session={session} ready={ready} admin>
      <div className={styles.stack}>
        <ConsoleError error={resource.error} retry={() => void resource.refresh()} />
        {!dialogOpen ? <ConsoleError error={action.error} /> : null}
        {action.message ? <p className={styles.success} role="status">{action.message}</p> : null}
        {resource.loading && (section === "overview" || section === "settings") ? <p role="status"><PendingLabel>{t({ en: "Loading current records…", zh: "正在读取最新记录…" })}</PendingLabel></p> : null}
        {overview ? <div className={adminStyles.metrics}>{([
          ["users", { en: "Registered users", zh: "注册用户" }], ["active_subscriptions", { en: "Active subscriptions", zh: "有效订阅" }], ["referrals", { en: "Total invitations", zh: "邀请总数" }], ["pending_referrals", { en: "Awaiting activation", zh: "待激活邀请" }], ["activated_referrals", { en: "Activated invitations", zh: "已激活邀请" }]
        ] as const).map(([key, label]) => <div className={adminStyles.metric} key={key}><span>{t(label)}</span><strong>{overview[key] ?? "—"}</strong></div>)}</div> : null}
        {section === "overview" ? <Panel title={t({ en: "Quick access", zh: "常用管理" })}><div className={adminStyles.quickLinks}>{(["users", "subscriptions", "integrations"] as const).map(destination => <Link key={destination} href={adminSectionHref(destination)}><div><strong>{t(names[destination])}</strong><p>{t(adminSectionDescriptions[destination])}</p></div><InterfaceIcon name="arrow" /></Link>)}</div></Panel> : null}
        {section !== "overview" && section !== "settings" ? <Panel title={t({ en: "Records", zh: "数据记录" })} actions={result ? <span className={adminStyles.recordCount}>{t({ en: `${result.total} records`, zh: `共 ${result.total} 条` })}</span> : undefined}>
          <form className={adminStyles.toolbar} onSubmit={submitSearch}>
            <div className={adminStyles.search}><div className="field"><label htmlFor="operator-search">{t(searchLabel)}</label><input id="operator-search" type="search" value={search} onChange={(event) => setSearch(event.target.value)} /></div><button className="button" type="submit" disabled={resource.loading}>{t({ en: "Search", zh: "搜索" })}</button></div>
            {section === "referrals" ? <div className="field"><label htmlFor="referral-status">{t({ en: "Activation status", zh: "激活状态" })}</label><select id="referral-status" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}><option value="">{t({ en: "All statuses", zh: "全部状态" })}</option><option value="pending">{t({ en: "Pending", zh: "待激活" })}</option><option value="activated">{t({ en: "Activated", zh: "已激活" })}</option><option value="expired">{t({ en: "Expired", zh: "已过期" })}</option></select></div> : null}
          </form>
          <DataTable caption={t(names[section])} columns={tableColumns} rows={result?.items || []} rowKey={row => String(row.id ?? row.user_id)} loading={resource.loading} loadingLabel={t({ en: "Loading records…", zh: "正在读取记录…" })} empty={t({ en: "No matching records. Try another search.", zh: "暂无匹配记录，可尝试其他搜索条件。" })} error={resource.error ? t({ en: "Records could not be loaded. Use the retry button above.", zh: "暂时无法读取记录，请点击上方重试。" }) : undefined} pagination={result ? { page: result.page, limit: result.limit, total: result.total, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (current, pages) => t({ en: `Page ${current} of ${pages}`, zh: `第 ${current} / ${pages} 页` }) } : undefined} />
        </Panel> : null}
        {section === "settings" && settings ? <Panel title={t({ en: "General settings", zh: "常规设置" })}><form className={adminStyles.settingsForm} onSubmit={saveSettings}>
          <fieldset className={adminStyles.formSection} disabled={action.busy}><legend>{t({ en: "Service details", zh: "服务资料" })}</legend><p>{t({ en: "These values describe the API service. Website branding, homepage copy and sharing cards are configured in content/site.ts and NEXT_PUBLIC_APP_NAME, then applied with a frontend rebuild.", zh: "以下资料用于 API 服务。网站品牌、首页文案和分享卡片在 content/site.ts 与 NEXT_PUBLIC_APP_NAME 中配置，重新构建前端后生效。" })}</p><div className={styles.formGrid}>{settingField("brand_name", { en: "Service name (API)", zh: "服务名称（API）" })}{settingField("description", { en: "Service description (API)", zh: "服务介绍（API）" })}</div></fieldset>
          <fieldset className={adminStyles.formSection} disabled={action.busy}><legend>{t({ en: "Help & support", zh: "帮助与支持" })}</legend><p>{t({ en: "Support channels take effect after saving.", zh: "支持渠道保存后生效。" })}</p><div className={styles.formGrid}>{settingField("support_email", { en: "Support email", zh: "支持邮箱" })}{settingField("support_url", { en: "Support URL (HTTPS)", zh: "支持页面（HTTPS）" })}</div></fieldset>
          <div className={adminStyles.formFooter}><div className="field"><label htmlFor="settings-reason">{t({ en: "Change reason", zh: "修改原因" })}</label><input id="settings-reason" required minLength={3} maxLength={500} placeholder={t({ en: "At least 3 characters; recorded in the audit log", zh: "至少填写 3 个字，将记录到审计日志" })} value={reason} disabled={action.busy} onChange={(event) => setReason(event.target.value)} /></div><button className="button primary" type="submit" aria-busy={action.busy} disabled={action.busy || !validReason || !settings.brand_name.trim()}>{pendingSubmit({ en: "Save changes", zh: "保存更改" })}</button></div>
        </form></Panel> : null}
      </div>
      <ResponsiveDialog open={dialogOpen} onClose={() => { setCreditAction(null); setSelectedReferral(null); }} title={selectedReferral !== null ? t({ en: `Reconcile invitation #${selectedReferral}`, zh: `核对邀请 #${selectedReferral} 的奖励` }) : t(creditAction === "grant" ? { en: "Grant credits", zh: "发放积分" } : { en: "Return consumed credits", zh: "退回已消费积分" })} description={selectedReferral !== null ? t({ en: "Replays the callback for an activated invitation. Rewards depend on your site’s rules; none are granted by default.", zh: "重新执行已激活邀请的回调，是否发奖取决于本站规则；默认不发放奖励。" }) : t(creditAction === "grant" ? { en: "The account, operator and reason are recorded. Leave expiry blank for credits that do not expire.", zh: "每次发放都会记录账号、操作人和原因，到期时间留空表示永久有效。" } : { en: "Enter the original consumption transaction. Each consumption can only be returned once.", zh: "填写原消费流水编号，同一笔消费只能退回一次。" })} closeLabel={t({ en: "Close", zh: "关闭" })}>
        {(dismiss) => <form className={adminStyles.dialogForm} onSubmit={selectedReferral !== null ? retryReward : creditAction === "grant" ? grant : refund}>
          <ConsoleError error={action.error} />
          {creditAction === "grant" ? <><div className="field"><label htmlFor="grant-user">{t({ en: "Account ID", zh: "账号 ID" })}</label><input id="grant-user" required disabled={action.busy} value={target} onChange={(event) => setTarget(event.target.value)} /></div><div className="field"><label htmlFor="grant-amount">{t({ en: "Credit amount", zh: "发放积分数" })}</label><input id="grant-amount" type="number" min={1} step={1} max={1000000} required disabled={action.busy} value={amount} onChange={(event) => setAmount(event.target.value)} /></div><div className="field"><label htmlFor="grant-expiry">{t({ en: "Expires at (optional)", zh: "到期时间（选填）" })}</label><input id="grant-expiry" type="datetime-local" disabled={action.busy} value={expiry} onChange={(event) => setExpiry(event.target.value)} /></div></> : creditAction === "refund" ? <div className="field"><label htmlFor="refund-id">{t({ en: "Consumption transaction ID", zh: "原消费流水编号" })}</label><input id="refund-id" required type="number" min={1} step={1} disabled={action.busy} value={refundID} onChange={(event) => setRefundID(event.target.value)} /></div> : null}
          <div className="field"><label htmlFor="operation-reason">{t({ en: "Reason", zh: "操作原因" })}</label><input id="operation-reason" required minLength={selectedReferral !== null ? 3 : 1} maxLength={500} disabled={action.busy} value={reason} onChange={(event) => setReason(event.target.value)} /></div>
          <div className="button-row"><button className="button primary" disabled={action.busy || !validReason} aria-busy={action.busy} type="submit">{pendingSubmit(selectedReferral !== null ? { en: "Confirm reconciliation", zh: "确认核对奖励" } : creditAction === "grant" ? { en: "Confirm grant", zh: "确认发放" } : { en: "Confirm credit return", zh: "确认退回积分" })}</button><button className="button" type="button" onClick={dismiss}>{t({ en: "Cancel", zh: "取消" })}</button></div>
        </form>}
      </ResponsiveDialog>
    </ConsoleGate>
  </>;
}
export function OperatorConsole({ section }: { section: OperatorSection }) {
  if (section === "integrations") return <IntegrationsConsole />;
  return <Suspense fallback={<CustomerPageLoading />}><OperatorConsoleInner key={section} section={section} /></Suspense>;
}
