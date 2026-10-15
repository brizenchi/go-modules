"use client";

import { useId } from "react";
import type { OperatorMetricDay, OperatorOverview as Overview, OperatorReportFilters } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import styles from "./admin-shell.module.css";
import { Select } from "./select";

type Metric = "registrations" | "referrals" | "payments" | "new_paying_users";
type Copy = { en: string; zh: string };
const metricNames: Record<Metric, Copy> = { registrations: { en: "Registrations", zh: "注册人数" }, referrals: { en: "Invitations", zh: "邀请数" }, payments: { en: "Payments", zh: "成功支付数" }, new_paying_users: { en: "First-time paying users", zh: "首次付费用户" } };
function Trend({ title, days, metrics, emptyLabel }: { title: string; days: OperatorMetricDay[]; metrics: [Metric, Metric]; emptyLabel: string }) {
 const { t } = useI18n();
 const id = useId();
 const maximum = Math.max(1, ...days.flatMap(day => metrics.map(key => day[key])));
 const ceiling = Math.max(2, Math.ceil(maximum / 2) * 2);
 const x = (index: number) => 34 + index * 510 / Math.max(1, days.length - 1);
 const y = (value: number) => 172 - value * 144 / ceiling;
 const any = days.some(day => metrics.some(key => day[key] > 0));
 return <section className={styles.chartCard} aria-labelledby={id}>
  <div className={styles.chartHeading}><h2 id={id}>{title}</h2><div className={styles.chartLegend}>{metrics.map((key, i) => <span key={key}><i style={{ background: i ? "var(--muted)" : "var(--accent-ink)" }} />{t(metricNames[key])}</span>)}</div></div>
  <svg className={styles.trendChart} viewBox="0 0 560 208" role="img" aria-label={`${title}. ${any ? t({ en: "Daily counts; exact values are available below.", zh: "每日数量，精确数据可在下方查看。" }) : emptyLabel}`}>
   {[0, ceiling/2, ceiling].map(value => <g key={value}><line x1="34" x2="544" y1={y(value)} y2={y(value)} stroke="var(--line)" /><text x="24" y={y(value)+4} textAnchor="end">{value}</text></g>)}
   {metrics.map((key, i) => <g key={key} stroke={i ? "var(--muted)" : "var(--accent-ink)"} fill="none"><polyline vectorEffect="non-scaling-stroke" strokeWidth="2" strokeDasharray={i ? "5 4" : undefined} points={days.map((day, index) => `${x(index)},${y(day[key])}`).join(" ")} />{days.map((day, index) => <circle key={day.date} cx={x(index)} cy={y(day[key])} r="3" fill="var(--surface)" strokeWidth="2"><title>{`${day.date} · ${t(metricNames[key])}: ${day[key]}`}</title></circle>)}</g>)}
   {[0, Math.floor((days.length-1)/2), days.length-1].filter((value,index,array) => array.indexOf(value) === index && value >= 0).map(index => <text key={index} x={x(index)} y="198" textAnchor={index === 0 ? "start" : index === days.length-1 ? "end" : "middle"}>{days[index]?.date.slice(5)}</text>)}
  </svg>
  {!any ? <p className={styles.chartEmpty}>{emptyLabel}</p> : null}
  <details className={styles.chartDetails}><summary>{t({ en: "View daily data", zh: "查看每日数据" })}</summary><div><table><caption className="sr-only">{title}</caption><thead><tr><th>{t({ en: "Date", zh: "日期" })}</th>{metrics.map(key => <th key={key}>{t(metricNames[key])}</th>)}</tr></thead><tbody>{days.map(day => <tr key={day.date}><td>{day.date}</td>{metrics.map(key => <td key={key}>{day[key]}</td>)}</tr>)}</tbody></table></div></details>
 </section>;
}
export function OperatorOverview({ data, filters, onFilters, loading }: { data: Overview | null; filters: OperatorReportFilters; onFilters: (next: OperatorReportFilters) => void; loading: boolean }) {
 const { t, locale } = useI18n();
 const ratio = data && data.users > 0 ? Math.min(100, data.paid_users / data.users * 100) : 0;
 const metrics = [
  ["today_registrations", { en: "Registrations today", zh: "今日注册数" }], ["today_referrals", { en: "Invitations today", zh: "今日邀请数" }],
  ["users", { en: "Total users", zh: "当前总用户数" }], ["today_payments", { en: "Payments today", zh: "今日支付数" }], ["paid_users", { en: "Paying users", zh: "总付费用户数" }]
 ] as const;
 return <div className={styles.overview}>
  <div className={styles.reportControls}><p>{t({ en: "Acquisition & conversion", zh: "用户增长与付费转化" })}</p><div>
   <label><span className="sr-only">{t({ en: "Date range", zh: "统计周期" })}</span><Select value={filters.days} onChange={event => onFilters({ ...filters, days: Number(event.target.value) as 7 | 30 })}><option value="7">{t({ en: "Last 7 days", zh: "近 7 天" })}</option><option value="30">{t({ en: "Last 30 days", zh: "近 30 天" })}</option></Select></label>
   <label><span className="sr-only">{t({ en: "Reporting timezone", zh: "统计时区" })}</span><Select value={filters.timezone} onChange={event => onFilters({ ...filters, timezone: event.target.value as OperatorReportFilters["timezone"] })}><option value="Asia/Shanghai">{t({ en: "UTC+8", zh: "北京时间 UTC+8" })}</option><option value="UTC">UTC</option></Select></label>
   <label><span className="sr-only">{t({ en: "Payment environment", zh: "支付环境" })}</span><Select value={filters.mode} onChange={event => onFilters({ ...filters, mode: event.target.value as "live" | "test" })}><option value="live">{t({ en: "Live payments", zh: "正式支付" })}</option><option value="test">{t({ en: "Test payments", zh: "测试支付" })}</option></Select></label>
  </div></div>
  <div className={styles.metrics} aria-busy={loading}>{metrics.map(([key, label]) => <div className={styles.metric} key={key}><span>{t(label)}</span><strong>{data ? data[key].toLocaleString() : "—"}</strong></div>)}</div>
  {data ? <>
   <div className={styles.charts}><Trend title={t({ en: "User growth", zh: "用户增长" })} days={data.series} metrics={["registrations", "referrals"]} emptyLabel={t({ en: "No registrations or invitations in this period.", zh: "本周期暂无注册或邀请记录。" })} /><Trend title={t({ en: "Payment conversion", zh: "付费转化" })} days={data.series} metrics={["payments", "new_paying_users"]} emptyLabel={t({ en: "No positive payments in this environment and period.", zh: "当前支付环境在本周期暂无实际付款记录。" })} /></div>
   <section className={styles.conversionCard}><div><span>{t({ en: "Historical paid conversion", zh: "累计付费转化率" })}</span><strong>{ratio.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", { maximumFractionDigits: 1 })}%</strong><p>{t({ en: `${data.paid_users.toLocaleString()} paying users / ${data.users.toLocaleString()} current users`, zh: `${data.paid_users.toLocaleString()} 位付费用户 / ${data.users.toLocaleString()} 位当前用户` })}</p></div><p>{t({ en: "Counts customers with at least one positive payment in the selected environment, including renewals and credit purchases. Free invoices and manual memberships are excluded. Refunds do not erase historical conversion. This is not visitor-to-signup conversion.", zh: "统计当前用户中，在所选环境至少实际付款一次的人数，含订阅、续费与积分购买。零元账单和手动会员不计入；退款不抹除历史付费记录。此处不代表访客到注册的转化率。" })}</p></section>
   <p className={styles.reportNote}>{t({ en: "Invitations count attributed registrations. Payments are deduplicated by invoice or checkout, using the provider’s payment event time. Registrations and invitations include all site users.", zh: "邀请数指已归因的受邀注册；支付按账单或结账单去重，以支付事件时间统计。注册与邀请覆盖本站全部用户。" })} <time dateTime={data.as_of}>{t({ en: "Updated", zh: "更新于" })} {new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en-US", { hour: "2-digit", minute: "2-digit", timeZone: data.timezone }).format(new Date(data.as_of))}</time></p>
  </> : null}
 </div>;
}
