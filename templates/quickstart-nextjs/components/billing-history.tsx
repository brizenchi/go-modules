"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { listInvoices, listPurchases, type InvoiceItem, type PurchaseItem } from "@/lib/api";
import { readSession } from "@/lib/auth";
import { formatBillingAmount } from "@/lib/billing-money";
import { formatCurrencyUSD, formatDate } from "@/lib/format";
import { invoiceDownloadURL, paymentRecords, reloadLoadedPaymentPages, type PaymentRecord } from "@/lib/payment-history";
import { retainEqualData } from "@/lib/resource-data";
import { describeRequestFailure } from "@/lib/request-state";
import { useI18n } from "@/lib/i18n";
import { ConsoleError } from "./console-kit";
import { DataTable } from "./data-table";
import { StatusBadge } from "./status-badge";
import { Panel } from "./ui";
import styles from "./billing-history.module.css";

type History = {
 owner: string; purchases: PurchaseItem[]; invoices: InvoiceItem[];
 nextPurchasePage: number | null; nextInvoicePage: number | null;
 loading: boolean; loaded: boolean; purchasePages: number; invoicePages: number; errors: unknown[];
};
const initialHistory = (): History => ({ owner: "", purchases: [], invoices: [], nextPurchasePage: 1, nextInvoicePage: 1, loading: true, loaded: false, purchasePages: 0, invoicePages: 0, errors: [] });

export function BillingHistory({ token, revision = 0, invoicesEnabled = true }: { token: string; revision?: number; invoicesEnabled?: boolean }) {
 const { t, locale } = useI18n();
 const [state, setState] = useState<History>(initialHistory);
 const snapshot = useRef(state);
 useEffect(() => { snapshot.current = state; }, [state]);
 const generation = useRef(0);
 const busy = useRef(false);
 const load = useCallback(async (owner: string, request: number, purchasePage: number | null, invoicePage: number | null, replace = false) => {
  if (busy.current || readSession()?.token !== owner) return;
  const current = snapshot.current.owner === owner ? snapshot.current : initialHistory();
  const background = replace && current.loaded;
  busy.current = true;
  if (!background) setState(previous => ({ ...(previous.owner === owner ? previous : initialHistory()), owner, loading: true, errors: [] }));
  const [purchases, invoices] = await Promise.allSettled([
   purchasePage ? replace ? reloadLoadedPaymentPages(page => listPurchases(owner, page), current.purchasePages) : listPurchases(owner, purchasePage) : Promise.resolve(null),
   invoicePage ? replace ? reloadLoadedPaymentPages(page => listInvoices(owner, page), current.invoicePages) : listInvoices(owner, invoicePage) : Promise.resolve(null)
  ]);
  if (generation.current !== request || readSession()?.token !== owner) return;
  busy.current = false;
  setState(previous => {
   const base = previous.owner === owner ? previous : initialHistory();
   const showFailure = (error: unknown) => !background || !describeRequestFailure(error, "Payments").retryable;
   const errors = [...(purchases.status === "rejected" && showFailure(purchases.reason) ? [purchases.reason] : []), ...(invoices.status === "rejected" && showFailure(invoices.reason) ? [invoices.reason] : [])];
   const next: History = {
    owner, loading: false,
    loaded: base.loaded || (purchases.status === "fulfilled" && purchases.value !== null) || (invoices.status === "fulfilled" && invoices.value !== null),
    purchases: purchases.status === "fulfilled" && purchases.value ? retainEqualData(base.purchases, [...(replace ? [] : base.purchases), ...purchases.value.items]) : purchases.status === "rejected" && showFailure(purchases.reason) && background ? [] : base.purchases,
    invoices: invoices.status === "fulfilled" && invoices.value ? retainEqualData(base.invoices, [...(replace ? [] : base.invoices), ...invoices.value.items]) : invoices.status === "rejected" && showFailure(invoices.reason) && background ? [] : base.invoices,
    purchasePages: purchases.status === "fulfilled" && purchases.value ? purchases.value.page : base.purchasePages,
    invoicePages: invoices.status === "fulfilled" && invoices.value ? invoices.value.page : base.invoicePages,
    nextPurchasePage: purchases.status === "rejected" ? base.nextPurchasePage : purchases.value && purchases.value.page * purchases.value.limit < purchases.value.total ? purchases.value.page + 1 : null,
    nextInvoicePage: invoices.status === "rejected" ? base.nextInvoicePage : invoices.value && invoices.value.page * invoices.value.limit < invoices.value.total ? invoices.value.page + 1 : null,
    errors: errors.length === 0 && base.errors.length === 0 ? base.errors : errors
   };
   return !base.loading && Object.keys(next).every(key => next[key as keyof History] === base[key as keyof History]) ? base : next;
  });
 }, []);
 useEffect(() => {
  const counter = generation;
  const request = ++counter.current;
  busy.current = false;
  void load(token, request, 1, invoicesEnabled ? 1 : null, true);
  return () => { counter.current++; };
 }, [token, revision, invoicesEnabled, load]);
 const history = state.owner === token ? state : initialHistory();
 const rows = paymentRecords(history.purchases, history.invoices);
 const loadMore = () => void load(token, generation.current, history.nextPurchasePage, history.nextInvoicePage);
 function paymentStatus(value: string) {
  return t(value === "paid" ? { en: "Paid", zh: "已付款" } : value === "failed" || value === "uncollectible" ? { en: "Failed", zh: "付款失败" }
   : value === "unpaid" || value === "open" ? { en: "Unpaid", zh: "待付款" } : value === "no_payment_required" ? { en: "No payment required", zh: "无需付款" }
   : value === "void" || value === "canceled" ? { en: "Canceled", zh: "已取消" } : { en: "Pending", zh: "待确认" });
 }
 const errors = history.errors.length ? <div className={styles.errors}>{history.errors.map((error, index) => <ConsoleError key={index} error={error} retry={loadMore} />)}</div> : undefined;
 return <Panel className="span-12" title={t({ en: "Payment history", zh: "支付记录" })}>
  <div id="history" /><div id="invoices" />
  <DataTable caption={t({ en: "Payment history", zh: "支付记录" })} columns={[
   { key: "purchase", heading: t({ en: "Purchase", zh: "订单" }), cell: (row: PaymentRecord) => <><strong>{t(row.purchase?.product_type === "credits" ? { en: "Credits", zh: "积分包" } : row.purchase?.product_type === "lifetime" ? { en: "Lifetime", zh: "终身会员" } : row.purchase?.product_type === "subscription" ? { en: "Subscription", zh: "订阅" } : { en: "Payment", zh: "付款" })}</strong><br /><small>{row.invoice?.id || row.id}</small></> },
   { key: "amount", heading: t({ en: "Amount", zh: "金额" }), cell: row => row.purchase ? formatBillingAmount(row.purchase.amount, row.purchase.currency, locale) : row.invoice ? formatCurrencyUSD(row.invoice.amount_usd) : "—" },
   { key: "status", heading: t({ en: "Status", zh: "状态" }), cell: row => { const status = row.invoice?.status || row.purchase?.status || ""; return <StatusBadge tone={status === "paid" || status === "no_payment_required" ? "success" : status === "failed" || status === "uncollectible" ? "danger" : status === "void" || status === "canceled" ? "neutral" : "warning"}>{paymentStatus(status)}</StatusBadge>; } },
   { key: "time", heading: t({ en: "Date", zh: "时间" }), cell: row => formatDate(row.created_at, locale) },
   { key: "invoice", heading: t({ en: "Invoice", zh: "发票" }), cell: row => { const url = invoiceDownloadURL(row.invoice); return url ? <a href={url} target="_blank" rel="noopener noreferrer">{t({ en: "Download", zh: "下载" })} ↗</a> : "—"; } }
  ]} rows={rows} rowKey={row => row.id} loading={history.loading} loadingLabel={t({ en: "Loading payments…", zh: "正在读取支付记录…" })}
   empty={<span className={styles.empty}>{t({ en: "No payments yet.", zh: "暂无支付记录。" })}</span>} error={errors} />
  {history.nextPurchasePage || history.nextInvoicePage ? <div className={styles.more}><button className="button" type="button" disabled={history.loading} onClick={loadMore}>{history.loading ? t({ en: "Loading…", zh: "加载中…" }) : t({ en: "Load more", zh: "加载更多" })}</button></div> : null}
 </Panel>;
}
