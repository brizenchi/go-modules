"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { getOperatorAccess, updateOperatorMembership, grantCredits, deductOperatorCredits, listCreditTransactions, listOperator, refundCredits, retryReferralReward, type MembershipPatch, type OperatorAccess, type PageResult } from "@/lib/operations-api";
import { ConsoleError, newIntentKey, useConsoleAction, useConsoleResource } from "./console-kit";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatDate } from "@/lib/format";
import { adminTag } from "@/lib/admin-presentation";
import { DataTable } from "./data-table";
import { StatusBadge } from "./status-badge";
import { ResponsiveDialog } from "./responsive-dialog";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";

export type UserDetailTab = "membership" | "credits" | "payments" | "invitations";
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
  <ConsoleError error={action.error} />
  {tab === "membership" ? <>
   <ConsoleError error={access.error} retry={() => void access.refresh()} />
   {access.loading ? <p role="status"><PendingLabel>{t({ en: "Loading membership…", zh: "正在读取会员信息…" })}</PendingLabel></p> : null}
   <div className="field"><label htmlFor="user-plan">{t({ en: "Membership level", zh: "会员级别" })}</label><select id="user-plan" value={plan} disabled={access.loading || !access.data || action.busy} onChange={event => { setPlan(event.target.value as MembershipPatch["plan"]); setExpires(""); }}>{planOptions.map(value => <option key={value} value={value}>{planLabel(value)}</option>)}</select></div>
   {timed ? <div className="field"><label htmlFor="user-member-expiry">{t({ en: "Access until (local time)", zh: "有效期至（当前设备时间）" })}</label><input id="user-member-expiry" type="datetime-local" value={expires} onChange={event => setExpires(event.target.value)} required disabled={action.busy} /><small>{t({ en: "When this expires, access follows the payment subscription again.", zh: "到期后，会员级别恢复为支付订阅决定的状态。" })}</small></div> : null}
   <p className={styles.dialogNote}>{plan === "automatic" ? t({ en: "Removes the manual override. Access follows the current payment subscription.", zh: "移除手动设置，恢复当前支付订阅对应的会员级别。" }) : t({ en: "Changes access on this site only. Existing subscription charges continue unchanged; this does not count as a payment.", zh: "仅调整本站使用权限。已有订阅仍按原规则扣费，本次操作不计为支付。" })}</p>
  </> : <>
   <div className="field"><label htmlFor="user-credit-operation">{t({ en: "Adjustment", zh: "调整方式" })}</label><select id="user-credit-operation" value={operation} onChange={event => { setOperation(event.target.value as "grant" | "deduct"); setExpires(""); }} disabled={action.busy}><option value="grant">{t({ en: "Add credits", zh: "增加积分" })}</option><option value="deduct">{t({ en: "Deduct credits", zh: "扣减积分" })}</option></select></div>
   <div className="field"><label htmlFor="user-credit-amount">{t({ en: "Credits", zh: "调整数量" })}</label><input id="user-credit-amount" type="number" required min={1} max={1000000} step={1} value={amount} onChange={event => setAmount(event.target.value)} disabled={action.busy} /></div>
   {operation === "grant" ? <div className="field"><label htmlFor="user-credit-expiry">{t({ en: "Expires at (optional, local time)", zh: "到期时间（选填，当前设备时间）" })}</label><input id="user-credit-expiry" type="datetime-local" value={expires} onChange={event => setExpires(event.target.value)} disabled={action.busy} /></div> : null}
   <p className={styles.dialogNote}>{t({ en: "Adjustments are recorded in credit history. A deduction cannot exceed the available balance.", zh: "每次调整都会记录积分流水；扣减数量不能超过实际可用余额。" })}</p>
  </>}
  <div className="field"><label htmlFor="user-edit-reason">{t({ en: "Reason", zh: "操作原因" })}</label><textarea id="user-edit-reason" rows={2} required minLength={3} maxLength={500} value={reason} onChange={event => setReason(event.target.value)} disabled={action.busy} placeholder={t({ en: "At least 3 characters", zh: "至少填写 3 个字" })} /></div>
  <div className="button-row"><button type="submit" className="button primary" disabled={action.busy || reasonLength < 3 || reasonLength > 500 || (tab === "membership" ? access.loading || !access.data || !validExpiry : !validAmount || !validCreditExpiry)} aria-busy={action.busy}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "保存中…" })}</PendingLabel> : t({ en: "Save changes", zh: "保存更改" })}</button><button type="button" className="button" onClick={close} disabled={action.busy}>{t({ en: "Cancel", zh: "取消" })}</button></div>
 </form>;
}

type HistoryRow=Record<string,unknown>;
function UserHistory({token,userID,kind,version,action,onSaved}:{token:string;userID:string;kind:"credits"|"payments"|"invitations";version:number;action:ReturnType<typeof useConsoleAction>;onSaved:(message:string)=>void}) {
 const {t,locale}=useI18n();const [page,setPage]=useState(1);const [pending,setPending]=useState<HistoryRow|null>(null);const [reason,setReason]=useState("");const intent=useRef({payload:"",key:""});
 const loader=useCallback(async(current:string):Promise<PageResult<HistoryRow>>=>{
  void version;
  if(kind==="credits"){const result=await listCreditTransactions(current,page,userID,true);return {...result,items:result.items.map(row=>({...row}))};}
  return listOperator<HistoryRow>(current,kind==="payments"?"orders":"referrals",page,"","",{user_id:userID});
 },[kind,page,userID,version]);
 const resource=useConsoleResource(token,loader);
 const cols=kind==="credits"?[["created_at",{en:"Date",zh:"时间"}],["kind",{en:"Type",zh:"类型"}],["amount",{en:"Credits",zh:"积分变动"}],["reason",{en:"Reason",zh:"原因"}]]:kind==="payments"?[["created_at",{en:"Date",zh:"时间"}],["product_type",{en:"Purchase",zh:"购买内容"}],["amount",{en:"Amount",zh:"实付金额"}],["status",{en:"Status",zh:"支付状态"}],["livemode",{en:"Environment",zh:"环境"}]]:[["referee_email",{en:"Invited user",zh:"受邀用户"}],["status",{en:"Status",zh:"状态"}],["reward_credits",{en:"Reward",zh:"奖励积分"}],["created_at",{en:"Date",zh:"注册时间"}]];
 function display(row:HistoryRow,key:string){
  const value=row[key];const section=kind==="payments"?"orders":kind==="invitations"?"referrals":"credits";const tag=adminTag(section,key,value);if(tag)return <StatusBadge tone={tag.tone}>{t(tag.label)}</StatusBadge>;
  if(key==="amount"&&kind==="payments")return formatBillingAmount(typeof value==="number"?value:null,String(row.currency||""),locale==="zh"?"zh-CN":"en-US");
  if(key==="created_at")return formatDate(String(value),locale==="zh"?"zh-CN":"en-US");
  if(key==="referee_email")return String(value||row.referee_id||"—");return value===null||value===undefined||value===""?"—":String(value);
 }
 function confirm(event:FormEvent){event.preventDefault();if(!pending||Array.from(reason.trim()).length<3)return;const payload={id:pending.id,kind,reason:reason.trim()};const serialized=JSON.stringify(payload);if(intent.current.payload!==serialized)intent.current={payload:serialized,key:newIntentKey()};void action.run(()=>kind==="credits"?refundCredits(token,{transaction_id:Number(pending.id),reason:reason.trim(),idempotency_key:intent.current.key}):retryReferralReward(token,Number(pending.id),reason.trim(),intent.current.key),()=>{setPending(null);setReason("");intent.current={payload:"",key:""};onSaved(t({en:"Operation completed and recorded.",zh:"操作已完成并记录。"}));void resource.refresh();});}
 const columns=cols.map(([key,label])=>({key:key as string,heading:t(label as {en:string;zh:string}),cell:(row:HistoryRow)=>display(row,key as string),align:(key==="amount"||key==="reward_credits"?"end":"start") as "end"|"start"}));
 if(kind!=="payments")columns.push({key:"actions",heading:t({en:"Action",zh:"操作"}),cell:row=>(kind==="credits"&&row.kind==="consume")||(kind==="invitations"&&row.status==="activated")?<button type="button" className="button" disabled={action.busy} onClick={()=>{setPending(row);setReason("");}}>{t(kind==="credits"?{en:"Return credits",zh:"退回积分"}:{en:"Reconcile",zh:"核对奖励"})}</button>:<span>—</span>,align:"start"});
 return <div className={styles.userHistory}><ConsoleError error={resource.error} retry={()=>void resource.refresh()}/><DataTable caption={t({en:"User history",zh:"用户记录"})} columns={columns} rows={resource.data?.items||[]} rowKey={row=>String(row.id)} loading={resource.loading} loadingLabel={t({en:"Loading records…",zh:"正在读取记录…"})} empty={t({en:"No records yet.",zh:"暂无记录。"})} pagination={resource.data?{...resource.data,onPage:setPage,previousLabel:t({en:"Previous",zh:"上一页"}),nextLabel:t({en:"Next",zh:"下一页"}),label:(p,n)=>`${p} / ${n}`}:undefined}/>{pending?<form className={styles.dialogForm} onSubmit={confirm}><p className={styles.dialogNote}>{t({en:`Confirm operation on record #${String(pending.id)}.`,zh:`确认处理记录 #${String(pending.id)}。`})}</p><div className="field"><label htmlFor="history-reason">{t({en:"Reason",zh:"操作原因"})}</label><input id="history-reason" required minLength={3} maxLength={500} disabled={action.busy} value={reason} onChange={event=>setReason(event.target.value)}/></div><div className="button-row"><button type="submit" className="button primary" disabled={action.busy||Array.from(reason.trim()).length<3}>{t({en:"Confirm",zh:"确认操作"})}</button><button type="button" className="button" disabled={action.busy} onClick={()=>setPending(null)}>{t({en:"Cancel",zh:"取消"})}</button></div></form>:null}</div>;
}
function UserDialog({user,token,tab,onClose,onSaved}:{user:UserTarget;token:string;tab:UserDetailTab;onClose:()=>void;onSaved:(message:string)=>void}) {
 const {t}=useI18n();const action=useConsoleAction(token);const [current,setCurrent]=useState<UserDetailTab>(tab);const [version,setVersion]=useState(0);
 const loader=useCallback((currentToken:string)=>getOperatorAccess(currentToken,user.id),[user.id]);const access=useConsoleResource(token,loader);
 function saved(message:string){action.setMessage(message);setVersion(value=>value+1);void access.refresh();onSaved(message);}
 const profile=access.data?.user;const membership=access.data?.membership;const subscription=access.data?.subscription;
 const tabs: Array<[UserDetailTab,{en:string;zh:string}]>=[["membership",{en:"Membership",zh:"会员与订阅"}],["credits",{en:"Credits",zh:"积分"}],["payments",{en:"Payments",zh:"支付记录"}],["invitations",{en:"Invitations",zh:"邀请奖励"}]];
 return <ResponsiveDialog open size="wide" onClose={onClose} dismissible={!action.busy} title={t({en:"Manage customer",zh:"用户管理"})} closeLabel={t({en:"Close",zh:"关闭"})}>{close=><div className={styles.userDialog}>
  <div className={styles.userSummary}><strong>{profile?.email||user.email}</strong><span>{user.id}</span><span>{membership?.plan ? `${membership.plan} · `:""}{t({en:`Available credits: ${profile ? profile.credits.toLocaleString() : "…"}`,zh:`可用积分：${profile ? profile.credits.toLocaleString() : "…"}`})}</span></div>
  <div className={styles.tabs} role="tablist" aria-label={t({en:"Customer details",zh:"用户详情"})}>{tabs.map(([value,label])=><button type="button" role="tab" key={value} aria-selected={current===value} disabled={action.busy} onClick={()=>setCurrent(value)}>{t(label)}</button>)}</div>
  <ConsoleError error={access.error} retry={()=>void access.refresh()}/><ConsoleError error={action.error}/>{action.message?<p role="status" className={styles.dialogNote}>{action.message}</p>:null}
  {current==="membership"&&subscription?<dl className={styles.userFacts}><dt>{t({en:"Payment subscription",zh:"支付订阅"})}</dt><dd>{subscription.plan} · {subscription.status}</dd><dt>{t({en:"Period end",zh:"本期截止"})}</dt><dd>{subscription.period_end?formatDate(subscription.period_end):"—"}</dd></dl>:null}
  {current==="membership"||current==="credits"?<UserEditor key={current} user={{...user,credits:profile?.credits??user.credits}} token={token} tab={current} close={close} action={action} access={access} onSaved={saved}/>:null}
  {current!=="membership"?<UserHistory key={current} token={token} userID={user.id} kind={current} version={version} action={action} onSaved={saved}/>:null}
 </div>}</ResponsiveDialog>;
}
export function OperatorUserDialog({user,...props}:{user:UserTarget|null;token:string;tab:UserDetailTab;onClose:()=>void;onSaved:(message:string)=>void}){return user?<UserDialog key={`${props.token}:${user.id}`} user={user} {...props}/>:null;}
