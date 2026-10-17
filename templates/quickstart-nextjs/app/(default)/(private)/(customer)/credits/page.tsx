"use client";
import { LocalizedLink as Link } from "@/components/localized-link";
import { useCallback, useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { DataTable } from "@/components/data-table";
import { CreditCheckoutNotice } from "@/components/credit-checkout-notice";
import { Panel } from "@/components/ui";
import { ConsoleError, ConsoleGate, consoleStyles as styles, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { getCreditBalance, listCreditTransactions } from "@/lib/operations-api";
import { useI18n } from "@/lib/i18n";
import { usePageReturn } from "@/components/use-page-return";
import { formatDate } from "@/lib/format";
import { creditSourceName, creditDescription, signedCredits } from "@/lib/customer-records";
import creditStyles from "./credits.module.css";
export default function CreditsPage() {
  const { t, locale } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const [page, setPage] = useState(1);
  useEffect(() => { setPage(1); }, [token]);
  const balance = useConsoleResource(token, getCreditBalance);
  const loader = useCallback((key: string) => listCreditTransactions(key, page), [page]);
  const history = useConsoleResource(token, loader);
  const refreshHistory = history.revalidate;
  const refreshBalance = balance.revalidate;
  const update = useCallback(async () => { await Promise.all([refreshHistory(), refreshBalance()]); }, [refreshHistory, refreshBalance]);
  usePageReturn(update);
  const numberFormat = new Intl.NumberFormat(locale === "zh" ? "zh-CN" : "en-US");
  return <SiteShell eyebrow={t({ en: "Your credits", zh: "我的积分" })} title={t({ en: "Credit history", zh: "积分明细" })} description={t({ en: "See where your credits came from and how they were used.", zh: "查看积分来源、每笔收支和变更后的余额。" })} actions={<div className="button-row"><Link className="button primary" href="/pricing#one-time">{t({ en: "Get more credits", zh: "购买积分" })}</Link></div>}>
    <ConsoleGate session={session} ready={ready}><div className={styles.stack}>
      <CreditCheckoutNotice token={token} onConfirmed={update} />
      <ConsoleError error={balance.error} retry={() => void balance.refresh()} />
      <div className={`${styles.metrics} ${creditStyles.metrics}`}><div className={styles.metric}><span>{t({ en: "Available credits", zh: "可用积分" })}</span><strong>{balance.data ? numberFormat.format(balance.data.balance) : "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Expiring within 30 days", zh: "30 天内到期" })}</span><strong>{balance.data ? numberFormat.format(balance.data.expiring_credits) : "—"}</strong></div><div className={styles.metric}><span>{t({ en: "Next expiry", zh: "最近到期时间" })}</span><p>{balance.data?.next_expiry_at ? formatDate(balance.data.next_expiry_at, locale === "zh" ? "zh-CN" : "en-US") : balance.data ? t({ en: "No upcoming expiry", zh: "暂无到期计划" }) : "—"}</p></div></div>
      <Panel title={t({ en: "Credit history", zh: "积分明细" })} subtitle={t({ en: "Referral rewards, signup bonuses, subscription allowances and purchases appear here when issued.", zh: "邀请、受邀、注册赠送、订阅发放和充值购买，按实际到账记录展示。" })}>
        <DataTable caption={t({ en: "Credit history", zh: "积分明细" })}
          columns={[
            {key:"source",heading:t({en:"Source",zh:"积分来源"}),cell:item=><div className={creditStyles.source}><strong>{t(creditSourceName(item))}</strong>{creditDescription(item) ? <small>{creditDescription(item)}</small> : null}{item.related_transaction_id ? <small>{t({en:`Return of transaction #${item.related_transaction_id}`,zh:`原消费流水 #${item.related_transaction_id}`})}</small> : null}</div>},
            {key:"amount",heading:t({en:"Change",zh:"积分变动"}),align:"end",cell:item=><span className={creditStyles.amount} data-positive={item.amount>0}>{signedCredits(item.amount)}</span>},
            {key:"balance",heading:t({en:"Balance after",zh:"变更后余额"}),align:"end",cell:item=>numberFormat.format(item.balance_after)},
            {key:"date",heading:t({en:"Date",zh:"时间"}),cell:item=>formatDate(item.created_at,locale==="zh"?"zh-CN":"en-US")},
            {key:"expires",heading:t({en:"Expires",zh:"到期时间"}),cell:item=>item.expires_at ? formatDate(item.expires_at,locale==="zh"?"zh-CN":"en-US") : "—"}
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
