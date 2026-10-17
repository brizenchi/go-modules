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
  createBillingPortalSession,
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
  stripeInvoiceURL,
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
  const [busy, setBusy] = useState<"" | "change" | "portal" | "cancel" | "reactivate">("");
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
  const cancellationPending = ongoingRecurringSubscription
    && (Boolean(subscription?.cancel_at_period_end || subscription?.effective_cancel_at) || subscription?.status.trim().toLowerCase() === "canceling");
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

  async function handleOpenPortal(target: FeedbackTarget = "current") {
    setFeedbackTarget(target);
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("portal");
    setActionFailure(null);
    setStatus("");

    try {
      const res = await createBillingPortalSession(requestToken, `${appEnv.appUrl}/billing`);
      if (!canCommitAction(requestToken, actionGeneration)) return;
      window.location.href = res.url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Billing portal");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleCancel(cancelType: "end_of_period" | "3days") {
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
      await cancelSubscription(requestToken, cancelType);
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(subscription?.current_period_end
        ? t({ en: `Renewal canceled. Access continues until ${formatDate(subscription.current_period_end, dateLocale)}.`, zh: `已取消续费，权益保留至 ${formatDate(subscription.current_period_end, dateLocale)}。` })
        : t({ en: "Renewal canceled. Your current paid period remains available.", zh: "已取消续费，当前已付费周期内仍可使用。" }));
      setCancelConfirmOpen(false);
      await loadBillingState(requestToken, true);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription cancellation");
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
    if (upgradeAction === "scheduled" || (upgradeAction === "payment" && subscription?.pending_payment)) return null;
    const label = t(billingPrimaryText(upgradeAction));
    if (upgradeAction === "change" && subscription) return <button className="button primary" disabled={baseActionDisabled} onClick={() => {
      setPlan(subscription.plan); setInterval(subscription.billing_cycle || defaultInterval);
      setStatus(""); setActionFailure(null); setPlansOpen(true);
    }}>{label}</button>;
    if (upgradeAction === "resume") return <button className="button primary" disabled={baseActionDisabled} onClick={handleReactivate}>{busy === "reactivate" ? t({ en: "Restoring…", zh: "恢复中…" }) : label}</button>;
    if (upgradeAction === "payment" || upgradeAction === "review") return <button className="button primary" disabled={baseActionDisabled} onClick={() => void handleOpenPortal()}>{busy === "portal" ? t({ en: "Opening…", zh: "正在打开…" }) : label}</button>;
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
              {subscription && ongoingRecurringSubscription ? <p>{intervalName(subscription.billing_cycle)} · {cancellationPending ? t({ en: "Access until", zh: "权益保留至" }) : upgradeAction === "payment" || subscription.status === "trialing" ? t({ en: "Current period ends", zh: "本期结束" }) : t({ en: "Next renewal", zh: "下次续费" })} {formatDate(subscription.effective_cancel_at || subscription.current_period_end, dateLocale)}</p> : null}
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
                  <p>{t({ en: "The new plan takes effect after payment is confirmed.", zh: "付款确认后，新套餐才会生效。" })}</p>
                  {stripeInvoiceURL(subscription.pending_payment.invoice_url)
                    ? <a className="button primary" href={stripeInvoiceURL(subscription.pending_payment.invoice_url)!} target="_blank" rel="noopener noreferrer">{t({ en: "Complete payment", zh: "完成付款" })}</a>
                    : <button className="button primary" disabled={baseActionDisabled} onClick={() => void handleOpenPortal()}>{t({ en: "Review payment", zh: "处理付款" })}</button>}
                </div>
              ) : null}
              {subscription.scheduled_change ? (
                <div className={styles.stateCard}>
                  <strong>{t({ en: "Upcoming plan change", zh: "已预约套餐变更" })}</strong>
                  <p>{planName(subscription.scheduled_change.plan)} · {intervalName(subscription.scheduled_change.billing_cycle)} · {formatDate(subscription.scheduled_change.effective_at, dateLocale)}</p>
                  <button className="button" disabled={baseActionDisabled} onClick={async () => {
                    if (!session) return;
                    const token = session.token;
                    const generation = beginRequestGeneration(actionGenerationRef);
                    setFeedbackTarget("current"); setBusy("change"); setActionFailure(null); setStatus("");
                    try {
                      await cancelScheduledChange(token);
                      if (canCommitAction(token, generation)) {
                        setStatus(t({ en: "Scheduled change removed. Your current plan is unchanged.", zh: "已撤销预约变更，当前套餐保持不变。" }));
                        await loadBillingState(token, true);
                      }
                    } catch (err) { if (canCommitAction(token, generation)) handleActionError(err, "Cancel scheduled change"); }
                    finally { if (canCommitAction(token, generation)) setBusy(""); }
                  }}>{busy === "change" ? t({ en: "Canceling change…", zh: "正在撤销…" }) : t({ en: "Cancel scheduled change", zh: "撤销预约变更" })}</button>
                </div>
              ) : null}
              {cancellationPending ? <p className={styles.stateNote}>{t(billingText.canceledRenewal)}</p> : null}
          </> : null}
              <div className={`button-row ${styles.manageActions}`}>
                {primaryPlanControl()}
                {subscription && (ongoingRecurringSubscription || hasLifetime) ? <button className="button" disabled={baseActionDisabled} onClick={() => void handleOpenPortal()}>
                  {busy === "portal" ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Billing settings", zh: "账单设置" })}
                </button> : null}
                {!hasLifetime && ongoingRecurringSubscription && !cancellationPending ? <button ref={cancelTriggerRef} className={`button ${styles.cancelTrigger}`} disabled={baseActionDisabled} aria-expanded={cancelConfirmOpen} aria-controls="cancel-renewal-confirmation" onClick={() => setCancelConfirmOpen(value => !value)}>{t({ en: "Cancel renewal", zh: "取消续费" })}</button> : null}
              </div>
              {subscription && !hasLifetime && cancelConfirmOpen && ongoingRecurringSubscription && !cancellationPending ? (
                <div id="cancel-renewal-confirmation" className={styles.confirmation} role="group" aria-labelledby="cancel-renewal-title">
                  <h3 id="cancel-renewal-title">{t({ en: "Cancel your next renewal?", zh: "确认取消后续续费？" })}</h3>
                  <p>{subscription.current_period_end
                    ? t({ en: `Your plan remains available until ${formatDate(subscription.current_period_end, dateLocale)}. It will not renew after that date.`, zh: `当前套餐可继续使用至 ${formatDate(subscription.current_period_end, dateLocale)}，此后不再自动续费。` })
                    : t({ en: "Your plan remains available until the end of the current paid period and will not renew. The exact end date will appear once billing is refreshed.", zh: "当前已付费周期内仍可使用，周期结束后不再续费。刷新账单后可查看具体结束日期。" })}</p>
                  <p>{t({ en: "Existing credits are kept under their original expiry rules.", zh: "已有积分保留，原有效期照常。" })}</p>
                  <div className="button-row">
                    <button className="button" disabled={busy !== ""} onClick={() => { setCancelConfirmOpen(false); cancelTriggerRef.current?.focus(); }}>{t({ en: "Keep renewal", zh: "保留续费" })}</button>
                    <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("end_of_period")}>{busy === "cancel" ? t({ en: "Canceling…", zh: "取消中…" }) : t({ en: "Confirm cancellation", zh: "确认取消续费" })}</button>
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
