"use client";

import { useCallback, useState } from "react";
import { listPurchases, type PurchaseItem } from "@/lib/api";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { ConsoleError, useConsoleResource } from "./console-kit";
import { DataTable } from "./data-table";
import { StatusBadge } from "./status-badge";
import { Panel } from "./ui";

export function BillingHistory({ token, revision = 0 }: { token: string; revision?: number }) {
 const { t, locale } = useI18n();
 const [page, setPage] = useState(1);
 const loader = useCallback((current: string) => { void revision; return listPurchases(current, page); }, [page, revision]);
 const history = useConsoleResource(token, loader);
 function paymentStatus(value: string) {
  return t(value === "paid" ? { en: "Paid", zh: "已付款" } : value === "failed" ? { en: "Failed", zh: "付款失败" }
   : value === "unpaid" ? { en: "Unpaid", zh: "待付款" } : value === "no_payment_required" ? { en: "No payment required", zh: "无需付款" }
   : { en: "Pending", zh: "待确认" });
 }
 return <Panel className="span-12" title={t({ en: "Payment history", zh: "支付记录" })}>
  <DataTable caption={t({ en: "Payment history", zh: "支付记录" })} columns={[
   { key: "purchase", heading: t({ en: "Purchase", zh: "订单" }), cell: (item: PurchaseItem) => <><strong>{t(item.product_type === "credits" ? { en: "Credits", zh: "积分包" } : item.product_type === "lifetime" ? { en: "Lifetime", zh: "终身会员" } : item.product_type === "subscription" ? { en: "Subscription", zh: "订阅" } : { en: "Payment", zh: "付款" })}</strong><br /><small>{item.id}</small></> },
   { key: "amount", heading: t({ en: "Amount", zh: "金额" }), cell: item => formatBillingAmount(item.amount, item.currency, locale) },
   { key: "status", heading: t({ en: "Status", zh: "状态" }), cell: item => <StatusBadge tone={item.status === "paid" || item.status === "no_payment_required" ? "success" : item.status === "failed" ? "danger" : "warning"}>{paymentStatus(item.status)}</StatusBadge> },
   { key: "time", heading: t({ en: "Date", zh: "时间" }), cell: item => formatDate(item.created_at, locale) }
  ]} rows={history.data?.items || []} rowKey={item => item.id} loading={history.loading} loadingLabel={t({ en: "Loading payments…", zh: "正在读取支付记录…" })}
   empty={t({ en: "No payments yet.", zh: "暂无支付记录。" })} error={history.error ? <ConsoleError error={history.error} /> : undefined}
   pagination={history.data && history.data.total > 10 ? { page, limit: 10, total: history.data.total, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (current, pages) => `${current} / ${pages}` } : undefined} />
 </Panel>;
}
