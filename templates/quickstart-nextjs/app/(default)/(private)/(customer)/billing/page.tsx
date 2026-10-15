"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { BillingHistory } from "@/components/billing-history";
import { SignInDialog } from "@/components/sign-in-dialog";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, LoadingState, Notice, Panel, DetailRows } from "@/components/ui";
import { ResourceFailure, SignInRequired } from "@/components/resource-feedback";
import {
  ApiError,
  cancelSubscription,
  cancelScheduledChange,
  changeSubscription,
  createBillingPortalSession,
  createCheckoutSession,
  getCapabilities,
  getSubscription,
  listInvoices,
  previewSubscriptionChange,
  reactivateSubscription,
  type CapabilitiesView,
  type InvoiceItem,
  type SubscriptionChangeMode,
  type SubscriptionPreview,
  type SubscriptionView
} from "@/lib/api";
import {
  readReferralCode,
  readSession,
  REFERRAL_EVENT,
  SESSION_EVENT
} from "@/lib/auth";
import {
  CHECKOUT_REFRESH_DELAYS_MS,
  stripeInvoiceURL,
  checkoutReturnStatus,
  hasOngoingRecurringSubscription,
  parseCreditsQuantity
} from "@/lib/billing-state";
import { appEnv, appUrl } from "@/lib/env";
import { readBillingIntent, type BillingIntent } from "@/lib/billing-intent";
import { billingPlaceholderState, billingSelectionPath, initialBillingSection, isBillingIntentAvailable } from "@/lib/billing-selection";
import { formatCurrencyUSD, formatDate } from "@/lib/format";
import { localizeHref } from "@/lib/locale";
import { useI18n } from "@/lib/i18n";
import {
  describeRequestFailure,
  idleResource,
  loadingResource,
  settleResource,
  type RequestFailure,
  type ResourceState
} from "@/lib/request-state";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";
import styles from "./billing.module.css";

type FeedbackTarget = "current" | "plans" | "credits";

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
  const [creditsOpen, setCreditsOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [busy, setBusy] = useState<"" | "subscription" | "change" | "portal" | "credits" | "cancel" | "reactivate" | "refresh">("");
  const [status, setStatus] = useState("");
  const [actionFailure, setActionFailure] = useState<RequestFailure | null>(null);
  const [capabilitiesState, setCapabilitiesState] = useState<ResourceState<CapabilitiesView>>(idleResource());
  const [subscriptionState, setSubscriptionState] = useState<ResourceState<SubscriptionView>>(idleResource());
  const [previewState, setPreviewState] = useState<ResourceState<SubscriptionPreview>>(idleResource());
  const [invoiceState, setInvoiceState] = useState<ResourceState<InvoiceItem[]>>(idleResource());
  const [plan, setPlan] = useState(defaultPlan);
  const [interval, setInterval] = useState(defaultInterval);
  const [creditsQuantity, setCreditsQuantity] = useState(String(appEnv.defaultCreditsQuantity));
  const [referralCode, setReferralCode] = useState("");
  const [checkoutReturn, setCheckoutReturn] = useState<"success" | "cancelled" | null>(null);
  const [checkoutRefreshAttempt, setCheckoutRefreshAttempt] = useState(0);
  const [checkoutRefreshActive, setCheckoutRefreshActive] = useState(false);
  const [checkoutRefreshComplete, setCheckoutRefreshComplete] = useState(false);
  const capabilitiesGenerationRef = useRef(0);
  const capabilitiesMountedRef = useRef(false);
  const subscriptionGenerationRef = useRef(0);
  const invoiceGenerationRef = useRef(0);
  const previewGenerationRef = useRef(0);
  const actionGenerationRef = useRef(0);
  const checkoutRefreshGenerationRef = useRef(0);
  const checkoutRefreshTimerRef = useRef<number | null>(null);
  const cancelTriggerRef = useRef<HTMLButtonElement | null>(null);
  const initialDisclosureTokenRef = useRef<string | null>(null);
  const subscription = subscriptionState.data;
  const preview = previewState.data;
  const invoices = invoiceState.data || [];
  const billingCapability = capabilitiesState.data?.billing;
  const subscriptionOffers = billingCapability?.offers.subscriptions || [];
  const availablePlans = useMemo(() => [
    ...(billingCapability?.offers.subscriptions || []).map((offer) => offer.plan),
    ...(billingCapability?.offers.lifetime ? ["lifetime"] : [])
  ], [billingCapability]);
  const selectedOffer = subscriptionOffers.find((offer) => offer.plan === plan);
  const availableIntervals = useMemo(
    () => selectedOffer?.intervals || [],
    [selectedOffer]
  );
  const currentPlan = subscription?.plan || "";
  const hasLifetime = currentPlan === "lifetime";
  const ongoingRecurringSubscription = hasOngoingRecurringSubscription(subscription);
  const cancellationPending = ongoingRecurringSubscription
    && (Boolean(subscription?.cancel_at_period_end || subscription?.effective_cancel_at) || subscription?.status.trim().toLowerCase() === "canceling");
  const selectedLifetime = plan === "lifetime";
  const parsedCreditsQuantity = parseCreditsQuantity(creditsQuantity);
  const billingReady = capabilitiesState.status === "ready" && capabilitiesState.data.billing.enabled;
  const subscriptionPlaceholder = billingPlaceholderState({ sessionReady, signedIn: Boolean(session), capabilities: capabilitiesState.status, billingEnabled: billingReady, resource: subscriptionState.status });
  const invoicePlaceholder = billingPlaceholderState({ sessionReady, signedIn: Boolean(session), capabilities: capabilitiesState.status, billingEnabled: billingReady, resource: invoiceState.status });
  const intentUnavailable = intentRequest !== null && (
    !intentRequest.intent || (capabilitiesState.status === "ready" && !isBillingIntentAvailable(intentRequest.intent, billingCapability))
  );
  const billingDisabled = (capabilitiesState.status === "ready" && !capabilitiesState.data.billing.enabled)
    || (subscriptionState.status === "error"
      && (subscriptionState.failure.kind === "disabled" || subscriptionState.failure.kind === "configuration"));
  const baseActionDisabled = !session || !billingReady || billingDisabled || busy !== "";
  const selectedOfferConfigured = selectedLifetime
    ? Boolean(billingCapability?.offers.lifetime)
    : availableIntervals.includes(interval);
  const selectionConfigured = billingReady && availablePlans.includes(plan) && selectedOfferConfigured;
  const purchaseDisabled = baseActionDisabled
    || intentUnavailable
    || subscriptionState.status !== "ready"
    || !availablePlans.includes(plan)
    || !selectedOfferConfigured;
  const lifetimePurchaseDisabled = baseActionDisabled
    || intentUnavailable
    || subscriptionState.status !== "ready"
    || !billingCapability?.offers.lifetime
    || ongoingRecurringSubscription;
  const previewMatchesSelection = previewState.status === "ready"
    && previewState.data.target_plan === plan
    && previewState.data.target_interval === interval;
  const creditsDisabled = baseActionDisabled
    || intentUnavailable
    || !billingCapability?.offers.credits
    || parsedCreditsQuantity === null;
  const billingCapabilityFailure: RequestFailure | null = capabilitiesState.status === "error"
    ? capabilitiesState.failure
    : capabilitiesState.status === "ready" && !capabilitiesState.data.billing.enabled
      ? {
          kind: "configuration",
          title: t({ en: "Subscription billing needs configuration", zh: "订阅与账单尚未配置完成" }),
          message: t({
            en: `${capabilitiesState.data.billing.provider || "Stripe"} is selected, but no purchasable offer is available yet.`,
            zh: `当前已选择 ${capabilitiesState.data.billing.provider || "Stripe"}，但尚未配置可购买的套餐。`
          }),
          retryable: false
        }
      : subscriptionState.status === "error"
          && (subscriptionState.failure.kind === "disabled" || subscriptionState.failure.kind === "configuration")
        ? subscriptionState.failure
        : null;
  const sessionToken = session?.token || "";

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
    if (value === "active") return t({ en: "Active", zh: "生效中" });
    if (value === "trialing") return t({ en: "Trial", zh: "试用中" });
    if (value === "canceling") return t({ en: "Ending", zh: "即将结束" });
    if (value === "past_due") return t({ en: "Past due", zh: "待付款" });
    if (value === "canceled") return t({ en: "Canceled", zh: "已取消" });
    if (value === "incomplete" || value === "payment_failed") return t({ en: "Payment needed", zh: "需要付款" });
    if (value === "expired" || value === "ended" || value === "incomplete_expired") return t({ en: "Ended", zh: "已结束" });
    return value || t({ en: "Not active", zh: "未开通" });
  }

  const loadSubscriptionState = useCallback(async (token: string) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(subscriptionGenerationRef);
    setSubscriptionState((current) => loadingResource(current.data));
    const nextState = await settleResource(getSubscription(token), "Subscription billing");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(subscriptionGenerationRef, generation)
    ) {
      setSubscriptionState(nextState);
    }
  }, []);

  const loadInvoiceState = useCallback(async (token: string) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(invoiceGenerationRef);
    setInvoiceState((current) => loadingResource(current.data));
    const nextState = await settleResource(
      listInvoices(token).then((data) => data.items),
      "Billing history"
    );
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(invoiceGenerationRef, generation)
    ) {
      setInvoiceState(nextState);
    }
  }, []);

  const loadBillingState = useCallback(async (token: string) => {
    await Promise.all([
      loadSubscriptionState(token),
      loadInvoiceState(token)
    ]);
  }, [loadInvoiceState, loadSubscriptionState]);

  const loadPreviewState = useCallback(async (
    token: string,
    targetPlan: string,
    targetInterval: string
  ) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(previewGenerationRef);
    setPreviewState(loadingResource());
    const nextState = await settleResource(
      previewSubscriptionChange(token, { plan: targetPlan, interval: targetInterval }),
      "Subscription change preview"
    );
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(previewGenerationRef, generation)
    ) {
      setPreviewState(nextState);
    }
  }, []);

  useEffect(() => {
    const syncSession = () => {
      setSession(readSession());
      setSessionReady(true);
    };
    const syncReferral = () => setReferralCode(readReferralCode());
    syncSession();
    syncReferral();
    window.addEventListener("storage", syncSession);
    window.addEventListener(SESSION_EVENT, syncSession);
    window.addEventListener(REFERRAL_EVENT, syncReferral);
    return () => {
      window.removeEventListener("storage", syncSession);
      window.removeEventListener(SESSION_EVENT, syncSession);
      window.removeEventListener(REFERRAL_EVENT, syncReferral);
    };
  }, []);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (["#history", "#invoices"].includes(window.location.hash)) setHistoryOpen(true);
    if (["#plans", "#subscription-checkout"].includes(window.location.hash)) setPlansOpen(true);
    if (["#credits", "#credits-checkout"].includes(window.location.hash)) setCreditsOpen(true);
    if (["product", "plan", "interval"].some(key => params.has(key))) {
      const intent = readBillingIntent(window.location.search);
      setIntentRequest({ intent });
      setPlansOpen(initialBillingSection(intent) === "plans");
      setCreditsOpen(initialBillingSection(intent) === "credits");
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
      invalidateRequestGeneration(previewGenerationRef);
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
      invalidateRequestGeneration(invoiceGenerationRef);
      invalidateRequestGeneration(previewGenerationRef);
      setSubscriptionState(idleResource());
      setInvoiceState(idleResource());
      setPreviewState(idleResource());
      return;
    }
    void loadBillingState(sessionToken);
    return () => {
      invalidateRequestGeneration(subscriptionGenerationRef);
      invalidateRequestGeneration(invoiceGenerationRef);
      invalidateRequestGeneration(actionGenerationRef);
    };
  }, [sessionToken, capabilitiesState, loadBillingState]);

  useEffect(() => {
    if (!sessionToken || subscriptionState.status !== "ready" || initialDisclosureTokenRef.current === sessionToken) return;
    initialDisclosureTokenRef.current = sessionToken;
    if (intentRequest !== null && !intentRequest.intent) return;
    const section = initialBillingSection(intentRequest?.intent || null, subscriptionState.data);
    if (section === "plans") setPlansOpen(true);
    if (section === "credits") setCreditsOpen(true);
  }, [intentRequest, sessionToken, subscriptionState]);

  useEffect(() => {
    if (!status && !actionFailure && busy === "") return;
    if (feedbackTarget === "plans") setPlansOpen(true);
    if (feedbackTarget === "credits") setCreditsOpen(true);
  }, [status, actionFailure, busy, feedbackTarget]);

  useEffect(() => {
    invalidateRequestGeneration(checkoutRefreshGenerationRef);
    if (checkoutRefreshTimerRef.current !== null) {
      window.clearTimeout(checkoutRefreshTimerRef.current);
      checkoutRefreshTimerRef.current = null;
    }
    setCheckoutRefreshAttempt(0);
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
        setCheckoutRefreshAttempt(nextAttempt + 1);
        await loadBillingState(sessionToken);
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

  useEffect(() => {
    if (
      !sessionToken
      || !billingReady
      || !ongoingRecurringSubscription
      || selectedLifetime
      || !availablePlans.includes(plan)
      || !availableIntervals.includes(interval)
    ) {
      invalidateRequestGeneration(previewGenerationRef);
      setPreviewState(idleResource());
      return;
    }

    void loadPreviewState(sessionToken, plan, interval);

    return () => {
      invalidateRequestGeneration(previewGenerationRef);
    };
  }, [sessionToken, billingReady, ongoingRecurringSubscription, plan, interval, selectedLifetime, availablePlans, availableIntervals, loadPreviewState]);

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
      setActionFailure({ kind: "unknown", title: t({ en: "Resolve the pending change first", zh: "请先处理已有变更" }), message: t({ en: "Complete the pending payment, remove the scheduled change, or reactivate your subscription before trying again.", zh: "请先完成待付款、撤销预约变更，或恢复续费后再操作。" }), retryable: false });
      return;
    }
    setActionFailure(describeRequestFailure(error, capability));
  }

  function canCommitAction(token: string, generation: number): boolean {
    return readSession()?.token === token
      && isCurrentRequestGeneration(actionGenerationRef, generation);
  }

  async function handleRefreshBillingState() {
    setFeedbackTarget("current");
    setStatus("");
    if (!session || !billingReady) {
      setActionFailure(session ? {
        kind: "configuration",
        title: "Billing is unavailable",
        message: "The backend billing capability is not ready.",
        retryable: true
      } : signedOutFailure);
      return;
    }

    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("refresh");
    setActionFailure(null);
    try {
      await loadBillingState(requestToken);
      if (canCommitAction(requestToken, actionGeneration)) {
        setStatus(t({ en: "Billing is up to date.", zh: "账单信息已刷新。" }));
      }
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleSubscriptionCheckout() {
    setFeedbackTarget("plans");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("subscription");
    setActionFailure(null);
    setStatus("");

    try {
      const res = await createCheckoutSession(requestToken, {
        product_type: "subscription",
        plan,
        interval,
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(t({ en: "Opening secure checkout…", zh: "正在打开安全结账页面…" }));
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription checkout");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleChangeSubscription() {
    setFeedbackTarget("plans");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("change");
    setActionFailure(null);
    setStatus("");

    try {
      const res = await changeSubscription(requestToken, {
        plan,
        interval
      });
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(res.pending_payment
        ? t({en:"Payment is required before the plan change takes effect.",zh:"套餐尚未变更，请先完成付款。"})
        : res.scheduled_change
          ? t({en:"Plan change scheduled for the next period.",zh:"已预约下周期变更套餐。"})
          : t({en:"Plan updated.",zh:"套餐已更新。"}));
      await loadBillingState(requestToken);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Plan change");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  function changeModeLabel(mode?: SubscriptionChangeMode): string {
    switch (mode) {
      case "immediate_reset_cycle":
        return t({ en: "Immediate switch, restart billing cycle", zh: "立即切换并重新开始计费周期" });
      case "period_end":
        return t({ en: "Takes effect next billing cycle", zh: "下个计费周期生效" });
      case "immediate_prorated":
        return t({ en: "Immediate switch with proration", zh: "立即切换并按比例结算" });
      default:
        return "-";
    }
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

  async function handleCreditsCheckout() {
    setFeedbackTarget("credits");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    const quantity = parseCreditsQuantity(creditsQuantity);
    if (quantity === null) {
      setActionFailure({
        kind: "unknown",
        title: "Invalid package quantity",
        message: "Enter a whole number from 1 through 100.",
        retryable: false
      });
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("credits");
    setActionFailure(null);
    setStatus("");

    try {
      const res = await createCheckoutSession(requestToken, {
        product_type: "credits",
        quantity,
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(t({ en: "Opening secure checkout…", zh: "正在打开安全结账页面…" }));
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Credits checkout");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleLifetimeCheckout() {
    setFeedbackTarget("plans");
    if (!session) {
      setActionFailure(signedOutFailure);
      return;
    }
    if (ongoingRecurringSubscription) {
      setActionFailure({
        kind: "unknown",
        title: "End the recurring subscription first",
        message: "Cancel the recurring subscription in the Billing Portal and wait until it has ended before buying Lifetime. This prevents overlapping recurring charges.",
        retryable: false
      });
      return;
    }
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("subscription");
    setActionFailure(null);
    setStatus("");

    try {
      const res = await createCheckoutSession(requestToken, {
        product_type: "lifetime",
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(t({ en: "Opening secure checkout…", zh: "正在打开安全结账页面…" }));
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Lifetime checkout");
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
      await loadBillingState(requestToken);
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
      await loadBillingState(requestToken);
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
      {actionFailure ? <ResourceFailure failure={actionFailure} /> : null}
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

  return (
    <SiteShell
      eyebrow={t({ en: "Account", zh: "账户" })}
      title={t({ en: "Subscription & billing", zh: "订阅与账单" })}
      description={t({ en: "Your access, payments, and next renewal in one place.", zh: "查看当前权益、管理续费与付款。" })}
      accountMenuData={{ capabilities: capabilitiesState, subscription: subscriptionState }}
      toc={[
        { id: "subscription-state", label: t({ en: "Overview", zh: "当前订阅" }) },
        { id: "plans", label: t({ en: "Plans", zh: "套餐选择" }) },
        { id: "credits", label: t({ en: "Credits", zh: "购买积分" }) },
        { id: "history", label: t({ en: "History", zh: "购买与账单" }) }
      ]}
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
          <div className="span-12"><ResourceFailure failure={billingCapabilityFailure} onRetry={capabilitiesState.status === "error" ? () => void loadCapabilitiesState() : undefined} /></div>
        ) : null}
        {intentUnavailable ? (
          <div className="span-12" role="status">
            <Notice>
              <strong>{t({ en: "This selection is unavailable", zh: "原选择暂不可购买" })}</strong>
              <p>{t({ en: "The linked product or billing interval is invalid or no longer offered. Review the available options and choose again before continuing.", zh: "链接中的商品或计费周期无效，或当前尚未开放。请查看可用方案并重新选择后继续。" })}</p>
              <button className="button" disabled={availablePlans.length === 0} onClick={() => changeSelection(availablePlans[0])}>
                {t({ en: "Choose another option", zh: "重新选择方案" })}
              </button>
            </Notice>
          </div>
        ) : null}
        {checkoutReturn === "success" ? (
          <div className="span-12" role="status">
            <Notice tone={checkoutRefreshComplete ? "success" : "default"}>
              {checkoutRefreshActive
                ? t({ en: "Checkout completed. We are waiting for payment confirmation and refreshing your billing.", zh: "结账已完成，正在等待付款确认并刷新账单。" })
                : checkoutRefreshComplete
                  ? t({ en: "Billing has been refreshed. If your payment is still processing, refresh again in a moment.", zh: "账单已刷新。如付款仍在处理中，请稍后再次刷新。" })
                  : session
                    ? t({ en: "Checkout completed. Your billing will refresh shortly.", zh: "结账已完成，账单状态即将自动刷新。" })
                    : t({ en: "Checkout completed. Sign in to refresh billing for this account.", zh: "结账已完成，请登录后刷新此账户的账单状态。" })}
            </Notice>
          </div>
        ) : checkoutReturn === "cancelled" ? (
          <div className="span-12"><Notice>{t({ en: "Checkout was canceled. No new purchase was confirmed.", zh: "结账已取消，没有确认新的购买。" })}</Notice></div>
        ) : null}

        <Panel className={`span-12 ${styles.overview}`} title={t({ en: "Current subscription", zh: "当前订阅" })}
          subtitle={t({ en: "Your access and next billing date.", zh: "查看你的权益与下一次计费时间。" })}
          actions={<button className="button" type="button" disabled={!session || !billingReady || busy !== "" || checkoutRefreshActive} onClick={() => void handleRefreshBillingState()}>
            {busy === "refresh" ? t({ en: "Refreshing…", zh: "刷新中…" }) : t({ en: "Refresh", zh: "刷新" })}
          </button>}
        >
          <div id="subscription-state" className={styles.anchor} />
          {subscription ? (
            <>
              <div className={styles.summary}>
                <div className={styles.planIdentity}>
                  <span className={styles.label}>{t({ en: "Your plan", zh: "当前套餐" })}</span>
                  <div className={styles.planHeading}><strong>{planName(subscription.plan)}</strong><span className={styles.status}>{statusName(subscription.status)}</span></div>
                  <p>{hasLifetime
                    ? t({ en: "One-time purchase · no recurring renewal", zh: "一次性购买 · 无周期续费" })
                    : ongoingRecurringSubscription ? intervalName(subscription.billing_cycle)
                      : t({ en: "Choose a plan when you are ready.", zh: "按需选择适合你的套餐。" })}</p>
                </div>
                <div className={styles.summaryDetails}>
                  <DetailRows rows={[
                    { label: cancellationPending ? t({ en: "Access until", zh: "权益保留至" }) : t({ en: "Current period ends", zh: "本期结束时间" }), value: hasLifetime ? t({ en: "Lifetime", zh: "永久有效" }) : formatDate(subscription.effective_cancel_at || subscription.current_period_end, dateLocale) },
                    { label: t({ en: "Renewal", zh: "续费状态" }), value: hasLifetime ? t({ en: "Not required", zh: "无需续费" }) : cancellationPending ? t({ en: "Canceled", zh: "已取消续费" }) : ongoingRecurringSubscription ? t({ en: "Automatic", zh: "自动续费" }) : t({ en: "No active subscription", zh: "暂无生效订阅" }) },
                    { label: t({ en: "Payment method", zh: "支付方式" }), value: subscription.payment_method ? `${subscription.payment_method.brand} •••• ${subscription.payment_method.last4}` : t({ en: "Not saved", zh: "尚未保存" }) }
                  ]} />
                </div>
              </div>
              {subscription.pending_payment ? (
                <div className={styles.stateCard} role="status">
                  <strong>{t({ en: "Complete payment to change your plan", zh: "完成付款后变更套餐" })}</strong>
                  <p>{t({ en: "Your current plan remains active until payment is confirmed.", zh: "付款确认前，当前套餐仍然有效。" })}</p>
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
                        await loadBillingState(token);
                      }
                    } catch (err) { if (canCommitAction(token, generation)) handleActionError(err, "Cancel scheduled change"); }
                    finally { if (canCommitAction(token, generation)) setBusy(""); }
                  }}>{t({ en: "Remove scheduled change", zh: "撤销预约变更" })}</button>
                </div>
              ) : null}
              {cancellationPending ? <p className={styles.stateNote}>{t({ en: "Renewal is canceled. You can keep using your plan through the access end date above.", zh: "已取消自动续费，上方权益截止日期前仍可继续使用。" })}</p> : null}
              <div className={`button-row ${styles.manageActions}`}>
                {ongoingRecurringSubscription || hasLifetime ? <button className="button" disabled={baseActionDisabled} onClick={() => void handleOpenPortal()}>
                  {busy === "portal" ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Manage payment & invoices", zh: "管理支付方式与发票" })}
                </button> : null}
                {!hasLifetime ? ongoingRecurringSubscription ? <a className="button" href="#plans" onClick={() => setPlansOpen(true)}>{t({ en: "Change plan", zh: "更换套餐" })}</a> : <Link className="button primary" href={localizeHref("/pricing", locale)}>{t({ en: "View plans", zh: "查看套餐" })}</Link> : null}
                {ongoingRecurringSubscription && cancellationPending ? <button className="button primary" disabled={baseActionDisabled} onClick={handleReactivate}>{busy === "reactivate" ? t({ en: "Restoring…", zh: "恢复中…" }) : t({ en: "Resume renewal", zh: "恢复续费" })}</button> : null}
                {ongoingRecurringSubscription && !cancellationPending ? <button ref={cancelTriggerRef} className={`button ${styles.cancelTrigger}`} disabled={baseActionDisabled} aria-expanded={cancelConfirmOpen} aria-controls="cancel-renewal-confirmation" onClick={() => setCancelConfirmOpen(value => !value)}>{t({ en: "Cancel renewal", zh: "取消续费" })}</button> : null}
              </div>
              {cancelConfirmOpen && ongoingRecurringSubscription && !cancellationPending ? (
                <div id="cancel-renewal-confirmation" className={styles.confirmation} role="group" aria-labelledby="cancel-renewal-title">
                  <h3 id="cancel-renewal-title">{t({ en: "Cancel your next renewal?", zh: "确认取消后续续费？" })}</h3>
                  <p>{subscription.current_period_end
                    ? t({ en: `Your plan remains available until ${formatDate(subscription.current_period_end, dateLocale)}. It will not renew after that date.`, zh: `当前套餐可继续使用至 ${formatDate(subscription.current_period_end, dateLocale)}，此后不再自动续费。` })
                    : t({ en: "Your plan remains available until the end of the current paid period and will not renew. The exact end date will appear once billing is refreshed.", zh: "当前已付费周期内仍可使用，周期结束后不再续费。刷新账单后可查看具体结束日期。" })}</p>
                  <div className="button-row">
                    <button className="button" disabled={busy !== ""} onClick={() => { setCancelConfirmOpen(false); cancelTriggerRef.current?.focus(); }}>{t({ en: "Keep renewal", zh: "保留续费" })}</button>
                    <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("end_of_period")}>{busy === "cancel" ? t({ en: "Canceling…", zh: "取消中…" }) : t({ en: "Confirm cancellation", zh: "确认取消续费" })}</button>
                  </div>
                </div>
              ) : null}
            </>
          ) : subscriptionState.status === "error" ? (
            <ResourceFailure failure={subscriptionState.failure} onRetry={() => session ? void loadSubscriptionState(session.token) : undefined} />
          ) : subscriptionPlaceholder === "loading" ? <LoadingState label={t({ en: "Loading subscription…", zh: "正在加载订阅…" })} />
            : <EmptyState>{subscriptionPlaceholder === "signed_out"
              ? t({ en: "Sign in to view your current access and renewal details.", zh: "登录后查看当前权益与续费信息。" })
              : t({ en: "Subscription information is temporarily unavailable. Your sign-in is still active; billing details will appear when the service is ready.", zh: "订阅信息暂不可用。你的登录仍然有效，计费服务就绪后即可查看权益与续费信息。" })}</EmptyState>}
          {actionFeedback("current")}
        </Panel>

        {ongoingRecurringSubscription ? <details id="plans" className={`span-12 ${styles.disclosure}`} open={plansOpen} onToggle={event => setPlansOpen(event.currentTarget.open)}>
          <summary><span className={styles.disclosureTitle}>{ongoingRecurringSubscription ? t({ en: "Change plan", zh: "更换套餐" }) : t({ en: "Choose a plan", zh: "套餐选择" })}</span><span className={styles.disclosureDescription}>{t({ en: "Compare options, choose a billing interval, and review costs.", zh: "查看可用套餐、计费周期与变更费用。" })}</span></summary>
          <div className={styles.disclosureBody}>
        <Panel className={styles.purchasePanel} title={t({ en: "Review your selection", zh: "核对套餐选择" })}
          subtitle={t({ en: "Review your selection before confirming any purchase or change.", zh: "购买或变更前，先核对你的选择。" })}
        >
          <div id="subscription-checkout" />
          <div className="field-grid">
            <div className="field">
              <label htmlFor="plan">{t({ en: "Plan", zh: "套餐" })}</label>
              <select id="plan" value={availablePlans.includes(plan) ? plan : ""} disabled={!billingReady || availablePlans.length === 0 || busy !== ""} onChange={event => changeSelection(event.target.value)}>
                {!availablePlans.includes(plan) ? <option value="">{availablePlans.length === 0 ? t({ en: "No available plans", zh: "暂无可购买套餐" }) : t({ en: "Choose an available plan", zh: "请选择可用套餐" })}</option> : null}
                {availablePlans.map(availablePlan => <option value={availablePlan} key={availablePlan}>{planName(availablePlan)}</option>)}
              </select>
            </div>
            {!selectedLifetime ? <div className="field">
              <label htmlFor="interval">{t({ en: "Billing interval", zh: "计费周期" })}</label>
              <select id="interval" value={availableIntervals.includes(interval) ? interval : ""} disabled={!billingReady || availableIntervals.length === 0 || busy !== ""} onChange={event => changeSelection(plan, event.target.value)}>
                {!availableIntervals.includes(interval) ? <option value="">{t({ en: "Choose an available interval", zh: "请选择可用周期" })}</option> : null}
                {availableIntervals.map(availableInterval => <option value={availableInterval} key={availableInterval}>{intervalName(availableInterval)}</option>)}
              </select>
            </div> : null}
          </div>
          {hasLifetime ? <Notice tone="success">{t({ en: "You already have lifetime access. No subscription is needed.", zh: "你已拥有终身权益，无需再购买订阅。" })}</Notice>
            : !selectionConfigured ? <Notice>{capabilitiesState.status === "idle" || capabilitiesState.status === "loading"
              ? t({ en: "Loading available plans…", zh: "正在加载可用套餐…" })
              : availablePlans.length === 0 ? t({ en: "No plans are available for purchase at the moment.", zh: "当前暂无可购买套餐。" })
                : t({ en: "Select an available plan and billing interval to continue.", zh: "请选择可用的套餐与计费周期后继续。" })}</Notice>
            : selectedLifetime && ongoingRecurringSubscription ? <Notice>{t({ en: "To avoid overlapping charges, cancel your recurring subscription and wait for its paid period to end before purchasing lifetime access.", zh: "为避免重叠扣款，请先取消周期订阅，并在已付费周期结束后购买终身权益。" })}</Notice>
              : selectedLifetime ? <div className={styles.orderSummary}><strong>{t({ en: "Lifetime access · one-time payment", zh: "终身权益 · 一次性付款" })}</strong><p>{t({ en: "No recurring renewal. Review the final price in secure checkout before paying.", zh: "没有周期续费。付款前请在安全结账页面确认最终价格。" })}</p></div>
                : ongoingRecurringSubscription ? (
                  previewMatchesSelection && preview ? <div className={styles.orderSummary} aria-live="polite">
                    <div className={styles.orderHeading}><span>{t({ en: "Due now", zh: "当前应付" })}</span><strong>{formatCurrencyUSD(preview.amount_due_now)}</strong></div>
                    <DetailRows rows={[
                      { label: t({ en: "Selected plan", zh: "目标套餐" }), value: `${planName(plan)} · ${intervalName(interval)}` },
                      { label: t({ en: "Change timing", zh: "变更方式" }), value: changeModeLabel(preview.change_mode) },
                      { label: t({ en: "Current period ends", zh: "本期结束" }), value: formatDate(preview.current_period_end, dateLocale) },
                      { label: t({ en: "Next billing date", zh: "下次计费" }), value: formatDate(preview.next_billing_at, dateLocale) }
                    ]} />
                  </div> : previewState.status === "error" ? <ResourceFailure failure={previewState.failure} onRetry={() => void loadPreviewState(sessionToken, plan, interval)} retryLabel={t({ en: "Retry price preview", zh: "重新获取费用预览" })} />
                    : <p className={styles.stateNote} role="status">{t({ en: "Waiting for the current price preview before you can confirm.", zh: "正在获取当前选择的费用预览，确认前请稍候。" })}</p>
                ) : <div className={styles.orderSummary}>
                  <strong>{planName(plan)} · {intervalName(interval)}</strong>
                  <p>{t({ en: "Review the final amount, renewal terms, and any trial in secure checkout before paying.", zh: "付款前请在安全结账页面核对最终金额、续费条款与适用的试用安排。" })}</p>
                </div>}
          <div className="button-row">
            {!hasLifetime && !selectionConfigured ? <button className="button primary" disabled>{t({ en: "Continue", zh: "继续" })}</button> : !hasLifetime && selectedLifetime ? (
              ongoingRecurringSubscription ? <button className="button" disabled={baseActionDisabled} onClick={() => void handleOpenPortal("plans")}>{t({ en: "Manage current subscription", zh: "管理当前订阅" })}</button>
                : <button className="button primary" disabled={lifetimePurchaseDisabled} onClick={handleLifetimeCheckout}>{busy === "subscription" ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Continue with Lifetime", zh: "继续购买终身版" })}</button>
            ) : !hasLifetime && ongoingRecurringSubscription ? (
              <button className="button primary" disabled={purchaseDisabled || !previewMatchesSelection} onClick={handleChangeSubscription}>{busy === "change" ? t({ en: "Updating…", zh: "更新中…" }) : t({ en: "Confirm plan change", zh: "确认变更套餐" })}</button>
            ) : !hasLifetime ? <button className="button primary" disabled={purchaseDisabled} onClick={handleSubscriptionCheckout}>{busy === "subscription" ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Continue to checkout", zh: "前往结账确认" })}</button> : null}
            <Link className="button" href="/pricing">{t({ en: "Compare plans", zh: "比较套餐" })}</Link>
          </div>
          <p className="footer-note">{t({ en: "Payments are handled securely by Stripe.", zh: "付款由 Stripe 安全处理。" })}{referralCode ? ` ${t({ en: "Your saved invitation applies automatically when eligible.", zh: "符合条件时，已保存的邀请码会自动应用。" })}` : ""}</p>
          {actionFeedback("plans")}
        </Panel>
          </div>
        </details> : !hasLifetime ? <Panel className="span-12" title={t({ en: "Choose your subscription", zh: "选择适合你的订阅" })} subtitle={t({ en: "Compare monthly and yearly plans on the pricing page and continue directly to secure checkout.", zh: "在价格页比较月付与年付套餐，选择后直接进入安全结账。" })}><Link className="button primary" href={localizeHref("/pricing", locale)}>{t({ en: "Explore plans", zh: "浏览套餐与价格" })}</Link></Panel> : null}

        <details id="credits" className={`span-12 ${styles.disclosure}`} open={creditsOpen} onToggle={event => {
          setCreditsOpen(event.currentTarget.open);
          if (event.currentTarget.open && !intentUnavailable) syncSelectionURL({ product: "credits" });
        }}>
          <summary><span className={styles.disclosureTitle}>{t({ en: "Add credits", zh: "购买积分" })}</span><span className={styles.disclosureDescription}>{t({ en: "Top up separately from your subscription.", zh: "按需补充使用积分，与订阅单独结算。" })}</span></summary>
          <div className={styles.disclosureBody}>
        <Panel className={styles.purchasePanel} title={t({ en: "Credit packages", zh: "积分包" })} subtitle={t({ en: "A separate one-time purchase for product usage.", zh: "按需补充产品使用积分，单独一次性购买。" })}>
          <div id="credits-checkout" />
          {billingReady && billingCapability?.offers.credits ? <>
            <div className="field">
              <label htmlFor="credits-qty">{t({ en: "Number of packages", zh: "购买份数" })}</label>
              <input id="credits-qty" type="number" min={1} max={100} step={1} inputMode="numeric" aria-describedby="credits-quantity-help" aria-invalid={parsedCreditsQuantity === null} value={creditsQuantity} disabled={busy !== ""} onChange={event => setCreditsQuantity(event.target.value)} />
              <small id="credits-quantity-help" className={parsedCreditsQuantity === null ? "field-error" : "muted"}>{t({ en: "Choose a whole number from 1 to 100.", zh: "可购买 1 至 100 份，请输入整数。" })}</small>
            </div>
            <div className={styles.orderSummary}>
              <div className={styles.orderHeading}><span>{t({ en: "Packages selected", zh: "已选份数" })}</span><strong>{parsedCreditsQuantity ?? "—"}</strong></div>
              <p>{t({ en: "The package price and credit amount are not displayed here. Check the product details and total in checkout before paying.", zh: "此页暂未展示每包价格与积分数量。付款前请在结账页面核对商品详情与总金额。" })}</p>
            </div>
            <div className="button-row"><button className="button primary" disabled={creditsDisabled} onClick={handleCreditsCheckout}>{busy === "credits" ? t({ en: "Opening…", zh: "正在打开…" }) : t({ en: "Continue to checkout", zh: "前往结账确认" })}</button></div>
          </> : <EmptyState>{t({ en: "Credit packages are not available yet.", zh: "积分包暂未开放购买。" })}</EmptyState>}
          <Link className={styles.textLink} href="/credits">{t({ en: "View credit balance & history →", zh: "查看积分余额与明细 →" })}</Link>
          {actionFeedback("credits")}
        </Panel>
          </div>
        </details>

        <details id="history" className={`span-12 ${styles.disclosure}`} open={historyOpen} onToggle={event => setHistoryOpen(event.currentTarget.open)}>
          <summary><span className={styles.disclosureTitle}>{t({ en: "Purchases & invoices", zh: "购买记录与账单" })}</span><span className={styles.disclosureDescription}>{t({ en: "Review past purchases, access records, and receipts.", zh: "查看历史购买、权益记录与收据。" })}</span></summary>
          <div className={styles.disclosureBody}>
        {session && billingReady ? <BillingHistory key={session.token} token={session.token} /> : null}
        <Panel className="span-12" title={t({ en: "Invoices", zh: "账单记录" })} subtitle={t({ en: "Payment history and downloadable receipts.", zh: "查看历史付款记录并下载收据。" })}>
          <div id="invoices" className={styles.anchor} />
          {invoices.length > 0 ? <div className={styles.tableWrap}><table className="table">
            <thead><tr><th>{t({ en: "Period", zh: "账期" })}</th><th>{t({ en: "Status", zh: "状态" })}</th><th>{t({ en: "Amount", zh: "金额" })}</th><th>{t({ en: "Issued", zh: "生成时间" })}</th><th>{t({ en: "Receipt", zh: "收据" })}</th></tr></thead>
            <tbody>{invoices.map(invoice => {
              const pdf = invoice.pdf_url || invoice.pdfurl;
              return <tr key={invoice.id}><td>{invoice.period}</td><td>{invoice.status}</td><td>{formatCurrencyUSD(invoice.amount_usd)}</td><td>{formatDate(invoice.created_at, dateLocale)}</td><td>{pdf ? <a href={pdf} target="_blank" rel="noopener noreferrer">{t({ en: "View receipt", zh: "查看收据" })}</a> : "—"}</td></tr>;
            })}</tbody>
          </table></div> : invoiceState.status === "error" ? <ResourceFailure failure={invoiceState.failure} onRetry={() => session ? void loadInvoiceState(session.token) : undefined} />
            : invoicePlaceholder === "loading" ? <LoadingState label={t({ en: "Loading invoices…", zh: "正在加载账单…" })} />
              : <EmptyState>{invoicePlaceholder === "signed_out" ? t({ en: "Sign in to view invoices for this account.", zh: "登录后查看此账户的账单。" })
                : invoicePlaceholder === "unavailable" ? t({ en: "Invoices are temporarily unavailable while billing is not ready.", zh: "计费服务暂未就绪，账单记录暂不可用。" })
                  : t({ en: "No invoices yet. Your receipts will appear here after a purchase.", zh: "暂无账单，购买后的收据将显示在这里。" })}</EmptyState>}
        </Panel>
          </div>
        </details>
      </div>
      <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} onSuccess={nextSession => { setSession(nextSession); setSessionReady(true); }} />
    </SiteShell>
  );
}
