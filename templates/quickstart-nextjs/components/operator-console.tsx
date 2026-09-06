"use client";

import Link from "next/link";
import { Suspense, useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useSearchParams } from "next/navigation";
import { SiteShell } from "./site-shell";
import { Panel } from "./ui";
import { ConsoleError, ConsoleGate, Pagination, consoleStyles as styles, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { useI18n } from "@/lib/i18n";
import { formatDate } from "@/lib/format";
import { getBusinessSettings, getOperatorOverview, grantCredits, listCreditTransactions, listOperator, refundCredits, retryReferralReward, saveBusinessSettings, type BusinessSettings, type PageResult } from "@/lib/operations-api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";

export const operatorSections = ["overview", "users", "orders", "subscriptions", "referrals", "credits", "settings", "audit"] as const;
export type OperatorSection = typeof operatorSections[number];
type Row = Record<string, unknown>;
const names: Record<OperatorSection, { en: string; zh: string }> = {
  overview: { en: "Overview", zh: "运营概况" }, users: { en: "Users", zh: "用户" }, orders: { en: "Payments", zh: "支付记录" }, subscriptions: { en: "Subscriptions", zh: "订阅" }, referrals: { en: "Invitations", zh: "邀请" }, credits: { en: "Credits", zh: "积分" }, settings: { en: "Site settings", zh: "网站配置" }, audit: { en: "Audit log", zh: "操作记录" }
};
const columns: Partial<Record<OperatorSection, Array<[string, { en: string; zh: string }]>>> = {
  users: [["email", { en: "Email", zh: "邮箱" }], ["id", { en: "Account ID", zh: "账号 ID" }], ["role", { en: "Role", zh: "角色" }], ["credits", { en: "Credits", zh: "积分" }], ["created_at", { en: "Joined", zh: "注册时间" }]],
  orders: [["id", { en: "Payment reference", zh: "支付编号" }], ["user_id", { en: "Account", zh: "账号" }], ["amount", { en: "Amount", zh: "金额" }], ["status", { en: "Status", zh: "状态" }], ["created_at", { en: "Recorded", zh: "记录时间" }]],
  subscriptions: [["user_id", { en: "Account", zh: "账号" }], ["plan", { en: "Plan", zh: "套餐" }], ["status", { en: "Status", zh: "状态" }], ["provider_subscription_id", { en: "Subscription", zh: "订阅编号" }], ["period_end", { en: "Period end", zh: "本期截止" }]],
  referrals: [["referrer_id", { en: "Inviter", zh: "邀请人" }], ["referee_id", { en: "Invitee", zh: "受邀人" }], ["status", { en: "Status", zh: "状态" }], ["reward_credits", { en: "Reward", zh: "奖励积分" }], ["expires_at", { en: "Deadline", zh: "奖励截止" }]],
  credits: [["id", { en: "Transaction", zh: "流水编号" }], ["user_id", { en: "Account", zh: "账号" }], ["kind", { en: "Type", zh: "类型" }], ["amount", { en: "Credits", zh: "积分变动" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]],
  audit: [["actor_id", { en: "Operator", zh: "操作人" }], ["action", { en: "Action", zh: "操作" }], ["target_id", { en: "Target", zh: "对象" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]]
};

function OperatorConsoleInner({ section }: { section: OperatorSection }) {
  const { t } = useI18n();
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
  const [selectedReferral, setSelectedReferral] = useState<number | null>(null);
  const [settings, setSettings] = useState<BusinessSettings | null>(null);
  const action = useConsoleAction(token);
  const intent = useRef({ payload: "", key: "" });
  function intentFor(payload: unknown) {
    const serialized = JSON.stringify(payload);
    if (intent.current.payload !== serialized || !intent.current.key) intent.current = { payload: serialized, key: newIntentKey() };
    return intent.current.key;
  }
  useEffect(() => { setPage(1); setSearch(""); setQuery(""); setStatus(""); setReason(""); setSelectedReferral(null); setSettings(null); setTarget(""); setRefundID(""); setExpiry(""); intent.current = { payload: "", key: "" }; }, [section, token]);
  useEffect(() => { if (section === "credits") { const userID = params.get("user_id") || ""; setTarget(userID); setQuery(userID); setSearch(userID); } }, [params, section]);
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
  function success(message: string) { action.setMessage(message); setReason(""); intent.current = { payload: "", key: "" }; setSelectedReferral(null); void resource.refresh(); }
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
    void action.run(() => retryReferralReward(token, id, reason.trim(), intentFor({ operation: "reward", id, reason: reason.trim() })), () => success(t({ en: "Reward reconciliation completed. Existing rewards are not granted twice.", zh: "奖励核对完成，已发放的奖励不会重复发放。" })));
  }
  function saveSettings(event: FormEvent) {
    event.preventDefault(); if (!settings || !validReason || !settings.brand_name.trim()) return;
    const payload = { ...settings, reason: reason.trim() };
    void action.run(() => saveBusinessSettings(token, payload, intentFor(payload)), () => { window.dispatchEvent(new Event(SITE_SETTINGS_EVENT)); success(t({ en: "Site settings saved.", zh: "网站配置已保存。" })); });
  }
  function display(row: Row, key: string): string {
    const value = row[key];
    if (key === "amount" && section === "orders") {
      if (typeof value !== "number") return t({ en: "Unknown", zh: "未知" });
      return `${value} ${String(row.currency || "").toUpperCase()}`;
    }
    if (value === null || value === undefined || value === "") return "—";
    if (key.endsWith("_at") || key === "period_end") return formatDate(String(value));
    const labels: Record<string, { en: string; zh: string }> = { admin: { en: "Administrator", zh: "管理员" }, user: { en: "Customer", zh: "用户" }, pending: { en: "Pending", zh: "待激活" }, activated: { en: "Activated", zh: "已激活" }, expired: { en: "Expired", zh: "已过期" }, grant: { en: "Grant", zh: "发放" }, consume: { en: "Consumption", zh: "消费" }, refund: { en: "Refund", zh: "退回" }, active: { en: "Active", zh: "有效" }, paid: { en: "Paid", zh: "已支付" }, trialing: { en: "Trial", zh: "试用中" }, canceled: { en: "Canceled", zh: "已取消" } };
    return labels[String(value)] ? t(labels[String(value)]) : String(value);
  }
  return <SiteShell eyebrow={t({ en: "Operator console", zh: "运营后台" })} title={t(names[section])} description={t({ en: "Manage customers, payment records and product operations with an auditable history.", zh: "管理用户、支付记录与产品运营，每次权益调整都有据可查。" })} actions={<button className="button" type="button" disabled={!token || resource.loading} onClick={() => void resource.refresh()}>{t({ en: "Refresh", zh: "刷新" })}</button>}>
    <ConsoleGate session={session} ready={ready} admin>
      <div className={styles.stack}>
        <nav className={styles.tabs} aria-label={t({ en: "Operator navigation", zh: "运营后台导航" })}>{operatorSections.map((item) => <Link key={item} href={item === "overview" ? "/admin" : `/admin/${item}`} aria-current={section === item ? "page" : undefined}>{t(names[item])}</Link>)}</nav>
        <ConsoleError error={resource.error} retry={() => void resource.refresh()} />
        <ConsoleError error={action.error} />
        {action.message ? <p className={styles.success} role="status">{action.message}</p> : null}
        {resource.loading ? <p role="status">{t({ en: "Loading current records…", zh: "正在读取最新记录…" })}</p> : null}
        {overview ? <div className={styles.metrics}>{([
          ["users", { en: "Registered accounts", zh: "注册用户" }], ["active_subscriptions", { en: "Active subscriptions", zh: "有效订阅" }], ["referrals", { en: "Invitations", zh: "邀请总数" }], ["pending_referrals", { en: "Awaiting activation", zh: "待激活邀请" }], ["activated_referrals", { en: "Activated invitations", zh: "已激活邀请" }]
        ] as const).map(([key, label]) => <div className={styles.metric} key={key}><span>{t(label)}</span><strong>{overview[key] ?? "—"}</strong></div>)}</div> : null}
        {section !== "overview" && section !== "settings" ? <Panel title={t(names[section])} subtitle={section === "orders" ? t({ en: "Recorded checkout and invoice payments. Amounts are displayed in the currency's smallest unit; unavailable values stay unknown.", zh: "已记录的收银台与账单支付，金额以货币最小单位显示；缺失金额保留为未知。" }) : undefined}>
          <form className={styles.toolbar} onSubmit={submitSearch}>
            <div className={styles.search}><div className="field"><label htmlFor="operator-search">{section === "credits" ? t({ en: "Account ID", zh: "账号 ID" }) : t({ en: "Search email or ID", zh: "搜索邮箱或编号" })}</label><input id="operator-search" value={search} onChange={(event) => setSearch(event.target.value)} /></div><button className="button" type="submit">{t({ en: "Search", zh: "搜索" })}</button></div>
            {section === "referrals" ? <div className="field"><label htmlFor="referral-status">{t({ en: "Status", zh: "状态" })}</label><select id="referral-status" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}><option value="">{t({ en: "All", zh: "全部" })}</option><option value="pending">{t({ en: "Pending", zh: "待激活" })}</option><option value="activated">{t({ en: "Activated", zh: "已激活" })}</option><option value="expired">{t({ en: "Expired", zh: "已过期" })}</option></select></div> : null}
          </form>
          {result?.items.length ? <div className={styles.tableWrap}><table className="table"><thead><tr>{columns[section]?.map(([key, label]) => <th key={key}>{t(label)}</th>)}{section === "users" || section === "referrals" ? <th>{t({ en: "Action", zh: "操作" })}</th> : null}</tr></thead><tbody>{result.items.map((row, index) => <tr key={String(row.id ?? index)}>{columns[section]?.map(([key]) => <td key={key}>{display(row, key)}</td>)}{section === "users" ? <td><Link className="button" href={`/admin/credits?user_id=${encodeURIComponent(String(row.id))}`}>{t({ en: "Manage credits", zh: "管理积分" })}</Link></td> : section === "referrals" ? <td><button className="button" type="button" disabled={row.status !== "activated" || action.busy} onClick={() => { setSelectedReferral(Number(row.id)); setReason(""); }}>{t({ en: "Reconcile reward", zh: "核对奖励" })}</button></td> : null}</tr>)}</tbody></table></div> : !resource.loading && !resource.error ? <p className="empty-state">{t({ en: "No matching records.", zh: "暂无匹配记录。" })}</p> : null}
          {result ? <Pagination result={result} loading={resource.loading} onPage={setPage} /> : null}
        </Panel> : null}
        {selectedReferral !== null ? <Panel title={t({ en: `Reconcile invitation #${selectedReferral}`, zh: `核对邀请 #${selectedReferral} 的奖励` })} subtitle={t({ en: "Replays the stored reward for an already activated invitation. This does not change qualification or reward amount.", zh: "仅重新核对已激活邀请记录的奖励，不改变资格或奖励金额。" })}><form onSubmit={retryReward}><div className="field"><label htmlFor="reward-reason">{t({ en: "Reason", zh: "操作原因" })}</label><input id="reward-reason" required minLength={3} maxLength={500} placeholder={t({ en: "At least 3 characters", zh: "至少填写 3 个字" })} value={reason} onChange={(event) => setReason(event.target.value)} /></div><div className="button-row"><button className="button primary" disabled={action.busy || !validReason} type="submit">{t({ en: "Confirm reconciliation", zh: "确认核对奖励" })}</button><button className="button" type="button" onClick={() => setSelectedReferral(null)}>{t({ en: "Cancel", zh: "取消" })}</button></div></form></Panel> : null}
        {section === "credits" ? <>
          <Panel title={t({ en: "Grant credits", zh: "发放积分" })} subtitle={t({ en: "A positive grant with a recorded operator and reason. Leave expiry empty for credits that do not expire.", zh: "每次发放都会记录操作人和原因。到期时间留空表示永久有效。" })}><form className={styles.formGrid} onSubmit={grant}>
            <div className="field"><label htmlFor="grant-user">{t({ en: "Account ID", zh: "账号 ID" })}</label><input id="grant-user" required value={target} onChange={(event) => setTarget(event.target.value)} /></div>
            <div className="field"><label htmlFor="grant-amount">{t({ en: "Credit amount", zh: "发放积分数" })}</label><input id="grant-amount" type="number" min={1} step={1} max={1000000} required value={amount} onChange={(event) => setAmount(event.target.value)} /></div>
            <div className="field"><label htmlFor="grant-expiry">{t({ en: "Expires at (optional)", zh: "到期时间（选填）" })}</label><input id="grant-expiry" type="datetime-local" value={expiry} onChange={(event) => setExpiry(event.target.value)} /></div>
            <div className="field"><label htmlFor="grant-reason">{t({ en: "Reason", zh: "发放原因" })}</label><input id="grant-reason" required maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} /></div>
            <div className={styles.full}><button className="button primary" type="submit" disabled={action.busy || !validReason}>{t({ en: "Confirm grant", zh: "确认发放" })}</button></div>
          </form></Panel>
          <Panel title={t({ en: "Return consumed credits", zh: "退回已消费积分" })} subtitle={t({ en: "Enter the original consumption transaction. Each consumption can only be refunded once.", zh: "填写原消费流水编号。同一笔消费只能退回一次。" })}><form className={styles.formGrid} onSubmit={refund}>
            <div className="field"><label htmlFor="refund-id">{t({ en: "Consumption transaction ID", zh: "原消费流水编号" })}</label><input id="refund-id" required type="number" min={1} step={1} value={refundID} onChange={(event) => setRefundID(event.target.value)} /></div>
            <div className="field"><label htmlFor="refund-reason">{t({ en: "Reason", zh: "退回原因" })}</label><input id="refund-reason" required maxLength={500} value={reason} onChange={(event) => setReason(event.target.value)} /></div>
            <div className={styles.full}><button className="button" type="submit" disabled={action.busy || !validReason}>{t({ en: "Confirm credit refund", zh: "确认退回积分" })}</button></div>
          </form></Panel>
        </> : null}
        {section === "settings" && settings ? <Panel title={t({ en: "Brand and support", zh: "品牌与支持" })} subtitle={t({ en: "These public business settings take effect after saving. Provider credentials remain in server deployment configuration.", zh: "这些公开业务配置保存后生效，服务商密钥仍由后端部署配置管理。" })}><form className={styles.formGrid} onSubmit={saveSettings}>
          {([ ["brand_name", { en: "Product name", zh: "产品名称" }], ["description", { en: "Product description", zh: "产品介绍" }], ["support_email", { en: "Support email", zh: "支持邮箱" }], ["support_url", { en: "Support URL (HTTPS)", zh: "支持页面（HTTPS）" }] ] as const).map(([key, label]) => <div className="field" key={key}><label htmlFor={`setting-${key}`}>{t(label)}</label><input id={`setting-${key}`} type={key === "support_email" ? "email" : key === "support_url" ? "url" : "text"} required={key === "brand_name"} pattern={key === "support_url" ? "https://.+" : undefined} maxLength={key === "brand_name" ? 100 : key === "description" ? 500 : key === "support_url" ? 1024 : 255} value={settings[key]} onChange={(event) => setSettings({ ...settings, [key]: event.target.value })} /></div>)}
          <div className="field"><label htmlFor="export-cost">{t({ en: "Credits per note export", zh: "每次笔记导出消耗积分" })}</label><input id="export-cost" type="number" required min={1} max={1000000} step={1} value={settings.export_credit_cost} onChange={(event) => setSettings({ ...settings, export_credit_cost: Number(event.target.value) })} /></div>
          <div className="field"><label htmlFor="settings-reason">{t({ en: "Change reason", zh: "修改原因" })}</label><input id="settings-reason" required minLength={3} maxLength={500} placeholder={t({ en: "At least 3 characters", zh: "至少填写 3 个字" })} value={reason} onChange={(event) => setReason(event.target.value)} /></div>
          <div className={styles.full}><button className="button primary" type="submit" disabled={action.busy || !validReason || !settings.brand_name.trim()}>{t({ en: "Save site settings", zh: "保存网站配置" })}</button></div>
        </form></Panel> : null}
      </div>
    </ConsoleGate>
  </SiteShell>;
}
export function OperatorConsole({ section }: { section: OperatorSection }) { return <Suspense fallback={null}><OperatorConsoleInner key={section} section={section} /></Suspense>; }
