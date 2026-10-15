"use client";

import Link from "next/link";
import { Suspense, useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useSearchParams } from "next/navigation";
import adminStyles from "./admin-shell.module.css";
import { Panel } from "./ui";
import { Select } from "./select";
import { ConsoleError, ConsoleGate, consoleStyles as styles, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { useI18n } from "@/lib/i18n";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatDate } from "@/lib/format";
import { getBusinessSettings, getOperatorOverview, grantCredits, listCreditTransactions, listOperator, refundCredits, retryReferralReward, saveBusinessSettings, type BusinessSettings, type PageResult, type OperatorOverview as OverviewData, type OperatorReportFilters } from "@/lib/operations-api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { adminSections, adminSectionNames } from "@/lib/admin-navigation";
import { ContentAdminList } from "./content-admin";
import { WebsiteConsole } from "./website-console";
import { AdminPageHeader } from "./admin-page-header";
import { DataTable, type DataColumn } from "./data-table";
import { StatusBadge } from "./status-badge";
import { ResponsiveDialog } from "./responsive-dialog";
import { PendingLabel } from "./pending-label";
import { CustomerPageLoading } from "./customer-page-loading";
import { OperatorOverview } from "./operator-overview";
import { OperatorUserDialog, type UserDetailTab } from "./operator-user-dialog";
import { adminTag, adminActionLabel } from "@/lib/admin-presentation";

export const operatorSections = adminSections;
export type OperatorSection = typeof operatorSections[number];
type Row = Record<string, unknown>;
const names = adminSectionNames;
const columns: Partial<Record<OperatorSection, Array<[string, { en: string; zh: string }]>>> = {
  users: [["email", { en: "Customer", zh: "用户" }], ["plan", { en: "Membership", zh: "会员级别" }], ["credits", { en: "Credits", zh: "积分" }], ["created_at", { en: "Joined", zh: "注册时间" }]],
  orders: [["created_at",{en:"Created",zh:"创建时间"}],["user_id",{en:"Customer",zh:"用户"}],["product_type",{en:"Purchase",zh:"购买内容"}],["amount",{en:"Paid amount",zh:"实付金额"}],["status",{en:"Status",zh:"支付状态"}],["id",{en:"Reference",zh:"交易编号"}]],
  subscriptions: [["user_id", { en: "Account", zh: "账号" }], ["plan", { en: "Plan", zh: "套餐" }], ["status", { en: "Status", zh: "状态" }], ["provider_subscription_id", { en: "Subscription", zh: "订阅编号" }], ["period_end", { en: "Period end", zh: "本期截止" }]],
  referrals: [["referrer_id", { en: "Inviter", zh: "邀请人" }], ["referee_id", { en: "Invitee", zh: "受邀人" }], ["status", { en: "Status", zh: "状态" }], ["reward_credits", { en: "Reward", zh: "奖励积分" }], ["expires_at", { en: "Deadline", zh: "奖励截止" }]],
  credits: [["id", { en: "Transaction", zh: "流水编号" }], ["user_id", { en: "Account", zh: "账号" }], ["kind", { en: "Type", zh: "类型" }], ["amount", { en: "Credits", zh: "积分变动" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]],
  audit: [["status", { en: "Outcome", zh: "结果" }], ["actor_id", { en: "Operator", zh: "操作人" }], ["action", { en: "Action", zh: "操作" }], ["target_id", { en: "Target", zh: "对象" }], ["reason", { en: "Reason", zh: "原因" }], ["created_at", { en: "Recorded", zh: "时间" }]]
};

function OperatorConsoleInner({ section }: { section: Exclude<OperatorSection, "integrations" | "articles"> }) {
  const { t, locale } = useI18n();
  const params = useSearchParams();
  const { session, ready } = useConsoleSession();
  const token = session?.user.role === "admin" ? session.token : "";
  const [page, setPage] = useState(1);
  const [reportFilters, setReportFilters] = useState<OperatorReportFilters>({ days: 30, timezone: "Asia/Shanghai", mode: "live" });
  const [selectedUser, setSelectedUser] = useState<{ id: string; email: string; credits: number } | null>(null);
  const [userTab, setUserTab] = useState<UserDetailTab>("overview");
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
 const [paymentMode,setPaymentMode]=useState("");const [from,setFrom]=useState("");const [to,setTo]=useState("");
 const [selectedAudit,setSelectedAudit]=useState<Row|null>(null);
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
  useEffect(() => { setPage(1); setSearch(""); setQuery(""); setStatus(""); setReason(""); setSelectedReferral(null); setSelectedUser(null); setCreditAction(null); setSettings(null); setTarget(""); setRefundID(""); setExpiry(""); intent.current = { payload: "", key: "" }; }, [section, token]);
  useEffect(() => { if (section === "credits") { const userID = params.get("user_id") || ""; setTarget(userID); setQuery(userID); setSearch(userID); } }, [params, section, token]);
  const loader = useCallback(async (currentToken: string): Promise<unknown> => {
    if (section === "overview") return getOperatorOverview(currentToken, undefined, reportFilters);
    if (section === "settings") return getBusinessSettings(currentToken);
    if (section === "credits") return listCreditTransactions(currentToken, page, query, true);
    return listOperator<Row>(currentToken, section, page, query, status,section==="orders"?{mode:paymentMode,from:from?new Date(`${from}T00:00:00`).toISOString():undefined,to:to?new Date(new Date(`${to}T00:00:00`).getTime()+86400000).toISOString():undefined}:{});
  }, [page, query, section, status, reportFilters,paymentMode,from,to]);
  const resource = useConsoleResource(token, loader);
  useEffect(() => { if (section === "settings" && resource.data) setSettings(resource.data as BusinessSettings); }, [resource.data, section]);
  const result = resource.data && section !== "overview" && section !== "settings" ? resource.data as PageResult<Row> : null;
  const overview = section === "overview" ? resource.data as OverviewData | null : null;
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
 if(section==="audit"&&key==="action")return t(adminActionLabel(String(value)));
 if(key==="product_type")return value==="subscription"?t({en:"Subscription",zh:"订阅"}):value==="credits"?t({en:"Credit package",zh:"积分包"}):value==="lifetime"?"Lifetime":"—";
    if (section === "users" && key === "email") return <span className={adminStyles.cellIdentity}><span title={String(value)}>{String(value)}</span><small title={String(row.id)}>{String(row.id)}</small></span>;
    const tag = adminTag(section, key, value);
    if (tag) return section === "users" && key === "plan" ? <span className={adminStyles.cellIdentity}><span><StatusBadge tone={tag.tone}>{t(tag.label)}</StatusBadge></span>{typeof row.membership === "object" && row.membership !== null && "source" in row.membership && row.membership.source === "manual" ? <small>{t({ en: "Manual access", zh: "手动授权" })}</small> : null}</span> : <StatusBadge tone={tag.tone}>{t(tag.label)}</StatusBadge>;
    if (key === "amount" && section === "orders") return typeof value === "number" ? formatBillingAmount(value, String(row.currency || ""), locale === "zh" ? "zh-CN" : "en-US") : t({ en: "Unknown", zh: "未知" });
    if (value === null || value === undefined || value === "") return "—";
    if (key.endsWith("_at") || key === "period_end") return formatDate(String(value), locale === "zh" ? "zh-CN" : "en-US");
    if (["user_id", "referrer_id", "referee_id"].includes(key)) {
      const email = row[key === "user_id" ? "email" : key === "referrer_id" ? "referrer_email" : "referee_email"];
      return <button type="button" className={adminStyles.identityButton} onClick={()=>{setSelectedUser({id:String(value),email:typeof email==="string"?email:String(value),credits:0});setUserTab(section==="orders"?"payments":"overview");}}><span className={adminStyles.cellIdentity}>{typeof email === "string" && email ? <span title={email}>{email}</span> : null}<small title={String(value)}>{String(value)}</small></span></button>;
    }
    if (key === "id" || key === "actor_id" || key === "target_id" || key === "provider_subscription_id") return <span className={adminStyles.mono} title={String(value)}>{String(value)}</span>;
    return String(value);
  }
  const tableColumns: DataColumn<Row>[] = (columns[section] || []).map(([key, label]) => ({ key, heading: t(label), cell: row => cell(row, key), align: ["amount", "credits", "reward_credits"].includes(key) ? "end" : "start" }));
  function editUser(row: Row, tab: UserDetailTab) { setSelectedUser({ id: String(row.id), email: String(row.email), credits: Number(row.credits) }); setUserTab(tab); }
  if (section === "users") tableColumns.push({ key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <button className={`button ${adminStyles.tableAction}`} type="button" onClick={() => editUser(row,"overview")}>{t({en:"Manage",zh:"管理用户"})}</button> });
  if (section === "audit") tableColumns.push({key:"details",heading:t({en:"Details",zh:"变更详情"}),cell:row=><button type="button" className={`button ${adminStyles.tableAction}`} onClick={()=>setSelectedAudit(row)}>{t({en:"View changes",zh:"查看变更"})}</button>});
  if (section === "referrals") tableColumns.push({ key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => row.status === "activated" ? <button className={`button ${adminStyles.tableAction}`} type="button" disabled={action.busy} onClick={() => { setSelectedReferral(Number(row.id)); setReason(""); }}>{t({ en: "Reconcile reward", zh: "核对奖励" })}</button> : <span className={adminStyles.mutedCell} title={t({ en: "Reward reconciliation is available after activation.", zh: "邀请激活后可核对奖励。" })}>—</span> });
  const searchLabel = section === "credits" ? { en: "Account ID", zh: "账号 ID" }
    : section === "audit" ? { en: "Operator, action, target or reason", zh: "操作人、操作、对象或原因" }
    : section === "referrals" ? { en: "Email, account ID or referral code", zh: "邮箱、账号 ID 或邀请码" }
    : section === "users" ? { en: "Email, username or account ID", zh: "邮箱、用户名或账号 ID" }
    : { en: "Email or reference ID", zh: "邮箱或关联编号" };
  const dialogOpen = creditAction !== null || selectedReferral !== null;
  const pendingSubmit = (label: { en: string; zh: string }) => action.busy ? <PendingLabel>{t({ en: "Processing…", zh: "处理中…" })}</PendingLabel> : t(label);
  const settingField = (key: "brand_name" | "description" | "support_email" | "support_url", label: { en: string; zh: string }) => settings ? <div className="field" key={key}><label htmlFor={`setting-${key}`}>{t(label)}</label><input id={`setting-${key}`} type={key === "support_email" ? "email" : key === "support_url" ? "url" : "text"} required={key === "brand_name"} pattern={key === "support_url" ? "https://.+" : undefined} maxLength={key === "brand_name" ? 100 : key === "description" ? 500 : key === "support_url" ? 1024 : 255} value={settings[key]} onChange={(event) => setSettings({ ...settings, [key]: event.target.value })} /></div> : null;
  return <>
    <AdminPageHeader section={(["users","orders","referrals"].includes(section)?"users":section)} onRefresh={() => void resource.refresh()} busy={resource.loading} disabled={!token || action.busy} actions={section === "credits" ? <><button type="button" className="button" disabled={action.busy} onClick={() => { setCreditAction("refund"); setReason(""); }}>{t({ en: "Return credits", zh: "退回积分" })}</button><button type="button" className="button primary" disabled={action.busy} onClick={() => { setCreditAction("grant"); setReason(""); }}>{t({ en: "Grant credits", zh: "发放积分" })}</button></> : undefined} />
    <ConsoleGate session={session} ready={ready} admin>
      {["users","orders","referrals"].includes(section)?<nav className={adminStyles.tabs} aria-label={t({en:"Customer operations",zh:"用户管理视图"})}>{[["users","/admin/users",{en:"Users",zh:"用户列表"}],["orders","/admin/users?view=payments",{en:"Payments",zh:"支付记录"}],["referrals","/admin/users?view=invitations",{en:"Invitations",zh:"邀请记录"}]].map(([value,href,label])=><Link key={value as string} href={href as string} aria-current={section===value?"page":undefined}>{t(label as {en:string;zh:string})}</Link>)}</nav>:null}
      <div className={styles.stack}>
        <ConsoleError error={resource.error} retry={() => void resource.refresh()} />
        {!dialogOpen ? <ConsoleError error={action.error} /> : null}
        {action.message ? <p className={styles.success} role="status">{action.message}</p> : null}
        {resource.loading && (section === "overview" || section === "settings") ? <p role="status"><PendingLabel>{t({ en: "Loading current records…", zh: "正在读取最新记录…" })}</PendingLabel></p> : null}
        {overview && !Array.isArray(overview.series) ? <p className={adminStyles.dialogNote} role="status">{t({ en: "Daily reporting requires the updated API service. Update and restart the local backend to load these metrics.", zh: "每日统计需要新版接口。请更新并重启本地后端后读取这些指标。" })}</p> : null}
        {section === "overview" ? <OperatorOverview data={overview && overview.mode === reportFilters.mode && overview.timezone === reportFilters.timezone && overview.series?.length === reportFilters.days ? overview : null} filters={reportFilters} onFilters={setReportFilters} loading={resource.loading} /> : null}
        {section !== "overview" && section !== "settings" ? <section className={adminStyles.records} aria-label={t(names[section])}>
          <div className={adminStyles.listControls}>
          <form className={adminStyles.toolbar} onSubmit={submitSearch} role="search" aria-label={t(names[section])}>
            <div className={adminStyles.search}><div className="field"><label className="sr-only" htmlFor="operator-search">{t(searchLabel)}</label><input id="operator-search" type="search" placeholder={t(searchLabel)} value={search} onChange={(event) => setSearch(event.target.value)} /></div><button className="button" type="submit" disabled={resource.loading}>{t({ en: "Search", zh: "搜索" })}</button></div>
            {section === "referrals" ? <div className="field"><label className="sr-only" htmlFor="referral-status">{t({ en: "Activation status", zh: "激活状态" })}</label><Select id="referral-status" value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}><option value="">{t({ en: "All statuses", zh: "全部状态" })}</option><option value="pending">{t({ en: "Pending", zh: "待激活" })}</option><option value="activated">{t({ en: "Activated", zh: "已激活" })}</option><option value="expired">{t({ en: "Expired", zh: "已过期" })}</option></Select></div> : null}
            {section==="orders"?<><div className="field"><label className="sr-only" htmlFor="payment-mode">{t({en:"Payment environment",zh:"支付环境"})}</label><Select id="payment-mode" value={paymentMode} onChange={event=>{setPaymentMode(event.target.value);setPage(1);}}><option value="">{t({en:"All environments",zh:"全部环境"})}</option><option value="live">{t({en:"Live",zh:"正式支付"})}</option><option value="test">{t({en:"Test",zh:"测试支付"})}</option></Select></div><div className="field"><label className="sr-only" htmlFor="payment-status">{t({en:"Payment status",zh:"支付状态"})}</label><Select id="payment-status" value={status} onChange={event=>{setStatus(event.target.value);setPage(1);}}><option value="">{t({en:"All statuses",zh:"全部状态"})}</option><option value="paid">{t({en:"Paid",zh:"已支付"})}</option><option value="unpaid">{t({en:"Unpaid",zh:"未支付"})}</option><option value="failed">{t({en:"Failed",zh:"失败"})}</option><option value="expired">{t({en:"Expired",zh:"已过期"})}</option></Select></div><div className="field"><label htmlFor="payment-from">{t({en:"From (local time)",zh:"开始日期（本机时间）"})}</label><input id="payment-from" type="date" value={from} max={to||undefined} onChange={event=>{setFrom(event.target.value);setPage(1);}}/></div><div className="field"><label htmlFor="payment-to">{t({en:"Through",zh:"结束日期"})}</label><input id="payment-to" type="date" value={to} min={from||undefined} onChange={event=>{setTo(event.target.value);setPage(1);}}/></div></>:null}
            {section==="audit"?<div className="field"><label className="sr-only" htmlFor="audit-outcome">{t({en:"Outcome",zh:"处理结果"})}</label><Select id="audit-outcome" value={status} onChange={event=>{setStatus(event.target.value);setPage(1);}}><option value="">{t({en:"All outcomes",zh:"全部结果"})}</option><option value="succeeded">{t({en:"Succeeded",zh:"已完成"})}</option><option value="failed">{t({en:"Failed",zh:"失败"})}</option><option value="pending">{t({en:"Processing",zh:"处理中"})}</option></Select></div>:null}
            {query || status || paymentMode || from || to ? <button className={`button ${adminStyles.resetFilter}`} type="button" onClick={() => { setPage(1); setSearch(""); setQuery(""); setStatus("");setPaymentMode("");setFrom("");setTo(""); }}>{t({ en: "Reset", zh: "重置" })}</button> : null}
            <span className={adminStyles.recordCount} role="status">{result ? t({ en: `${result.total.toLocaleString()} records`, zh: `共 ${result.total.toLocaleString()} 条` }) : "—"}</span>
          </form>
          </div>
          <DataTable caption={t(names[section])} columns={tableColumns} rows={result?.items || []} rowKey={row => String(row.id ?? row.user_id)} loading={resource.loading} loadingLabel={t({ en: "Loading records…", zh: "正在读取记录…" })} empty={t({ en: "No matching records. Try another search.", zh: "暂无匹配记录，可尝试其他搜索条件。" })} error={resource.error ? t({ en: "Records could not be loaded. Use the retry button above.", zh: "暂时无法读取记录，请点击上方重试。" }) : undefined} pagination={result ? { page: result.page, limit: result.limit, total: result.total, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (current, pages) => t({ en: `Page ${current} of ${pages}`, zh: `第 ${current} / ${pages} 页` }) } : undefined} />
        </section> : null}
        {section === "settings" && settings ? <Panel title={t({ en: "General settings", zh: "常规设置" })}><form className={adminStyles.settingsForm} onSubmit={saveSettings}>
          <fieldset className={adminStyles.formSection} disabled={action.busy}><legend>{t({ en: "Service details", zh: "服务资料" })}</legend><p>{t({ en: "These values describe the API service. Website branding, homepage copy and sharing cards are configured in content/site.ts and NEXT_PUBLIC_APP_NAME, then applied with a frontend rebuild.", zh: "以下资料用于 API 服务。网站品牌、首页文案和分享卡片在 content/site.ts 与 NEXT_PUBLIC_APP_NAME 中配置，重新构建前端后生效。" })}</p><div className={styles.formGrid}>{settingField("brand_name", { en: "Service name (API)", zh: "服务名称（API）" })}{settingField("description", { en: "Service description (API)", zh: "服务介绍（API）" })}</div></fieldset>
          <fieldset className={adminStyles.formSection} disabled={action.busy}><legend>{t({ en: "Help & support", zh: "帮助与支持" })}</legend><p>{t({ en: "Support channels take effect after saving.", zh: "支持渠道保存后生效。" })}</p><div className={styles.formGrid}>{settingField("support_email", { en: "Support email", zh: "支持邮箱" })}{settingField("support_url", { en: "Support URL (HTTPS)", zh: "支持页面（HTTPS）" })}</div></fieldset>
          <div className={adminStyles.formFooter}><div className="field"><label htmlFor="settings-reason">{t({ en: "Change reason", zh: "修改原因" })}</label><input id="settings-reason" required minLength={3} maxLength={500} placeholder={t({ en: "At least 3 characters; recorded in the audit log", zh: "至少填写 3 个字，将记录到审计日志" })} value={reason} disabled={action.busy} onChange={(event) => setReason(event.target.value)} /></div><button className="button primary" type="submit" aria-busy={action.busy} disabled={action.busy || !validReason || !settings.brand_name.trim()}>{pendingSubmit({ en: "Save changes", zh: "保存更改" })}</button></div>
        </form></Panel> : null}
      </div>
      <ResponsiveDialog open={selectedAudit!==null} size="wide" onClose={()=>setSelectedAudit(null)} title={t({en:"Audit details",zh:"审计详情"})} closeLabel={t({en:"Close",zh:"关闭"})}>{selectedAudit?<div className={adminStyles.dialogForm}><dl className={adminStyles.userFacts}>{["actor_id","action","target_id","reason","created_at"].map(key=><div key={key}><dt>{key}</dt><dd>{String(selectedAudit[key]||"—")}</dd></div>)}</dl><pre className={adminStyles.auditDetails}>{(()=>{try{return JSON.stringify(JSON.parse(String(selectedAudit.details||"{}")),null,2);}catch{return String(selectedAudit.details||"—");}})()}</pre></div>:null}</ResponsiveDialog>
      <OperatorUserDialog user={selectedUser} token={token} tab={userTab} onClose={() => setSelectedUser(null)} onSaved={message => { action.setMessage(message); void resource.refresh(); }} />
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
function CustomerManagement(){const params=useSearchParams();const view=params.get("view");const section=view==="payments"?"orders":view==="invitations"?"referrals":"users";return <OperatorConsoleInner key={section} section={section}/>;}
export function OperatorConsole({ section }: { section: OperatorSection }) {
 if(section==="users")return <Suspense fallback={<CustomerPageLoading/>}><CustomerManagement/></Suspense>;
 if(section==="articles")return <ContentAdminList/>;
 if(section==="integrations"||section==="settings")return <WebsiteConsole/>;
 return <Suspense fallback={<CustomerPageLoading/>}><OperatorConsoleInner key={section} section={section}/></Suspense>;
}
