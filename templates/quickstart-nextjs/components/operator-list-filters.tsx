"use client";

import type { FormEvent } from "react";
import type { AdminSection } from "@/lib/admin-navigation";
import { useI18n } from "@/lib/i18n";
import { Select } from "./select";
import styles from "./operator-list-filters.module.css";

type Props = {
  section: AdminSection; label: string; searchLabel: string;
  search: string; status: string; mode: string; from: string; to: string;
  total?: number; loading: boolean; filtered: boolean;
  onSearch: (value: string) => void; onStatus: (value: string) => void;
  onMode: (value: string) => void; onFrom: (value: string) => void; onTo: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void; onReset: () => void;
};

export function OperatorListFilters(props: Props) {
  const { t } = useI18n();
  const { section } = props;
  return <form className={styles.form} onSubmit={props.onSubmit} role="search" aria-label={props.label}>
    <div className={`${styles.fields}${section === "orders" ? ` ${styles.payments}` : ""}`}>
      <div className={`field ${styles.searchField}`}><label htmlFor="operator-search">{t({ en: "Search", zh: "搜索" })}</label>
        <div className={styles.search}><input id="operator-search" type="search" placeholder={props.searchLabel} aria-label={props.searchLabel} value={props.search} onChange={event => props.onSearch(event.target.value)} /><button className="button" type="submit" disabled={props.loading}>{t({ en: "Search", zh: "搜索" })}</button></div>
      </div>
      {section === "orders" ? <>
        <div className="field"><label htmlFor="payment-mode">{t({ en: "Environment", zh: "支付环境" })}</label><Select id="payment-mode" value={props.mode} onChange={event => props.onMode(event.target.value)}><option value="">{t({ en: "All", zh: "全部" })}</option><option value="live">{t({ en: "Live", zh: "正式支付" })}</option><option value="test">{t({ en: "Test", zh: "测试支付" })}</option></Select></div>
        <div className="field"><label htmlFor="payment-status">{t({ en: "Payment status", zh: "支付状态" })}</label><Select id="payment-status" value={props.status} onChange={event => props.onStatus(event.target.value)}><option value="">{t({ en: "All", zh: "全部" })}</option><option value="paid">{t({ en: "Paid", zh: "已支付" })}</option><option value="unpaid">{t({ en: "Unpaid", zh: "未支付" })}</option><option value="failed">{t({ en: "Failed", zh: "失败" })}</option><option value="expired">{t({ en: "Expired", zh: "已过期" })}</option></Select></div>
      </> : null}
      {section === "referrals" ? <div className="field"><label htmlFor="referral-status">{t({ en: "Activation status", zh: "激活状态" })}</label><Select id="referral-status" value={props.status} onChange={event => props.onStatus(event.target.value)}><option value="">{t({ en: "All", zh: "全部" })}</option><option value="pending">{t({ en: "Pending", zh: "待激活" })}</option><option value="activated">{t({ en: "Activated", zh: "已激活" })}</option><option value="expired">{t({ en: "Expired", zh: "已过期" })}</option></Select></div> : null}
      {section === "audit" ? <div className="field"><label htmlFor="audit-outcome">{t({ en: "Outcome", zh: "处理结果" })}</label><Select id="audit-outcome" value={props.status} onChange={event => props.onStatus(event.target.value)}><option value="">{t({ en: "All", zh: "全部" })}</option><option value="succeeded">{t({ en: "Succeeded", zh: "已完成" })}</option><option value="failed">{t({ en: "Failed", zh: "失败" })}</option><option value="pending">{t({ en: "Processing", zh: "处理中" })}</option></Select></div> : null}
    </div>
    <div className={styles.footer}>
      {section === "orders" ? <div className={styles.dates}>
        <div className="field"><label htmlFor="payment-from">{t({ en: "From date", zh: "开始日期" })}</label><input id="payment-from" type="date" value={props.from} max={props.to || undefined} onChange={event => props.onFrom(event.target.value)} /></div>
        <div className="field"><label htmlFor="payment-to">{t({ en: "Through date", zh: "结束日期" })}</label><input id="payment-to" type="date" value={props.to} min={props.from || undefined} onChange={event => props.onTo(event.target.value)} /></div>
        <span className={styles.timezone}>{t({ en: "Local time", zh: "本机时区" })}</span>
      </div> : null}
      <div className={styles.meta}>
        {props.filtered ? <button className={`button ${styles.reset}`} type="button" onClick={props.onReset}>{t({ en: "Clear filters", zh: "清除筛选" })}</button> : null}
        <span role="status" aria-live="polite">{props.total === undefined ? "—" : t({ en: `${props.total.toLocaleString()} records`, zh: `共 ${props.total.toLocaleString()} 条` })}</span>
      </div>
    </div>
  </form>;
}
