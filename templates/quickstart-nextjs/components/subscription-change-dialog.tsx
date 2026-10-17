"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import Link from "next/link";
import { ApiError, changeSubscription, previewSubscriptionChange, type CapabilitiesView, type SubscriptionView } from "@/lib/api";
import { readSession } from "@/lib/auth";
import { stripeInvoiceURL } from "@/lib/billing-state";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { idleResource, loadingResource, settleResource, type RequestFailure } from "@/lib/request-state";
import { canConfirmSubscriptionChange, subscriptionPlanAction, subscriptionPreviewAmount, type SubscriptionSelection } from "@/lib/subscription-change";
import { useConsoleAction, ConsoleError } from "./console-kit";
import { ResourceFailure } from "./resource-feedback";
import { ResponsiveDialog } from "./responsive-dialog";
import { Select } from "./select";
import { DetailRows, Notice } from "./ui";
import styles from "./subscription-change-dialog.module.css";

type Props = {
  token: string;
  subscription: SubscriptionView;
  offers: CapabilitiesView["billing"]["offers"]["subscriptions"];
  selection: SubscriptionSelection;
  onSelection: (selection: SubscriptionSelection) => void;
  onClose: () => void;
  onChanged: () => void | Promise<void>;
};

export function SubscriptionChangeDialog({ token, subscription, offers, selection, onSelection, onClose, onChanged }: Props) {
  const { locale, t } = useI18n();
  const id = useId();
  const action = useConsoleAction(token);
  const [preview, setPreview] = useState(idleResource<Awaited<ReturnType<typeof previewSubscriptionChange>>>());
  const [result, setResult] = useState<Awaited<ReturnType<typeof changeSubscription>> | null>(null);
  const generation = useRef(0);
  const { plan, interval } = selection;
  const intervals = offers.find(offer => offer.plan === plan)?.intervals || [];
  const configured = intervals.includes(interval);
  const choice = subscriptionPlanAction(selection, subscription);
  const changeable = ["upgrade", "downgrade", "interval"].includes(choice);
  const dateLocale = locale === "zh" ? "zh-CN" : "en-US";
  const planName = (value: string) => t(({ starter: { en: "Starter", zh: "入门版" }, pro: { en: "Pro", zh: "专业版" }, premium: { en: "Premium", zh: "高级版" } })[value as "starter" | "pro" | "premium"] || { en: value, zh: value });
  const intervalName = (value: string) => t(value === "yearly" ? { en: "Yearly", zh: "按年" } : { en: "Monthly", zh: "按月" });
  const loadPreview = useCallback(async () => {
    if (readSession()?.token !== token) return;
    const current = ++generation.current;
    setPreview(previous => loadingResource(previous.data));
    const next = await settleResource(previewSubscriptionChange(token, { plan, interval }), "Subscription change preview");
    if (generation.current === current && readSession()?.token === token) setPreview(next);
  }, [token, plan, interval]);

  useEffect(() => {
    if (configured && changeable) void loadPreview();
    else setPreview(idleResource());
    const counter = generation;
    return () => { counter.current++; };
  }, [configured, changeable, loadPreview]);

  useEffect(() => { setResult(null); }, [token, plan, interval]);

  const ready = configured && preview.status === "ready" && !result;
  const confirmable = canConfirmSubscriptionChange({ selection, subscription, preview: preview.data, ready, busy: action.busy });
  const pendingError: RequestFailure | null = action.error instanceof ApiError && action.error.reason === "subscription_change_pending"
    ? { kind: "unknown", title: t({ en: "Resolve the pending change first", zh: "请先处理已有变更" }), message: t({ en: "Review the pending payment or scheduled change in billing before trying again.", zh: "请先在订阅与账单中处理待付款或预约变更，再重新选择。" }), retryable: false }
    : null;

  function confirm() {
    if (!confirmable || readSession()?.token !== token) return;
    void action.run(() => changeSubscription(token, { plan, interval }), saved => {
      setResult(saved);
      void onChanged();
    });
  }

  return <ResponsiveDialog open onClose={onClose} dismissible={!action.busy} title={t({ en: "Change plan", zh: "更换套餐" })} closeLabel={t({ en: "Close", zh: "关闭" })}>
    <div className="field-grid">
      <div className="field"><label htmlFor={`${id}-plan`}>{t({ en: "Plan", zh: "套餐" })}</label>
        <Select id={`${id}-plan`} value={plan} disabled={action.busy || Boolean(result)} onChange={event => {
          const next = offers.find(offer => offer.plan === event.target.value);
          onSelection({ plan: event.target.value, interval: next?.intervals.includes(interval) ? interval : next?.intervals[0] || interval });
        }}>{!offers.some(offer => offer.plan === plan) ? <option value={plan}>{planName(plan)}</option> : null}{offers.map(offer => <option key={offer.plan} value={offer.plan}>{planName(offer.plan)}</option>)}</Select>
      </div>
      <div className="field"><label htmlFor={`${id}-interval`}>{t({ en: "Billing interval", zh: "计费周期" })}</label>
        <Select id={`${id}-interval`} value={interval} disabled={action.busy || Boolean(result)} onChange={event => onSelection({ plan, interval: event.target.value })}>
          {!intervals.includes(interval) ? <option value={interval}>{intervalName(interval)}</option> : null}{intervals.map(value => <option key={value} value={value}>{intervalName(value)}</option>)}
        </Select>
      </div>
    </div>
    {result ? <div className={styles.feedback} role="status"><Notice tone="success">{t(result.pending_payment
      ? { en: "Complete payment before the new plan takes effect.", zh: "完成付款后，新套餐才会生效。" }
      : result.scheduled_change ? { en: "Your plan change is scheduled for the next billing cycle.", zh: "套餐将在下个计费周期变更。" }
        : { en: "Your plan has been updated.", zh: "套餐已更新。" })}</Notice>
      {result.pending_payment ? stripeInvoiceURL(result.pending_payment.invoice_url)
        ? <a className="button primary" href={stripeInvoiceURL(result.pending_payment.invoice_url)!} target="_blank" rel="noopener noreferrer">{t({ en: "Complete payment", zh: "完成付款" })}</a>
        : <Link className="button" href="/billing">{t({ en: "Review payment", zh: "查看付款" })}</Link> : null}
    </div> : choice === "current" ? <p className={styles.note} role="status">{t({ en: "This is your current plan and billing interval.", zh: "这是你当前的套餐与计费周期。" })}</p>
      : !configured ? <p className={styles.note}>{t({ en: "Choose an available plan and billing interval.", zh: "请选择可用套餐与计费周期。" })}</p>
        : !changeable ? <p className={styles.note}>{t({ en: "Review your existing payment or scheduled change in billing first.", zh: "请先在订阅与账单中处理已有付款或预约变更。" })} <Link href="/billing">{t({ en: "Manage subscription", zh: "管理订阅" })}</Link></p>
          : preview.status === "ready" && preview.data.target_plan === plan && preview.data.target_interval === interval ? <div className={styles.summary} aria-live="polite">
            <div className={styles.total}><span>{t({ en: "Due now", zh: "当前应付" })}</span><strong>{subscriptionPreviewAmount(preview.data, dateLocale)}</strong></div>
            <DetailRows rows={[
              { label: t({ en: "Selected plan", zh: "目标套餐" }), value: `${planName(plan)} · ${intervalName(interval)}` },
              { label: t({ en: "Change timing", zh: "变更方式" }), value: t(preview.data.change_mode === "period_end" ? { en: "Next billing cycle", zh: "下个计费周期生效" } : preview.data.change_mode === "immediate_reset_cycle" ? { en: "Immediately, new billing cycle", zh: "立即生效，重新开始计费周期" } : { en: "Immediately, with proration", zh: "立即生效，按比例结算" }) },
              { label: t({ en: "Current period ends", zh: "本期结束" }), value: formatDate(preview.data.current_period_end, dateLocale) },
              { label: t({ en: "Next billing date", zh: "下次计费" }), value: formatDate(preview.data.next_billing_at, dateLocale) }
            ]} />
          </div> : preview.status === "error" ? <ResourceFailure failure={preview.failure} onRetry={() => void loadPreview()} /> : <p className={styles.note} role="status">{t({ en: "Loading the price preview…", zh: "正在获取费用预览…" })}</p>}
    {pendingError ? <ResourceFailure failure={pendingError} /> : <ConsoleError error={action.error} />}
    <div className={styles.actions}>
      {!result ? <button className="button primary" type="button" disabled={!confirmable} onClick={confirm}>{action.busy ? t({ en: "Updating…", zh: "更新中…" }) : t({ en: "Confirm plan change", zh: "确认变更套餐" })}</button> : null}
      <button className="button" type="button" disabled={action.busy} onClick={onClose}>{t(result ? { en: "Done", zh: "完成" } : { en: "Cancel", zh: "取消" })}</button>
    </div>
  </ResponsiveDialog>;
}
