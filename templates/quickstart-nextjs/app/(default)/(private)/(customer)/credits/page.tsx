"use client";
import { LocalizedLink as Link } from "@/components/localized-link";
import { useCallback, useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { DataTable } from "@/components/data-table";
import { Panel } from "@/components/ui";
import { ConsoleError, ConsoleGate, consoleStyles as styles, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { getCreditBalance, listCreditTransactions } from "@/lib/operations-api";
import { useI18n } from "@/lib/i18n";
import { usePageReturn } from "@/components/use-page-return";
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
  const refreshHistory = history.refresh;
  const refreshBalance = balance.refresh;
  const update = useCallback(() => { void refreshHistory(); void refreshBalance(); }, [refreshHistory, refreshBalance]);
  usePageReturn(update);
  const labels: Record<string, { en: string; zh: string }> = { opening: { en: "Opening balance", zh: "期初余额" }, grant: { en: "Received", zh: "获得积分" }, consume: { en: "Used", zh: "消费积分" }, refund: { en: "Returned", zh: "积分退回" }, expire: { en: "Expired", zh: "积分过期" }, expiry: { en: "Expired", zh: "积分过期" } };
  return <SiteShell eyebrow={t({ en: "Your credits", zh: "我的积分" })} title={t({ en: "Credit history", zh: "积分明细" })} description={t({ en: "See your available balance, upcoming expiry and the history of every grant, purchase and use.", zh: "查看可用积分、即将到期的权益，以及奖励、购买和使用产生的每一笔记录。" })} actions={<div className="button-row"><Link className="button primary" href="/pricing#one-time">{t({ en: "Get more credits", zh: "购买积分" })}</Link></div>}>
    <ConsoleGate session={session} ready={ready}><div className={styles.stack}>
      <ConsoleError error={balance.error} retry={() => void balance.refresh()} />
      <div className={styles.metrics}><div className={styles.metric}><span>{t({ en: "Available credits", zh: "可用积分" })}</span><strong>{balance.data?.balance ?? "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Expiring within 30 days", zh: "30 天内到期" })}</span><strong>{balance.data?.expiring_credits ?? "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Next expiry", zh: "最近到期时间" })}</span><p>{balance.data?.next_expiry_at ? formatDate(balance.data.next_expiry_at, locale === "zh" ? "zh-CN" : "en-US") : balance.data ? t({ en: "No upcoming expiry", zh: "暂无到期计划" }) : "—"}</p></div></div>
      <Panel title={t({ en: "Credit history", zh: "积分明细" })} subtitle={t({ en: "Credits with an earlier expiry are used first. Refunds remain linked to the original consumption.", zh: "使用时优先消耗临近到期的积分，退回记录与原消费记录关联。" })}>
        <DataTable caption={t({ en: "Credit history", zh: "积分明细" })}
          columns={[
            {key:"id",heading:t({en:"Reference",zh:"流水编号"}),cell:item=>`#${item.id}`},
            {key:"kind",heading:t({en:"Type",zh:"类型"}),cell:item=>labels[item.kind] ? t(labels[item.kind]) : item.kind},
            {key:"amount",heading:t({en:"Change",zh:"积分变动"}),align:"end",cell:item=>`${item.amount>0?"+":""}${item.amount}`},
            {key:"reason",heading:t({en:"Reason",zh:"说明"}),cell:item=><>{item.reason||item.source}{item.related_transaction_id ? ` · #${item.related_transaction_id}` : ""}</>},
            {key:"expires",heading:t({en:"Expires",zh:"到期时间"}),cell:item=>item.expires_at ? formatDate(item.expires_at,locale==="zh"?"zh-CN":"en-US") : "—"},
            {key:"date",heading:t({en:"Date",zh:"时间"}),cell:item=>formatDate(item.created_at,locale==="zh"?"zh-CN":"en-US")}
          ]} rows={history.data?.items||[]} rowKey={item=>item.id} loading={history.loading}
          loadingLabel={t({en:"Loading history…",zh:"正在加载明细…"})}
          empty={t({en:"Your first credit transaction will appear here.",zh:"还没有积分记录，获得或使用积分后会显示在这里。"})}
          error={history.error ? <ConsoleError error={history.error} retry={()=>void history.refresh()} /> : undefined}
          pagination={{page,limit:history.data?.limit||20,total:history.data?.total||0,onPage:setPage,
            previousLabel:t({en:"Previous",zh:"上一页"}),nextLabel:t({en:"Next",zh:"下一页"}),
            label:(current,pages)=>t({en:`Page ${current} of ${pages}`,zh:`第 ${current} / ${pages} 页`})}} />
      </Panel>
    </div></ConsoleGate>
  </SiteShell>;
}
