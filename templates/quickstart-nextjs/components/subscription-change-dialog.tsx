"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { ApiError, cancelScheduledChange, changeSubscription, createBillingPortalSession, getSubscription, reactivateSubscription, previewSubscriptionChange, type CapabilitiesView, type SubscriptionView } from "@/lib/api";
import { readSession } from "@/lib/auth";
import { stripeInvoiceURL } from "@/lib/billing-state";
import { formatDate } from "@/lib/format";
import { formatBillingAmount } from "@/lib/billing-money";
import { useI18n } from "@/lib/i18n";
import { appEnv } from "@/lib/env";
import { confirmPlanSelection, isScheduledSelection, keepCurrentSubscription } from "@/lib/subscription-management";
import { idleResource, loadingResource, settleResource } from "@/lib/request-state";
import { canConfirmSubscriptionChange, canKeepCurrentPlan, isValidSubscriptionPreview, resumeRenewalForChange, subscriptionPlanIntent, subscriptionChangeBlocker, subscriptionPreviewAmount, type SubscriptionSelection } from "@/lib/subscription-change";
import { billingText, cancelPlanChangeText, changeDialogTitle, changeSubmitText, changeTimingText, changePaymentText, changeResultText, planActionText, subscriptionRestrictionText } from "@/lib/billing-presentation";
import { useConsoleAction, ConsoleError } from "./console-kit";
import { BillingFailure } from "./billing-failure";
import { ResponsiveDialog } from "./responsive-dialog";
import { DetailRows, Notice } from "./ui";
import styles from "./subscription-change-dialog.module.css";

type Props = {
  token: string;
  subscription: SubscriptionView;
  offers: CapabilitiesView["billing"]["offers"]["subscriptions"];
  selection: SubscriptionSelection;
  onClose: () => void;
  onChanged: () => void | Promise<void>;
};

export function SubscriptionChangeDialog({ token, subscription, offers, selection, onClose, onChanged }: Props) {
  const { locale, t } = useI18n();
  const action = useConsoleAction(token);
  const [preview, setPreview] = useState(idleResource<Awaited<ReturnType<typeof previewSubscriptionChange>>>());
  const [result, setResult] = useState<Awaited<ReturnType<typeof changeSubscription>> | null>(null);
  const [refreshed, setRefreshed] = useState<{ owner: string; base: SubscriptionView; value: SubscriptionView } | null>(null);
  const [revision, setRevision] = useState(0);
  const [reviewRequired, setReviewRequired] = useState(false);
  const [kept, setKept] = useState(false);
  const currentSubscription = refreshed?.owner === token && refreshed.base === subscription ? refreshed.value : subscription;
  const generation = useRef(0);
  const { plan, interval } = selection;
  const intervals = offers.find(offer => offer.plan === plan)?.intervals || [];
  const configured = intervals.includes(interval);
  const choice = subscriptionPlanIntent(selection, currentSubscription);
  const blocker = subscriptionChangeBlocker(currentSubscription);
  const restriction = subscriptionRestrictionText(blocker);
  const intent = subscriptionPlanIntent(selection, currentSubscription);
  const keepRequested = subscriptionPlanIntent(selection, subscription) === "keep";
  const keepConfirmable = keepRequested && !kept && !action.busy && canKeepCurrentPlan(currentSubscription)
    && currentSubscription.plan === plan && currentSubscription.billing_cycle === interval;
  const alreadyScheduled = !keepRequested && isScheduledSelection(selection, currentSubscription);
  const previewable = !alreadyScheduled && ["upgrade", "downgrade", "interval"].includes(intent) && (!blocker || blocker === "cancellation" || blocker === "scheduled");
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

  useEffect(() => { setResult(null); setKept(false); setReviewRequired(false); }, [token, plan, interval]);

  const validPreview = preview.status === "ready" && isValidSubscriptionPreview(preview.data) && preview.data.target_plan === plan && preview.data.target_interval === interval ? preview.data : null;
  const paymentCopy = validPreview ? changePaymentText(validPreview) : null;
  const ready = configured && Boolean(validPreview) && !result;
  const resultCopy = result ? changeResultText(result) : alreadyScheduled ? changeResultText(currentSubscription) : null;
  const confirmable = !alreadyScheduled && canConfirmSubscriptionChange({ selection, subscription: { ...currentSubscription, scheduled_change: null }, preview: preview.data, ready, busy: action.busy });
  const pendingError = action.error instanceof ApiError && action.error.reason === "subscription_change_pending";
  const paymentURL = stripeInvoiceURL(currentSubscription.pending_payment?.invoice_url);

  function confirm() {
    if (!confirmable || !validPreview || readSession()?.token !== token) return;
    const attempt = generation.current;
    void action.run(async () => {
      try {
        return await confirmPlanSelection({ selection, subscription: currentSubscription, preview: validPreview }, {
          load: () => getSubscription(token), removeSchedule: () => cancelScheduledChange(token),
          preview: () => previewSubscriptionChange(token, { plan, interval }),
          submit: () => changeSubscription(token, { plan, interval }), current: () => readSession()?.token === token && generation.current === attempt
        });
      } catch (error) {
        void onChanged();
        throw error;
      }
    }, saved => {
      if (!saved) return;
      if (saved.state === "review") { setReviewRequired(true); acceptSnapshot(saved.subscription); }
      else { setResult(saved.result); void onChanged(); }
    });
  }

  function acceptSnapshot(latest: SubscriptionView | null) {
    if (!latest) return;
    generation.current++;
    setRefreshed({ owner: token, base: subscription, value: latest });
    setPreview(idleResource());
    setRevision(value => value + 1);
    void onChanged();
  }

  function keepPlan() {
    if (!keepConfirmable || readSession()?.token !== token) return;
    const attempt = generation.current;
    void action.run(async () => {
      try {
        return await keepCurrentSubscription(currentSubscription, {
          load: () => getSubscription(token), removeSchedule: () => cancelScheduledChange(token),
          current: () => readSession()?.token === token && generation.current === attempt
        });
      } catch (error) {
        void onChanged();
        throw error;
      }
    }, saved => {
      if (!saved) return;
      if (saved.state === "kept") setKept(true);
      else setReviewRequired(true);
      acceptSnapshot(saved.subscription);
    });
  }

  function resume() {
    void action.run(() => resumeRenewalForChange({
      load: () => getSubscription(token), resume: () => reactivateSubscription(token),
      current: () => readSession()?.token === token
    }), acceptSnapshot);
  }

  function openPaymentSettings() {
    void action.run(() => createBillingPortalSession(token, `${appEnv.appUrl}/billing`), saved => {
      window.location.href = saved.url;
    });
  }

  return <ResponsiveDialog open onClose={onClose} dismissible={!action.busy} title={t(kept ? cancelPlanChangeText.successTitle : keepRequested ? changeDialogTitle("keep", planName(plan)) : resultCopy?.title || changeDialogTitle(intent))} closeLabel={t({ en: "Close", zh: "关闭" })}>
    <p className={styles.selection}><strong>{planName(plan)}</strong><span>{intervalName(interval)}</span></p>
    {kept ? <div className={styles.feedback} role="status"><Notice tone="success">{t(cancelPlanChangeText.successBody)}</Notice></div>
      : keepRequested ? <div className={styles.summary}>
        <p className={styles.note}>{t(cancelPlanChangeText.description)}</p>
        {currentSubscription.scheduled_change ? <DetailRows rows={[{
          label: t(cancelPlanChangeText.scheduledChange),
          value: `${planName(currentSubscription.scheduled_change.plan)} · ${intervalName(currentSubscription.scheduled_change.billing_cycle)} · ${formatDate(currentSubscription.scheduled_change.effective_at, dateLocale)}`
        }]} /> : null}
      </div> : result && resultCopy ? <div className={styles.feedback} role="status"><Notice tone={resultCopy.state === "payment" ? "default" : "success"}>{t(resultCopy.body)}</Notice>
      {result.scheduled_change ? <p className={styles.note}>{planName(result.scheduled_change.plan)} · {intervalName(result.scheduled_change.billing_cycle)} · {formatDate(result.scheduled_change.effective_at, dateLocale)}</p> : null}
      {result.pending_payment ? stripeInvoiceURL(result.pending_payment.invoice_url)
        ? <a className="button primary" href={stripeInvoiceURL(result.pending_payment.invoice_url)!} target="_blank" rel="noopener noreferrer">{t({ en: "Complete payment", zh: "完成付款" })}</a>
        : <Link className="button" href="/billing">{t({ en: "Review payment", zh: "查看付款" })}</Link> : null}
      {resultCopy.state === "updated" ? <Link className="button" href="/billing#history" onClick={onClose}>{t({ en: "View payment history", zh: "查看支付记录" })}</Link> : null}
    </div> : alreadyScheduled && currentSubscription.scheduled_change ? <div className={styles.feedback} role="status">
      <Notice tone="success">{t({ en: "This change is already scheduled.", zh: "此套餐变更已预约。" })}</Notice>
      <DetailRows rows={[{ label: t({ en: "Effective date", zh: "生效日期" }), value: formatDate(currentSubscription.scheduled_change.effective_at, dateLocale) }]} />
    </div> : blocker === "payment" || blocker === "unavailable" ? <div className={styles.feedback}>
      <p className={styles.note}>{t(restriction!)}</p>
      {paymentURL ? <a className="button primary" href={paymentURL} target="_blank" rel="noopener noreferrer">{t({ en: "Complete payment", zh: "完成付款" })}</a>
        : <button className="button primary" disabled={action.busy} onClick={openPaymentSettings}>{action.busy ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Review billing", zh: "处理账单" })}</button>}
    </div> : choice === "current" ? <p className={styles.note} role="status">{t({ en: "This is your current plan.", zh: "这是你的当前套餐。" })}</p>
      : !configured ? <p className={styles.note}>{t({ en: "This plan is currently unavailable.", zh: "此套餐暂不可用。" })}</p>
        : !previewable ? <p className={styles.note}>{t(restriction || { en: "Review your subscription in billing before choosing another plan.", zh: "请先在订阅与账单中查看当前订阅，再选择其他套餐。" })} <Link href="/billing">{t(billingText.viewBilling)}</Link></p>
          : validPreview ? <div className={styles.summary} aria-live="polite">
            <div className={styles.total}><span>{t({ en: "Due now", zh: "当前应付" })}</span><strong>{subscriptionPreviewAmount(validPreview, dateLocale)}</strong></div>
            <DetailRows rows={[
              ...(Number.isSafeInteger(validPreview.unused_credit_amount) && validPreview.unused_credit_amount! > 0 ? [{ label: t({ en: "Unused time credit", zh: "未使用费用抵扣" }), value: `−${formatBillingAmount(validPreview.unused_credit_amount!, validPreview.currency, dateLocale)}` }] : []),
              { label: t({ en: "Change timing", zh: "生效时间" }), value: validPreview.effective_at_period_end ? `${t(changeTimingText(validPreview).label)} · ${formatDate(validPreview.current_period_end || validPreview.next_billing_at, dateLocale)}` : t(changeTimingText(validPreview).label) },
              ...(!validPreview.effective_at_period_end ? [{ label: t({ en: "Next billing date", zh: "下次计费" }), value: formatDate(validPreview.next_billing_at, dateLocale) }] : [])
            ]} />
            <p className={styles.note}>{t(changeTimingText(validPreview).detail)}</p>
            {blocker === "scheduled" ? <p className={styles.note}>{t({ en: "Confirming replaces your previously scheduled change.", zh: "确认后将替换原先预约的套餐变更。" })}</p> : null}
            {paymentCopy ? <p className={styles.note}>{t(paymentCopy)}</p> : null}
          </div> : preview.status === "error" ? <BillingFailure failure={preview.failure} onRetry={() => void loadPreview()} />
            : preview.status === "ready" ? <BillingFailure failure={{ kind: "unavailable", title: "", message: "", retryable: true }} onRetry={() => void loadPreview()} />
              : <p className={styles.note} role="status">{t({ en: "Loading the price preview…", zh: "正在获取费用预览…" })}</p>}
    {!result && blocker === "cancellation" ? <p className={styles.note} role="status">{t({ en: "Renewal is canceled. Resume renewal for your current plan to continue, then review and confirm this change separately.", zh: "已取消续费。先恢复当前套餐续费，再核对费用并确认变更。" })}</p> : null}
    {!result && !kept && !alreadyScheduled && reviewRequired ? <p className={styles.note} role="status">{t({ en: "Billing details changed. Review the updated details before confirming again.", zh: "账单信息已变化，请核对最新费用和生效时间后重新确认。" })}</p> : null}
    {pendingError ? <p className={styles.note} role="alert">{t({ en: "Your billing status has changed. Review billing before trying again.", zh: "订阅状态已变更，请先查看账单后重试。" })} <Link href="/billing">{t(billingText.viewBilling)}</Link></p> : <ConsoleError error={action.error} />}
    <div className={styles.actions}>
      {!result && !kept && !alreadyScheduled && blocker !== "payment" && blocker !== "unavailable" ? keepRequested
        ? <button className="button primary" type="button" disabled={!keepConfirmable} onClick={keepPlan}>{t(action.busy ? cancelPlanChangeText.submitting : planActionText("keep", planName(plan), interval))}</button>
        : blocker === "cancellation"
        ? <button className="button primary" type="button" disabled={action.busy || !configured} onClick={resume}>{action.busy ? t({ en: "Restoring…", zh: "恢复中…" }) : t({ en: "Resume renewal to continue", zh: "恢复续费并继续" })}</button>
        : <button className="button primary" type="button" disabled={!confirmable} onClick={confirm}>{action.busy ? t({ en: "Submitting…", zh: "正在提交…" }) : t(changeSubmitText({ choice, configured, phase: preview.status, preview: validPreview, amount: validPreview ? subscriptionPreviewAmount(validPreview, dateLocale) : "" }))}</button> : null}
      <button className="button" type="button" disabled={action.busy} onClick={onClose}>{t(result || kept || alreadyScheduled ? { en: "Done", zh: "完成" } : keepRequested ? cancelPlanChangeText.back : { en: "Cancel", zh: "取消" })}</button>
    </div>
  </ResponsiveDialog>;
}
