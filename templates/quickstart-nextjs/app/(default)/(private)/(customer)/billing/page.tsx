"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { BillingHistory } from "@/components/billing-history";
import { SignInDialog } from "@/components/sign-in-dialog";
import { SubscriptionChangeDialog } from "@/components/subscription-change-dialog";
import { SiteShell } from "@/components/site-shell";
import { Notice, Panel } from "@/components/ui";
import { SignInRequired } from "@/components/resource-feedback";
import { BillingFailure } from "@/components/billing-failure";
import { billingPrimaryAction, billingPrimaryText, billingText, subscriptionStatusText, type BillingRequestFailure } from "@/lib/billing-presentation";
import {
  ApiError,
  cancelSubscription,
  cancelScheduledChange,
  getCapabilities,
  getMembership,
  getSubscription,
  reactivateSubscription,
  type CapabilitiesView,
  type SubscriptionView
} from "@/lib/api";
import {
  readSession,
  SESSION_EVENT
} from "@/lib/auth";
import {
  CHECKOUT_REFRESH_DELAYS_MS,
  checkoutReturnStatus,
  hasOngoingRecurringSubscription
} from "@/lib/billing-state";
import { appEnv } from "@/lib/env";
import { readBillingIntent, type BillingIntent } from "@/lib/billing-intent";
import { billingSelectionPath, isBillingIntentAvailable } from "@/lib/billing-selection";
import { formatDate } from "@/lib/format";
import { localizeHref } from "@/lib/locale";
import { useI18n } from "@/lib/i18n";
import {
  describeRequestFailure,
  idleResource,
  loadingResource,
  startResourceRefresh,
  reconcileResourceRefresh,
  settleResource,
  type RequestFailure,
  type ResourceState
} from "@/lib/request-state";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";
import { ConsoleError, useConsoleResource } from "@/components/console-kit";
import { usePageReturn } from "@/components/use-page-return";
import { getCreditBalance } from "@/lib/operations-api";
import { pricingPurchaseHref } from "@/lib/pricing-checkout";
import { cancelRenewalForSubscription, subscriptionRenewalAction } from "@/lib/subscription-management";
import styles from "./billing.module.css";

type FeedbackTarget = "current" | "plans";

const defaultPlan = ["starter", "pro", "premium", "lifetime"].includes(appEnv.defaultPlan)
  ? appEnv.defaultPlan
  : "pro";

const defaultInterval = appEnv.defaultInterval === "yearly" ? "yearly" : "monthly";

const signedOutFailure: RequestFailure = {
  kind: "auth",
  title: "Sign in to continue",
  message: "Billing actions require a valid account session.",
  retryable: false
};

export default function BillingPage() {
  const { locale, t } = useI18n();
  const dateLocale = locale === "zh" ? "zh-CN" : "en-US";
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [sessionReady, setSessionReady] = useState(false);
  const [signInOpen, setSignInOpen] = useState(false);
  const [cancelConfirmOpen, setCancelConfirmOpen] = useState(false);
  const [feedbackTarget, setFeedbackTarget] = useState<FeedbackTarget>("current");
  const [intentRequest, setIntentRequest] = useState<{ intent: BillingIntent | null } | null>(null);
  const [plansOpen, setPlansOpen] = useState(false);
  const [historyRevision, setHistoryRevision] = useState(0);
  const [busy, setBusy] = useState<"" | "cancel" | "reactivate">("");
  const [status, setStatus] = useState("");
  const [actionFailure, setActionFailure] = useState<BillingRequestFailure | null>(null);
  const [capabilitiesState, setCapabilitiesState] = useState<ResourceState<CapabilitiesView>>(idleResource());
  const [subscriptionState, setSubscriptionState] = useState<ResourceState<SubscriptionView>>(idleResource());
  const [plan, setPlan] = useState(defaultPlan);
  const [interval, setInterval] = useState(defaultInterval);
  const [checkoutReturn, setCheckoutReturn] = useState<"success" | "cancelled" | null>(null);
  const [checkoutRefreshActive, setCheckoutRefreshActive] = useState(false);
  const [checkoutRefreshComplete, setCheckoutRefreshComplete] = useState(false);
  const capabilitiesGenerationRef = useRef(0);
  const capabilitiesMountedRef = useRef(false);
  const subscriptionGenerationRef = useRef(0);
  const actionGenerationRef = useRef(0);
  const checkoutRefreshGenerationRef = useRef(0);
  const checkoutRefreshTimerRef = useRef<number | null>(null);
  const cancelTriggerRef = useRef<HTMLButtonElement | null>(null);
  const subscription = subscriptionState.data;
  const billingCapability = capabilitiesState.data?.billing;
  const subscriptionOffers = billingCapability?.offers.subscriptions || [];
  const availablePlans = useMemo(() => (billingCapability?.offers.subscriptions || []).map(offer => offer.plan), [billingCapability]);
  const selectedOffer = subscriptionOffers.find((offer) => offer.plan === plan);
  const availableIntervals = useMemo(
    () => selectedOffer?.intervals || [],
    [selectedOffer]
  );
  const ongoingRecurringSubscription = hasOngoingRecurringSubscription(subscription);
  const renewalAction = subscriptionRenewalAction(subscription);
  const cancellationPending = renewalAction === "renew";
  const selectedLifetime = plan === "lifetime";
  const billingReady = capabilitiesState.status === "ready" && capabilitiesState.data.billing.enabled;
  const intentUnavailable = intentRequest !== null && (
    !intentRequest.intent || (capabilitiesState.status === "ready" && !isBillingIntentAvailable(intentRequest.intent, billingCapability))
  );
  const billingDisabled = (capabilitiesState.status === "ready" && !capabilitiesState.data.billing.enabled)
    || (subscriptionState.status === "error"
      && (subscriptionState.failure.kind === "disabled" || subscriptionState.failure.kind === "configuration"));
  const baseActionDisabled = !session || !billingReady || billingDisabled || busy !== "";
  const billingCapabilityFailure: RequestFailure | null = capabilitiesState.status === "error"
    ? capabilitiesState.failure
    : capabilitiesState.status === "ready" && !capabilitiesState.data.billing.enabled
      ? {
          kind: "configuration",
          title: t({ en: "Billing is currently unavailable", zh: "账单服务暂不可用" }),
          message: t({ en: "Please contact support if you need help with a payment.", zh: "如需处理付款问题，请联系支持。" }),
          retryable: false
        }
      : null;
  const sessionToken = session?.token || "";
  const membership = useConsoleResource(sessionToken, getMembership);
  const credits = useConsoleResource(sessionToken, getCreditBalance);
  const refreshMembership = membership.refresh;
  const refreshCredits = credits.refresh;
  const revalidateMembership = membership.revalidate;
  const revalidateCredits = credits.revalidate;
  const upgradeAction = billingPrimaryAction(membership.data?.plan, subscription);
  const hasLifetime = upgradeAction === "lifetime";
  const upgradeLoading = membership.loading || upgradeAction === "loading" || (!subscription && subscriptionState.status !== "error");

  function planName(value?: string): string {
    switch ((value || "").toLowerCase()) {
      case "starter": return t({ en: "Starter", zh: "入门版" });
      case "pro": return t({ en: "Pro", zh: "专业版" });
      case "premium": return t({ en: "Premium", zh: "高级版" });
      case "lifetime": return t({ en: "Lifetime", zh: "终身版" });
      case "free": return t({ en: "Free", zh: "免费版" });
      default: return value || t({ en: "Free", zh: "免费版" });
    }
  }

  function intervalName(value?: string): string {
    if (value === "monthly") return t({ en: "Monthly", zh: "按月" });
    if (value === "yearly") return t({ en: "Yearly", zh: "按年" });
    return value || "—";
  }

  function statusName(value?: string): string {
    return t(subscriptionStatusText(value || ""));
  }

  const loadSubscriptionState = useCallback(async (token: string, background = false) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(subscriptionGenerationRef);
    setSubscriptionState(current => startResourceRefresh(current, background));
    const nextState = await settleResource(getSubscription(token), "Subscription billing");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(subscriptionGenerationRef, generation)
    ) {
      setSubscriptionState(current => reconcileResourceRefresh(current, nextState, background));
    }
  }, []);

  const loadBillingState = useCallback(async (token: string, background = false) => {
    await Promise.all([
      loadSubscriptionState(token, background),
      background ? revalidateMembership() : refreshMembership(),
      background ? revalidateCredits() : refreshCredits()
    ]);
    if (readSession()?.token === token) setHistoryRevision(value => value + 1);
  }, [loadSubscriptionState, refreshMembership, refreshCredits, revalidateMembership, revalidateCredits]);

  const updateOnReturn = useCallback(() => {
    if (sessionToken && billingReady) void loadBillingState(sessionToken, true);
    else { void revalidateMembership(); void revalidateCredits(); setHistoryRevision(value => value + 1); }
  }, [revalidateMembership, revalidateCredits, sessionToken, billingReady, loadBillingState]);
  usePageReturn(updateOnReturn);

  useEffect(() => {
    const syncSession = () => {
      setSession(readSession());
      setSessionReady(true);
    };
    syncSession();
    window.addEventListener("storage", syncSession);
    window.addEventListener(SESSION_EVENT, syncSession);
    return () => {
      window.removeEventListener("storage", syncSession);
      window.removeEventListener(SESSION_EVENT, syncSession);
    };
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (["#plans", "#subscription-checkout"].includes(window.location.hash)) setPlansOpen(true);
    if (["product", "plan", "interval"].some(key => params.has(key))) {
      const intent = readBillingIntent(window.location.search);
      setIntentRequest({ intent });
      setPlansOpen(intent?.product === "subscription");
      if (intent?.product === "subscription") {
        setPlan(intent.plan);
        setInterval(intent.interval);
      } else if (intent?.product === "lifetime") {
        setPlan("lifetime");
      }
    }
    const returnStatus = checkoutReturnStatus(window.location.search);
    setCheckoutReturn(returnStatus);
    if (returnStatus) {
      const url = new URL(window.location.href);
      url.searchParams.delete("checkout");
      window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
    }

    return () => {
      invalidateRequestGeneration(checkoutRefreshGenerationRef);
      if (checkoutRefreshTimerRef.current !== null) {
        window.clearTimeout(checkoutRefreshTimerRef.current);
        checkoutRefreshTimerRef.current = null;
      }
    };
  }, []);

  useEffect(() => {
    capabilitiesMountedRef.current = true;
    void loadCapabilitiesState();
    return () => {
      capabilitiesMountedRef.current = false;
      invalidateRequestGeneration(capabilitiesGenerationRef);
    };
  }, []);

  useEffect(() => {
    invalidateRequestGeneration(actionGenerationRef);
    setBusy("");
    setStatus("");
    setActionFailure(null);
    setCancelConfirmOpen(false);
    if (!sessionToken || capabilitiesState.status !== "ready" || !capabilitiesState.data.billing.enabled) {
      invalidateRequestGeneration(subscriptionGenerationRef);
      setSubscriptionState(idleResource());
      return;
    }
    void loadSubscriptionState(sessionToken);
    return () => {
      invalidateRequestGeneration(subscriptionGenerationRef);
      invalidateRequestGeneration(actionGenerationRef);
    };
  }, [sessionToken, capabilitiesState, loadSubscriptionState]);

  useEffect(() => {
    const intent = intentRequest?.intent;
    if (!intent || (intent.product === "subscription" && subscriptionState.status !== "ready")) return;
    if (intent.product !== "subscription" || !ongoingRecurringSubscription) {
      window.location.replace(pricingPurchaseHref(intent, locale));
    }
  }, [intentRequest, subscriptionState.status, ongoingRecurringSubscription, locale]);

  useEffect(() => {
    invalidateRequestGeneration(checkoutRefreshGenerationRef);
    if (checkoutRefreshTimerRef.current !== null) {
      window.clearTimeout(checkoutRefreshTimerRef.current);
      checkoutRefreshTimerRef.current = null;
    }
    setCheckoutRefreshActive(false);
    setCheckoutRefreshComplete(false);

    if (checkoutReturn !== "success" || !sessionToken || !billingReady) {
      return;
    }

    const generation = beginRequestGeneration(checkoutRefreshGenerationRef);
    let nextAttempt = 0;

    const scheduleRefresh = () => {
      const delay = CHECKOUT_REFRESH_DELAYS_MS[nextAttempt];
      checkoutRefreshTimerRef.current = window.setTimeout(async () => {
        checkoutRefreshTimerRef.current = null;
        if (
          readSession()?.token !== sessionToken
          || !isCurrentRequestGeneration(checkoutRefreshGenerationRef, generation)
        ) {
          return;
        }

        setCheckoutRefreshActive(true);
        await loadBillingState(sessionToken, true);
        if (
          readSession()?.token !== sessionToken
          || !isCurrentRequestGeneration(checkoutRefreshGenerationRef, generation)
        ) {
          return;
        }

        nextAttempt += 1;
        if (nextAttempt < CHECKOUT_REFRESH_DELAYS_MS.length) {
          scheduleRefresh();
          return;
        }

        setCheckoutRefreshActive(false);
        setCheckoutRefreshComplete(true);
      }, delay);
    };

    scheduleRefresh();
    return () => {
      invalidateRequestGeneration(checkoutRefreshGenerationRef);
      if (checkoutRefreshTimerRef.current !== null) {
        window.clearTimeout(checkoutRefreshTimerRef.current);
        checkoutRefreshTimerRef.current = null;
      }
    };
  }, [billingReady, checkoutReturn, loadBillingState, sessionToken]);

  useEffect(() => {
    if (intentRequest !== null || capabilitiesState.status !== "ready" || !capabilitiesState.data.billing.enabled) {
      return;
    }
    if (availablePlans.length > 0 && !availablePlans.includes(plan)) {
      setPlan(availablePlans[0]);
    }
  }, [availablePlans, capabilitiesState, plan, intentRequest]);

  useEffect(() => {
    if (intentRequest !== null || selectedLifetime || availableIntervals.length === 0 || availableIntervals.includes(interval)) {
      return;
    }
    setInterval(availableIntervals[0]);
  }, [availableIntervals, interval, selectedLifetime, intentRequest]);

  async function loadCapabilitiesState() {
    if (!capabilitiesMountedRef.current) return;
    const generation = beginRequestGeneration(capabilitiesGenerationRef);
    setCapabilitiesState((current) => loadingResource(current.data));
    const nextState = await settleResource(getCapabilities(), "API capabilities");
    if (
      capabilitiesMountedRef.current
      && isCurrentRequestGeneration(capabilitiesGenerationRef, generation)
    ) {
      setCapabilitiesState(nextState);
    }
  }

  function handleActionError(error: unknown, capability: string) {
    if (error instanceof ApiError && error.reason === "subscription_change_pending") {
      setActionFailure({ kind: "unknown", title: "", message: "", reason: "subscription_change_pending", retryable: true });
      return;
    }
    setActionFailure(describeRequestFailure(error, capability));
  }

  function canCommitAction(token: string, generation: number): boolean {
    return readSession()?.token === token
      && isCurrentRequestGeneration(actionGenerationRef, generation);
  }

  async function handleCancel() {
    setFeedbackTarget("current");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("cancel");
    setActionFailure(null);
    setStatus("");

    try {
      const result = await cancelRenewalForSubscription({
        load: () => getSubscription(requestToken),
        removeSchedule: () => cancelScheduledChange(requestToken),
        cancel: () => cancelSubscription(requestToken, "end_of_period"),
        current: () => canCommitAction(requestToken, actionGeneration)
      });
      if (!result || !canCommitAction(requestToken, actionGeneration)) return;
      if (result.state === "blocked") {
        setActionFailure({ kind: "unknown", title: "", message: "", reason: "subscription_change_pending", retryable: true });
        await loadBillingState(requestToken, true);
        return;
      }
      setStatus(result.state === "unconfirmed"
        ? t({ en: "Confirming cancellation. Check your subscription status before trying again.", zh: "正在确认取消结果，请先核对订阅状态，避免重复操作。" })
        : result.state === "ended"
          ? t({ en: "This subscription has ended.", zh: "此订阅已结束。" })
          : result.subscription.current_period_end
            ? t({ en: `Renewal canceled. Access continues until ${formatDate(result.subscription.current_period_end, dateLocale)}.`, zh: `已取消续费，权益保留至 ${formatDate(result.subscription.current_period_end, dateLocale)}。` })
            : t({ en: "Renewal canceled. Your current paid period remains available.", zh: "已取消续费，当前已付费周期内仍可使用。" }));
      setCancelConfirmOpen(false);
      await loadBillingState(requestToken, true);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) {
        handleActionError(err, "Subscription cancellation");
        await loadBillingState(requestToken, true);
      }
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleReactivate() {
    setFeedbackTarget("current");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("reactivate");
    setActionFailure(null);
    setStatus("");

    try {
      await reactivateSubscription(requestToken);
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(t({ en: "Your subscription will renew again.", zh: "已恢复订阅续费。" }));
      await loadBillingState(requestToken, true);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription reactivation");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  function actionFeedback(target: FeedbackTarget) {
    if (feedbackTarget !== target) return null;
    return <div className={styles.feedback}>
      {status ? <div role="status"><Notice tone="success">{status}</Notice></div> : null}
      {actionFailure ? <BillingFailure failure={actionFailure} onRetry={() => session ? void loadBillingState(session.token, true) : undefined} /> : null}
    </div>;
  }

  function changeSelection(nextPlan: string, nextInterval = interval) {
    const intervals = billingCapability?.offers.subscriptions.find(offer => offer.plan === nextPlan)?.intervals || [];
    const selectedInterval = intervals.includes(nextInterval) ? nextInterval : intervals[0] || nextInterval;
    const intent: BillingIntent | null = nextPlan === "lifetime"
      ? { product: "lifetime" }
      : selectedInterval === "monthly" || selectedInterval === "yearly"
        ? { product: "subscription", plan: nextPlan, interval: selectedInterval }
        : null;
    if (intent) syncSelectionURL(intent);
    setIntentRequest(null);
    setPlansOpen(true);
    setPlan(nextPlan);
    setInterval(selectedInterval);
    setStatus("");
    setActionFailure(null);
  }

  function syncSelectionURL(intent: BillingIntent) {
    const path = billingSelectionPath(window.location.search, intent, billingCapability);
    if (path) window.history.replaceState(window.history.state, "", path);
  }

  function primaryPlanControl() {
    if (hasLifetime) return <button className="button" disabled>{t(billingText.lifetime)}</button>;
    if (upgradeLoading) return <button className="button primary" disabled>{t(billingText.loading)}</button>;
    if (subscriptionState.status === "error" || billingDisabled) return <button className="button" disabled>{t({ en: "Plan unavailable", zh: "暂时无法读取套餐" })}</button>;
    const label = t(billingPrimaryText(upgradeAction));
    if (ongoingRecurringSubscription && subscription) return <button className="button primary" disabled={baseActionDisabled} onClick={() => {
      setPlan(subscription.plan); setInterval(subscription.billing_cycle || defaultInterval);
      setStatus(""); setActionFailure(null); setPlansOpen(true);
    }}>{t({ en: "Change plan", zh: "更改套餐" })}</button>;
    if (baseActionDisabled) return <button className="button primary" disabled>{label}</button>;
    return <Link className="button primary" href={localizeHref("/pricing", locale)}>{label}</Link>;
  }

  return (
    <SiteShell
      eyebrow={t({ en: "Account", zh: "账户" })}
      title={t({ en: "Subscription & billing", zh: "订阅与账单" })}
      description={t({ en: "Your plan, credit balance and payment history.", zh: "查看当前套餐、剩余积分和支付记录。" })}
      accountMenuData={{ capabilities: capabilitiesState, subscription: subscriptionState }}
    >
      <div className={`page-grid ${styles.layout}`}>
        {sessionReady && !session ? (
          <div className="span-12">
            <SignInRequired
              title={t({ en: "Sign in to continue", zh: "登录后继续" })}
              actionLabel={t({ en: "Sign in", zh: "登录" })}
              onSignIn={() => setSignInOpen(true)}
              message={t({ en: "Your selection will be kept while you sign in to review your billing.", zh: "登录后查看订阅与账单，已选择的商品会为你保留。" })}
            />
          </div>
        ) : null}
        {billingCapabilityFailure ? (
          <div className="span-12"><BillingFailure failure={billingCapabilityFailure} onRetry={capabilitiesState.status === "error" ? () => void loadCapabilitiesState() : undefined} /></div>
        ) : null}
        {intentUnavailable ? (
          <div className="span-12" role="status">
            <Notice>
              <strong>{t({ en: "This selection is unavailable", zh: "原选择暂不可购买" })}</strong>
              <p>{t({ en: "The linked product or billing interval is invalid or no longer offered. Review the available options and choose again before continuing.", zh: "链接中的商品或计费周期无效，或当前尚未开放。请查看可用方案并重新选择后继续。" })}</p>
              <Link className="button" href={localizeHref("/pricing", locale)}>{t({ en: "View plans", zh: "查看套餐" })}</Link>
            </Notice>
          </div>
        ) : null}
        {checkoutReturn === "success" ? (
          <div className="span-12" role="status">
            <Notice>
              {checkoutRefreshActive
                ? t(billingText.checkoutChecking)
                : checkoutRefreshComplete
                  ? t(billingText.checkoutWaiting)
                  : session
                    ? t(billingText.checkoutChecking)
                    : t({ en: "You have returned from checkout. Sign in to view the payment result.", zh: "你已返回网站，请登录查看付款结果。" })}
            </Notice>
          </div>
        ) : checkoutReturn === "cancelled" ? (
          <div className="span-12"><Notice>{t(billingText.checkoutCanceled)}</Notice></div>
        ) : null}

        {session ? <Panel className={`span-12 ${styles.overview}`} title={t({ en: "Plan & credits", zh: "套餐与积分" })}>
          <div id="subscription-state" className={styles.anchor} />
          <div className={styles.summary}>
            <div className={styles.planIdentity}>
              <span className={styles.label}>{t({ en: "Current plan", zh: "当前套餐" })}</span>
              <div className={styles.planHeading}><strong>{membership.data || subscription ? planName(membership.data?.plan || subscription?.plan) : "—"}</strong>{subscription && ongoingRecurringSubscription ? <span className={styles.status}>{statusName(subscription.status)}</span> : null}</div>
              {subscription && ongoingRecurringSubscription ? <p>{intervalName(subscription.billing_cycle)} · {cancellationPending ? t({ en: "Access until", zh: "权益保留至" }) : subscription.pending_payment || ["past_due", "incomplete", "unpaid", "payment_failed", "trialing"].includes(subscription.status) ? t({ en: "Current period ends", zh: "本期结束" }) : t({ en: "Next renewal", zh: "下次续费" })} {formatDate(subscription.effective_cancel_at || subscription.current_period_end, dateLocale)}</p> : null}
              {hasLifetime ? <p>{t({ en: "Lifetime access", zh: "永久有效" })}</p> : null}
            </div>
            <div className={styles.creditBalance}>
              <span className={styles.label}>{t({ en: "Remaining credits", zh: "剩余积分" })}</span>
              <strong>{credits.data ? new Intl.NumberFormat(dateLocale).format(credits.data.balance) : "—"}</strong>
              <Link href="/credits">{t({ en: "Credit history", zh: "积分变更记录" })} →</Link>
            </div>
          </div>
          <ConsoleError error={membership.error} /><ConsoleError error={credits.error} />
          {subscription ? <>
              {subscription.pending_payment ? (
                <div className={styles.stateCard} role="status">
                  <strong>{t({ en: "Complete payment to change your plan", zh: "完成付款后变更套餐" })}</strong>
                  <p>{t({ en: "Open Change plan to complete payment. The new plan takes effect after payment is confirmed.", zh: "打开“更改套餐”完成付款，付款确认后新套餐才会生效。" })}</p>
                </div>
              ) : null}
              {subscription.scheduled_change ? (
                <div className={styles.stateCard}>
                  <strong>{t({ en: "Upcoming plan change", zh: "已预约套餐变更" })}</strong>
                  <p>{planName(subscription.scheduled_change.plan)} · {intervalName(subscription.scheduled_change.billing_cycle)} · {formatDate(subscription.scheduled_change.effective_at, dateLocale)}</p>
                </div>
              ) : null}
              {cancellationPending ? <p className={styles.stateNote}>{t(billingText.canceledRenewal)}</p> : null}
          </> : null}
              <div className={`button-row ${styles.manageActions}`}>
                {primaryPlanControl()}
                {!hasLifetime && renewalAction === "renew" ? <button className="button" disabled={baseActionDisabled} onClick={() => void handleReactivate()}>{busy === "reactivate" ? t({ en: "Renewing…", zh: "恢复中…" }) : t({ en: "Renew", zh: "恢复续费" })}</button> : null}
                {!hasLifetime && renewalAction === "cancel" ? <button ref={cancelTriggerRef} className={`button ${styles.cancelTrigger}`} disabled={baseActionDisabled} aria-expanded={cancelConfirmOpen} aria-controls="cancel-renewal-confirmation" onClick={() => setCancelConfirmOpen(value => !value)}>{t({ en: "Cancel subscription", zh: "取消订阅" })}</button> : null}
              </div>
              {subscription && !hasLifetime && cancelConfirmOpen && ongoingRecurringSubscription && !cancellationPending ? (
                <div id="cancel-renewal-confirmation" className={styles.confirmation} role="group" aria-labelledby="cancel-renewal-title">
                  <h3 id="cancel-renewal-title">{t({ en: "Cancel your next renewal?", zh: "确认取消后续续费？" })}</h3>
                  <p>{subscription.current_period_end
                    ? t({ en: `Your plan remains available until ${formatDate(subscription.current_period_end, dateLocale)}. It will not renew after that date.`, zh: `当前套餐可继续使用至 ${formatDate(subscription.current_period_end, dateLocale)}，此后不再自动续费。` })
                    : t({ en: "Your plan remains available until the end of the current paid period and will not renew. The exact end date will appear once billing is refreshed.", zh: "当前已付费周期内仍可使用，周期结束后不再续费。刷新账单后可查看具体结束日期。" })}</p>
                  {subscription.scheduled_change ? <p>{t({ en: "Your scheduled plan change will also be canceled.", zh: "尚未生效的套餐预约也会一并撤销。" })}</p> : null}
                  <p>{t({ en: "Existing credits are kept under their original expiry rules.", zh: "已有积分保留，原有效期照常。" })}</p>
                  <div className="button-row">
                    <button className="button" disabled={busy !== ""} onClick={() => { setCancelConfirmOpen(false); cancelTriggerRef.current?.focus(); }}>{t({ en: "Keep renewal", zh: "保留续费" })}</button>
                    <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel()}>{busy === "cancel" ? t({ en: "Canceling…", zh: "取消中…" }) : t({ en: "Confirm cancellation", zh: "确认取消续费" })}</button>
                  </div>
                </div>
              ) : null}
          {subscriptionState.status === "error" ? <BillingFailure failure={subscriptionState.failure} onRetry={() => session ? void loadSubscriptionState(session.token) : undefined} /> : null}
          {actionFeedback("current")}
        </Panel> : null}

        {plansOpen && ongoingRecurringSubscription && !hasLifetime && subscription ? <SubscriptionChangeDialog
          token={sessionToken} subscription={subscription} offers={subscriptionOffers} selection={{ plan, interval }}
          onSelection={selection => changeSelection(selection.plan, selection.interval)}
          onClose={() => setPlansOpen(false)} onChanged={() => loadBillingState(sessionToken, true)} /> : null}

        {session ? <BillingHistory key={session.token} token={session.token} revision={historyRevision} invoicesEnabled={billingReady} /> : null}
      </div>
      <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} onSuccess={nextSession => { setSession(nextSession); setSessionReady(true); }} />
    </SiteShell>
  );
}
