"use client";

import { useCallback, useEffect, useId, useRef, useState, type FormEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { getOperatorAccess, updateOperatorMembership, grantCredits, deductOperatorCredits, listCreditTransactions, listOperator, type MembershipPatch, type OperatorAccess, type PageResult, type CreditTransaction, type OperatorOrder } from "@/lib/operations-api";
import { ConsoleError, newIntentKey, useConsoleAction, useConsoleResource } from "./console-kit";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatDate } from "@/lib/format";
import { adminTag } from "@/lib/admin-presentation";
import { DataTable, type DataColumn } from "./data-table";
import { StatusBadge } from "./status-badge";
import { ResponsiveDialog } from "./responsive-dialog";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";

export type UserDetailTab = "overview" | "payments" | "credits";
import { customerTabs, customerTabNames, creditSourceName, signedCredits } from "@/lib/customer-records";
import { Select } from "./select";
import { adminSubscriptionPresentation } from "@/lib/admin-subscription";
type UserTarget = { id: string; email: string; credits: number };
const planOptions = ["automatic", "free", "starter", "pro", "premium", "lifetime"] as const;
function localDate(value: string | null | undefined) {
 if (!value) return "";
 const date = new Date(value);
 return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0,16);
}

function UserEditor({ user, token, tab, close, onSaved, action, access }: { user: UserTarget; token: string; tab: "membership" | "credits"; close: () => void; onSaved: (message: string) => void; action: ReturnType<typeof useConsoleAction>; access: {data:OperatorAccess|null;loading:boolean;error:unknown;refresh:()=>Promise<void>} }) {
 const { t } = useI18n();
 const [plan, setPlan] = useState<MembershipPatch["plan"]>("automatic");
 const [expires, setExpires] = useState("");
 const [operation, setOperation] = useState<"grant" | "deduct">("grant");
 const [amount, setAmount] = useState("50");
 const [reason, setReason] = useState("");
 const intent = useRef({ payload: "", key: "" });
 useEffect(() => {
  if (tab !== "membership" || !access.data) return;
  const override = access.data.override;
  const active = override && (!override.expires_at || new Date(override.expires_at).getTime() > Date.now());
  setPlan(active && planOptions.includes(override.plan as MembershipPatch["plan"]) ? override.plan as MembershipPatch["plan"] : "automatic");
  setExpires(active ? localDate(override.expires_at) : "");
 }, [access.data, tab]);
 const reasonLength = Array.from(reason.trim()).length;
 const timed = plan === "starter" || plan === "pro" || plan === "premium";
 const validExpiry = !timed || Boolean(expires && new Date(expires).getTime() > Date.now());
 const numericAmount = Number(amount);
 const validAmount = Number.isSafeInteger(numericAmount) && numericAmount > 0 && numericAmount <= 1000000;
 const validCreditExpiry = operation !== "grant" || !expires || new Date(expires).getTime() > Date.now();
 function submit(event: FormEvent) {
  event.preventDefault();
  if (reasonLength < 3 || reasonLength > 500 || (tab === "membership" ? !validExpiry || !access.data : !validAmount || !validCreditExpiry)) return;
  const membership: MembershipPatch = { plan, ...(timed && expires ? { expires_at: new Date(expires).toISOString() } : {}), reason: reason.trim() };
  const payload = tab === "membership" ? { user: user.id, membership } : { user: user.id, operation, amount: numericAmount, reason: reason.trim(), expires };
  const serialized = JSON.stringify(payload);
  if (intent.current.payload !== serialized) intent.current = { payload: serialized, key: newIntentKey() };
  const key = intent.current.key;
  void action.run(() => tab === "membership" ? updateOperatorMembership(token, user.id, membership, key)
   : operation === "grant" ? grantCredits(token, { user_id: user.id, amount: numericAmount, ...(expires ? { expires_at: new Date(expires).toISOString() } : {}), reason: reason.trim(), idempotency_key: key })
   : deductOperatorCredits(token, user.id, numericAmount, reason.trim(), key), () => {
    onSaved(t(tab === "membership" ? { en: "Membership updated and recorded.", zh: "会员级别已更新，操作已记录。" } : { en: "Credits adjusted and recorded.", zh: "积分已调整，操作已记录。" }));
    setReason(""); intent.current={payload:"",key:""};
   });
 }
 const planLabel = (value: typeof planOptions[number]) => value === "automatic" ? t({ en: "Follow payment subscription", zh: "跟随支付订阅（自动）" }) : value === "free" ? "Free" : value === "lifetime" ? "Lifetime" : value[0].toUpperCase() + value.slice(1);
 return <form className={styles.dialogForm} onSubmit={submit}>
  {tab === "membership" ? <>
   {access.loading ? <p role="status"><PendingLabel>{t({ en: "Loading membership…", zh: "正在读取会员信息…" })}</PendingLabel></p> : null}
   <div className="field"><label htmlFor="user-plan">{t({ en: "Membership level", zh: "会员级别" })}</label><Select id="user-plan" value={plan} disabled={access.loading || !access.data || action.busy} onChange={event => { setPlan(event.target.value as MembershipPatch["plan"]); setExpires(""); }}>{planOptions.map(value => <option key={value} value={value}>{planLabel(value)}</option>)}</Select></div>
   {timed ? <div className="field"><label htmlFor="user-member-expiry">{t({ en: "Access until (local time)", zh: "有效期至（当前设备时间）" })}</label><input id="user-member-expiry" type="datetime-local" value={expires} onChange={event => setExpires(event.target.value)} required disabled={action.busy} /><small>{t({ en: "When this expires, access follows the payment subscription again.", zh: "到期后，会员级别恢复为支付订阅决定的状态。" })}</small></div> : null}
   <p className={styles.dialogNote}>{plan === "automatic" ? t({ en: "Removes the manual override. Access follows the current payment subscription.", zh: "移除手动设置，恢复当前支付订阅对应的会员级别。" }) : t({ en: "Changes access on this site only. Existing subscription charges continue unchanged; this does not count as a payment.", zh: "仅调整本站使用权限。已有订阅仍按原规则扣费，本次操作不计为支付。" })}</p>
  </> : <>
   <div className="field"><label htmlFor="user-credit-operation">{t({ en: "Adjustment", zh: "调整方式" })}</label><Select id="user-credit-operation" value={operation} onChange={event => { setOperation(event.target.value as "grant" | "deduct"); setExpires(""); }} disabled={action.busy}><option value="grant">{t({ en: "Add credits", zh: "增加积分" })}</option><option value="deduct">{t({ en: "Deduct credits", zh: "扣减积分" })}</option></Select></div>
   <div className="field"><label htmlFor="user-credit-amount">{t({ en: "Credits", zh: "调整数量" })}</label><input id="user-credit-amount" type="number" required min={1} max={1000000} step={1} value={amount} onChange={event => setAmount(event.target.value)} disabled={action.busy} /></div>
   {operation === "grant" ? <div className="field"><label htmlFor="user-credit-expiry">{t({ en: "Expires at (optional, local time)", zh: "到期时间（选填，当前设备时间）" })}</label><input id="user-credit-expiry" type="datetime-local" value={expires} onChange={event => setExpires(event.target.value)} disabled={action.busy} /></div> : null}
   <p className={styles.dialogNote}>{t({ en: "Adjustments are recorded in credit history. A deduction cannot exceed the available balance.", zh: "每次调整都会记录积分流水；扣减数量不能超过实际可用余额。" })}</p>
  </>}
  <div className="field"><label htmlFor="user-edit-reason">{t({ en: "Reason", zh: "操作原因" })}</label><textarea id="user-edit-reason" rows={2} required minLength={3} maxLength={500} value={reason} onChange={event => setReason(event.target.value)} disabled={action.busy} placeholder={t({ en: "At least 3 characters", zh: "至少填写 3 个字" })} /></div>
  <div className="button-row"><button type="submit" className="button primary" disabled={action.busy || reasonLength < 3 || reasonLength > 500 || (tab === "membership" ? access.loading || !access.data || !validExpiry : !validAmount || !validCreditExpiry)} aria-busy={action.busy}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "保存中…" })}</PendingLabel> : t({ en: "Save changes", zh: "保存更改" })}</button><button type="button" className="button" onClick={close} disabled={action.busy}>{t({ en: "Cancel", zh: "取消" })}</button></div>
 </form>;
}

function UserHistory({ token, userID, kind }: { token: string; userID: string; kind: "credits" | "payments" }) {
 const { t, locale } = useI18n();
 const [page, setPage] = useState(1);
 const loader = useCallback(async (current: string): Promise<PageResult<CreditTransaction | OperatorOrder>> => kind === "credits"
  ? listCreditTransactions(current, page, userID, true)
  : listOperator<OperatorOrder>(current, "orders", page, "", "", { user_id: userID }), [kind, page, userID]);
 const resource = useConsoleResource(token, loader);
 const date = (value: string) => formatDate(value, locale === "zh" ? "zh-CN" : "en-US");
 const creditColumns: DataColumn<CreditTransaction>[] = [
  { key: "created_at", heading: t({ en: "Date", zh: "时间" }), cell: row => date(row.created_at) },
  { key: "source", heading: t({ en: "Source", zh: "变更来源" }), cell: row => <span className={styles.cellIdentity}><span>{t(creditSourceName(row))}</span>{row.actor_id ? <small title={row.actor_id}>{t({ en: "Operator", zh: "操作人" })}: {row.actor_id}</small> : null}</span> },
  { key: "amount", heading: t({ en: "Change", zh: "变动积分" }), align: "end", cell: row => <strong className={row.amount > 0 ? styles.creditIncrease : styles.creditDecrease}>{signedCredits(row.amount)}</strong> },
  { key: "balance_after", heading: t({ en: "Balance after", zh: "变更后余额" }), align: "end", cell: row => row.balance_after.toLocaleString() },
  { key: "reason", heading: t({ en: "Reason / reference", zh: "原因与关联记录" }), cell: row => <span className={styles.cellIdentity}><span title={row.reason}>{row.reason || "—"}</span><small title={row.source_id}>{row.source_id || `#${row.id}`}</small></span> }
 ];
 const paymentColumns: DataColumn<OperatorOrder>[] = [
  { key: "created_at", heading: t({ en: "Created", zh: "创建时间" }), cell: row => <span className={styles.cellIdentity}><span>{date(row.created_at)}</span><small title={row.id}>{row.id}</small></span> },
  { key: "purchase", heading: t({ en: "Purchase", zh: "购买内容" }), cell: row => <span className={styles.cellIdentity}><span>{row.product_type === "credits" ? t({ en: "Credit pack", zh: "积分充值" }) : row.product_type === "lifetime" ? "Lifetime" : row.plan || t({ en: "Subscription", zh: "订阅套餐" })}</span>{row.billing_interval ? <small>{t(row.billing_interval === "yearly" ? { en: "Yearly", zh: "按年" } : { en: "Monthly", zh: "按月" })}</small> : null}</span> },
  { key: "amount_due", heading: t({ en: "Order total", zh: "订单金额" }), align: "end", cell: row => formatBillingAmount(row.amount_due ?? row.amount, row.currency, locale) },
  { key: "amount", heading: t({ en: "Paid", zh: "实付金额" }), align: "end", cell: row => formatBillingAmount(row.status === "paid" ? row.amount : null, row.currency, locale) },
  { key: "status", heading: t({ en: "Status", zh: "状态" }), cell: row => { const status = adminTag("orders", "status", row.status); return <StatusBadge tone={status?.tone || "neutral"}>{status ? t(status.label) : row.status}</StatusBadge>; } }
 ];
 return <div className={styles.userHistory}>
  <p className={styles.historyDescription}>{t(kind === "credits" ? { en: "All changes in one ledger: grants, purchases, referral rewards, usage and expiration.", zh: "统一记录管理员调整、邀请奖励、充值、周期发放、活动赠送、消费与到期。" } : { en: "Checkout creation and verified payment outcomes. An unpaid checkout does not grant access.", zh: "查看收银台创建与实际支付结果；创建订单不代表已付款或已获得权益。" })}</p>
  <ConsoleError error={resource.error} retry={() => void resource.refresh()} />
  {kind === "credits" ? <DataTable caption={t(customerTabNames.credits)} columns={creditColumns} rows={(resource.data?.items || []) as CreditTransaction[]} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading records…", zh: "正在读取记录…" })} empty={t({ en: "No credit changes yet.", zh: "暂无积分变更记录。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} /> : <DataTable caption={t(customerTabNames.payments)} columns={paymentColumns} rows={(resource.data?.items || []) as OperatorOrder[]} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading records…", zh: "正在读取记录…" })} empty={t({ en: "No checkout or payment records yet.", zh: "暂无订单或支付记录。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} />}
 </div>;
}

function UserDialog({ user, token, tab, onClose, onSaved }: { user: UserTarget; token: string; tab: UserDetailTab; onClose: () => void; onSaved: (message: string) => void }) {
 const { t, locale } = useI18n();
 const id = useId();
 const action = useConsoleAction(token);
 const [current, setCurrent] = useState<UserDetailTab>(tab);
 const [editor, setEditor] = useState<"membership" | "credits" | null>(null);
 const loader = useCallback((currentToken: string) => getOperatorAccess(currentToken, user.id), [user.id]);
 const access = useConsoleResource(token, loader);
 function saved(message: string) { action.setMessage(message); setEditor(null); void access.refresh(); onSaved(message); }
 function selectTab(next: UserDetailTab) { if (!action.busy) { setEditor(null); setCurrent(next); action.setMessage(""); } }
 const profile = access.data?.user;
 const membership = access.data?.membership;
 const subscription = adminSubscriptionPresentation(access.data?.subscription);
 const plan = adminTag("users", "plan", membership?.plan || profile?.plan);
 return <ResponsiveDialog open size="wide" onClose={onClose} dismissible={!action.busy} title={t({ en: "Manage customer", zh: "用户管理" })} closeLabel={t({ en: "Close", zh: "关闭" })}>
  <div className={styles.userDialog}>
   <div className={styles.userSummary}><strong>{profile?.email || user.email}</strong><span>{user.id}</span></div>
   <div className={styles.tabs} role="tablist" aria-label={t({ en: "Customer details", zh: "用户详情" })} onKeyDown={event => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key) || action.busy) return;
    event.preventDefault();
    const index = customerTabs.indexOf(current);
    const next = event.key === "Home" ? 0 : event.key === "End" ? customerTabs.length - 1 : (index + (event.key === "ArrowRight" ? 1 : -1) + customerTabs.length) % customerTabs.length;
    selectTab(customerTabs[next]);
    event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]')[next]?.focus();
   }}>{customerTabs.map(value => <button type="button" role="tab" id={`${id}-${value}`} aria-controls={`${id}-panel`} tabIndex={current === value ? 0 : -1} key={value} aria-selected={current === value} disabled={action.busy} onClick={() => selectTab(value)}>{t(customerTabNames[value])}</button>)}</div>
   <ConsoleError error={access.error} retry={() => void access.refresh()} /><ConsoleError error={action.error} />
   {action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
   <div id={`${id}-panel`} role="tabpanel" aria-labelledby={`${id}-${current}`}>
    {current === "overview" ? <section className={styles.customerOverview}>
     <dl className={styles.customerFacts}>
      <div><dt>{t({ en: "Joined", zh: "注册时间" })}</dt><dd>{profile ? formatDate(profile.created_at, locale === "zh" ? "zh-CN" : "en-US") : "—"}</dd></div>
      <div><dt>{t({ en: "Membership", zh: "会员级别" })}</dt><dd>{plan ? <StatusBadge tone={plan.tone}>{t(plan.label)}</StatusBadge> : "—"}{membership?.source === "manual" ? <small className={styles.membershipSource}>{t({ en: "Manual access", zh: "手动授权" })}</small> : null}</dd></div>
      <div><dt>{t({ en: "Available credits", zh: "可用积分" })}</dt><dd>{profile ? profile.credits.toLocaleString() : "—"}</dd></div>
     </dl>
     {subscription ? <section className={styles.customerSubscription} aria-label={t({ en: "Payment subscription", zh: "支付订阅" })}>
      <div className={styles.subscriptionHeading}><h3>{t({ en: "Payment subscription", zh: "支付订阅" })}</h3><StatusBadge tone={subscription.status.tone}>{t(subscription.status.label)}</StatusBadge></div>
      <dl className={styles.userFacts}>
       <dt>{t({ en: "Plan", zh: "订阅套餐" })}</dt><dd>{t(subscription.plan)}</dd>
       {subscription.periodEnd ? <><dt>{t(subscription.endLabel)}</dt><dd>{formatDate(subscription.periodEnd, locale)}</dd></> : null}
       {subscription.upcoming ? <><dt>{t({ en: "Upcoming plan", zh: "下期套餐" })}</dt><dd>{t(subscription.upcoming.plan)}</dd><dt>{t({ en: "Starts", zh: "生效时间" })}</dt><dd>{subscription.upcoming.effectiveAt ? formatDate(subscription.upcoming.effectiveAt, locale) : t({ en: "Date pending", zh: "时间待确认" })}</dd></> : null}
      </dl>
     </section> : null}
     {!editor ? <div className="button-row"><button type="button" className="button" disabled={access.loading || !profile} onClick={() => setEditor("membership")}>{t({ en: "Adjust membership", zh: "调整会员级别" })}</button><button type="button" className="button" disabled={access.loading || !profile} onClick={() => setEditor("credits")}>{t({ en: "Adjust credits", zh: "调整积分" })}</button></div> : <section className={styles.customerEditor} aria-label={t(editor === "credits" ? { en: "Adjust credits", zh: "调整积分" } : { en: "Adjust membership", zh: "调整会员级别" })}><h3>{t(editor === "credits" ? { en: "Adjust credits", zh: "调整积分" } : { en: "Adjust membership", zh: "调整会员级别" })}</h3><UserEditor key={`editor-${editor}`} user={{ ...user, credits: profile?.credits ?? user.credits }} token={token} tab={editor} close={() => setEditor(null)} action={action} access={access} onSaved={saved} /></section>}
    </section> : <UserHistory key={`history-${current}`} token={token} userID={user.id} kind={current} />}
   </div>
  </div>
 </ResponsiveDialog>;
}
export function OperatorUserDialog({ user, ...props }: { user: UserTarget | null; token: string; tab: UserDetailTab; onClose: () => void; onSaved: (message: string) => void }) {
 return user ? <UserDialog key={`${props.token}:${user.id}`} user={user} {...props} /> : null;
}
