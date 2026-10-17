"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import Link from "next/link";
import { ApiError, changeSubscription, getSubscription, reactivateSubscription, previewSubscriptionChange, type CapabilitiesView, type SubscriptionView } from "@/lib/api";
import { readSession } from "@/lib/auth";
import { stripeInvoiceURL } from "@/lib/billing-state";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { idleResource, loadingResource, settleResource } from "@/lib/request-state";
import { canConfirmSubscriptionChange, isValidSubscriptionPreview, resumeRenewalForChange, subscriptionPlanAction, subscriptionPlanIntent, subscriptionChangeBlocker, subscriptionPreviewAmount, type SubscriptionSelection } from "@/lib/subscription-change";
import { billingText, changeDialogTitle, changeSubmitText, changeTimingText, changePaymentText, changeResultText, subscriptionRestrictionText } from "@/lib/billing-presentation";
import { useConsoleAction, ConsoleError } from "./console-kit";
import { BillingFailure } from "./billing-failure";
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
  const [restored, setRestored] = useState<{ owner: string; base: SubscriptionView; value: SubscriptionView } | null>(null);
  const [revision, setRevision] = useState(0);
  const currentSubscription = restored?.owner === token && restored.base === subscription ? restored.value : subscription;
  const generation = useRef(0);
  const { plan, interval } = selection;
  const intervals = offers.find(offer => offer.plan === plan)?.intervals || [];
  const configured = intervals.includes(interval);
  const choice = subscriptionPlanAction(selection, currentSubscription);
  const blocker = subscriptionChangeBlocker(currentSubscription);
  const restriction = subscriptionRestrictionText(blocker);
  const intent = subscriptionPlanIntent(selection, currentSubscription);
  const previewable = ["upgrade", "downgrade", "interval"].includes(intent) && (!blocker || blocker === "cancellation");
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
    if (configured && previewable) void loadPreview();
    else setPreview(idleResource());
    const counter = generation;
    return () => { counter.current++; };
  }, [configured, previewable, loadPreview, revision]);

  useEffect(() => { setResult(null); }, [token, plan, interval]);

  const validPreview = preview.status === "ready" && isValidSubscriptionPreview(preview.data) && preview.data.target_plan === plan && preview.data.target_interval === interval ? preview.data : null;
  const paymentCopy = validPreview ? changePaymentText(validPreview) : null;
  const ready = configured && Boolean(validPreview) && !result;
  const resultCopy = result ? changeResultText(result) : null;
  const confirmable = canConfirmSubscriptionChange({ selection, subscription: currentSubscription, preview: preview.data, ready, busy: action.busy });
  const pendingError = action.error instanceof ApiError && action.error.reason === "subscription_change_pending";

  function confirm() {
    if (!confirmable || readSession()?.token !== token) return;
    void action.run(() => changeSubscription(token, { plan, interval }), saved => {
      setResult(saved);
      void onChanged();
    });
  }

  function resume() {
    void action.run(() => resumeRenewalForChange({
      load: () => getSubscription(token), resume: () => reactivateSubscription(token),
      current: () => readSession()?.token === token
    }), latest => {
      if (!latest) return;
      generation.current++;
      setRestored({ owner: token, base: subscription, value: latest });
      setPreview(idleResource());
      setRevision(value => value + 1);
      void onChanged();
    });
  }

  return <ResponsiveDialog open onClose={onClose} dismissible={!action.busy} title={t(resultCopy?.title || changeDialogTitle(intent))} closeLabel={t({ en: "Close", zh: "关闭" })}>
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
    {result && resultCopy ? <div className={styles.feedback} role="status"><Notice tone={resultCopy.state === "payment" ? "default" : "success"}>{t(resultCopy.body)}</Notice>
      {result.scheduled_change ? <p className={styles.note}>{planName(result.scheduled_change.plan)} · {intervalName(result.scheduled_change.billing_cycle)} · {formatDate(result.scheduled_change.effective_at, dateLocale)}</p> : null}
      {result.pending_payment ? stripeInvoiceURL(result.pending_payment.invoice_url)
        ? <a className="button primary" href={stripeInvoiceURL(result.pending_payment.invoice_url)!} target="_blank" rel="noopener noreferrer">{t({ en: "Complete payment", zh: "完成付款" })}</a>
        : <Link className="button" href="/billing">{t({ en: "Review payment", zh: "查看付款" })}</Link> : null}
      {resultCopy.state === "updated" ? <Link className="button" href="/billing#history" onClick={onClose}>{t({ en: "View payment history", zh: "查看支付记录" })}</Link> : null}
    </div> : choice === "current" ? <p className={styles.note} role="status">{t({ en: "Choose another plan or billing interval.", zh: "请选择其他套餐或计费周期。" })}</p>
      : !configured ? <p className={styles.note}>{t({ en: "Choose an available plan and billing interval.", zh: "请选择可用套餐与计费周期。" })}</p>
        : !previewable ? <p className={styles.note}>{t(restriction || { en: "Review your subscription in billing before choosing another plan.", zh: "请先在订阅与账单中查看当前订阅，再选择其他套餐。" })} <Link href="/billing">{t(billingText.viewBilling)}</Link></p>
          : validPreview ? <div className={styles.summary} aria-live="polite">
            <div className={styles.total}><span>{t({ en: "Due now", zh: "当前应付" })}</span><strong>{subscriptionPreviewAmount(validPreview, dateLocale)}</strong></div>
            <DetailRows rows={[
              { label: t({ en: "Selected plan", zh: "目标套餐" }), value: `${planName(plan)} · ${intervalName(interval)}` },
              { label: t({ en: "Change timing", zh: "生效时间" }), value: t(changeTimingText(validPreview).label) },
              { label: t({ en: "Current period ends", zh: "本期结束" }), value: formatDate(validPreview.current_period_end, dateLocale) },
              { label: t({ en: "Next billing date", zh: "下次计费" }), value: formatDate(validPreview.next_billing_at, dateLocale) }
            ]} />
            <p className={styles.note}>{t(changeTimingText(validPreview).detail)}</p>
            {paymentCopy ? <p className={styles.note}>{t(paymentCopy)}</p> : null}
          </div> : preview.status === "error" ? <BillingFailure failure={preview.failure} onRetry={() => void loadPreview()} />
            : preview.status === "ready" ? <BillingFailure failure={{ kind: "unavailable", title: "", message: "", retryable: true }} onRetry={() => void loadPreview()} />
              : <p className={styles.note} role="status">{t({ en: "Loading the price preview…", zh: "正在获取费用预览…" })}</p>}
    {!result && blocker === "cancellation" ? <p className={styles.note} role="status">{t({ en: "Renewal is canceled. Resume renewal for your current plan to continue, then review and confirm this change separately.", zh: "已取消续费。先恢复当前套餐续费，再核对费用并确认变更。" })}</p> : null}
    {pendingError ? <p className={styles.note} role="alert">{t({ en: "Your billing status has changed. Review billing before trying again.", zh: "订阅状态已变更，请先查看账单后重试。" })} <Link href="/billing">{t(billingText.viewBilling)}</Link></p> : <ConsoleError error={action.error} />}
    <div className={styles.actions}>
      {!result ? blocker === "cancellation"
        ? <button className="button primary" type="button" disabled={action.busy || !configured} onClick={resume}>{action.busy ? t({ en: "Restoring…", zh: "恢复中…" }) : t({ en: "Resume renewal to continue", zh: "恢复续费并继续" })}</button>
        : <button className="button primary" type="button" disabled={!confirmable} onClick={confirm}>{action.busy ? t({ en: "Submitting…", zh: "正在提交…" }) : t(changeSubmitText({ choice, configured, phase: preview.status, preview: validPreview, amount: validPreview ? subscriptionPreviewAmount(validPreview, dateLocale) : "" }))}</button> : null}
      <button className="button" type="button" disabled={action.busy} onClick={onClose}>{t(result ? { en: "Done", zh: "完成" } : { en: "Cancel", zh: "取消" })}</button>
    </div>
  </ResponsiveDialog>;
}
