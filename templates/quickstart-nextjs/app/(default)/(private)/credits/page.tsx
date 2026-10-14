"use client";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { Panel } from "@/components/ui";
import { ConsoleError, ConsoleGate, Pagination, consoleStyles as styles, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { getCreditBalance, listCreditTransactions } from "@/lib/operations-api";
import { useI18n } from "@/lib/i18n";
import { formatDate } from "@/lib/format";
export default function CreditsPage() {
  const { t, locale } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const [page, setPage] = useState(1);
  useEffect(() => { setPage(1); }, [token]);
  const balance = useConsoleResource(token, getCreditBalance);
  const loader = useCallback((key: string) => listCreditTransactions(key, page), [page]);
  const history = useConsoleResource(token, loader);
  const labels: Record<string, { en: string; zh: string }> = { opening: { en: "Opening balance", zh: "期初余额" }, grant: { en: "Received", zh: "获得积分" }, consume: { en: "Used", zh: "消费积分" }, refund: { en: "Returned", zh: "积分退回" }, expire: { en: "Expired", zh: "积分过期" }, expiry: { en: "Expired", zh: "积分过期" } };
  return <SiteShell eyebrow={t({ en: "Your credits", zh: "我的积分" })} title={t({ en: "Every credit, accounted for.", zh: "积分收支，清楚可查。" })} description={t({ en: "See your available balance, upcoming expiry and the history of every grant, purchase and use.", zh: "查看可用积分、即将到期的权益，以及奖励、购买和使用产生的每一笔记录。" })} actions={<div className="button-row"><Link className="button primary" href="/billing">{t({ en: "Get more credits", zh: "购买积分" })}</Link><Link className="button" href="/notes">{t({ en: "Use credits", zh: "使用积分" })}</Link></div>}>
    <ConsoleGate session={session} ready={ready}><div className={styles.stack}>
      <ConsoleError error={balance.error} retry={() => void balance.refresh()} />
      <div className={styles.metrics}><div className={styles.metric}><span>{t({ en: "Available credits", zh: "可用积分" })}</span><strong>{balance.data?.balance ?? "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Expiring within 30 days", zh: "30 天内到期" })}</span><strong>{balance.data?.expiring_credits ?? "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Next expiry", zh: "最近到期时间" })}</span><p>{balance.data?.next_expiry_at ? formatDate(balance.data.next_expiry_at, locale === "zh" ? "zh-CN" : "en-US") : balance.data ? t({ en: "No upcoming expiry", zh: "暂无到期计划" }) : "—"}</p></div></div>
      <Panel title={t({ en: "Credit history", zh: "积分明细" })} subtitle={t({ en: "Credits with an earlier expiry are used first. Refunds remain linked to the original consumption.", zh: "使用时优先消耗临近到期的积分，退回记录与原消费记录关联。" })} actions={<button className="button" type="button" disabled={history.loading || balance.loading} onClick={() => { void history.refresh(); void balance.refresh(); }}>{t({ en: "Refresh", zh: "刷新" })}</button>}>
        <ConsoleError error={history.error} retry={() => void history.refresh()} />
        {history.loading ? <p role="status">{t({ en: "Loading history…", zh: "正在加载明细…" })}</p> : null}
        {history.data?.items.length ? <div className={styles.tableWrap}><table className="table"><thead><tr>{[t({ en: "Reference", zh: "流水编号" }), t({ en: "Type", zh: "类型" }), t({ en: "Change", zh: "积分变动" }), t({ en: "Reason", zh: "说明" }), t({ en: "Expires", zh: "到期时间" }), t({ en: "Date", zh: "时间" })].map((name) => <th key={name}>{name}</th>)}</tr></thead><tbody>{history.data.items.map((item) => <tr key={item.id}><td>#{item.id}</td><td>{labels[item.kind] ? t(labels[item.kind]) : item.kind}</td><td>{item.amount > 0 ? "+" : ""}{item.amount}</td><td>{item.reason || item.source}{item.related_transaction_id ? ` · #${item.related_transaction_id}` : ""}</td><td>{item.expires_at ? formatDate(item.expires_at, locale === "zh" ? "zh-CN" : "en-US") : "—"}</td><td>{formatDate(item.created_at, locale === "zh" ? "zh-CN" : "en-US")}</td></tr>)}</tbody></table></div> : !history.loading && !history.error ? <p className="empty-state">{t({ en: "Your first credit transaction will appear here.", zh: "还没有积分记录，获得或使用积分后会显示在这里。" })}</p> : null}
        {history.data ? <Pagination result={history.data} loading={history.loading} onPage={setPage} /> : null}
      </Panel>
    </div></ConsoleGate>
  </SiteShell>;
}
