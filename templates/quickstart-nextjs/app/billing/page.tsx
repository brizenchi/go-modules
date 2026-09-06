"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel, DetailRows } from "@/components/ui";
import { ResourceFailure, SignInRequired } from "@/components/resource-feedback";
import {
  cancelSubscription,
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
  checkoutReturnStatus,
  hasOngoingRecurringSubscription,
  parseCreditsQuantity
} from "@/lib/billing-state";
import { appEnv, appUrl } from "@/lib/env";
import { formatCurrencyUSD, formatDate } from "@/lib/format";
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
    && (Boolean(subscription?.cancel_at_period_end) || subscription?.status.trim().toLowerCase() === "canceling");
  const selectedLifetime = plan === "lifetime";
  const parsedCreditsQuantity = parseCreditsQuantity(creditsQuantity);
  const billingReady = capabilitiesState.status === "ready" && capabilitiesState.data.billing.enabled;
  const billingDisabled = (capabilitiesState.status === "ready" && !capabilitiesState.data.billing.enabled)
    || (subscriptionState.status === "error"
      && (subscriptionState.failure.kind === "disabled" || subscriptionState.failure.kind === "configuration"));
  const baseActionDisabled = !session || !billingReady || billingDisabled || busy !== "";
  const selectedOfferConfigured = selectedLifetime
    ? Boolean(billingCapability?.offers.lifetime)
    : availableIntervals.includes(interval);
  const purchaseDisabled = baseActionDisabled
    || subscriptionState.status !== "ready"
    || !availablePlans.includes(plan)
    || !selectedOfferConfigured;
  const lifetimePurchaseDisabled = baseActionDisabled
    || subscriptionState.status !== "ready"
    || !billingCapability?.offers.lifetime
    || ongoingRecurringSubscription;
  const previewMatchesSelection = previewState.status === "ready"
    && previewState.data.target_plan === plan
    && previewState.data.target_interval === interval;
  const creditsDisabled = baseActionDisabled
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
    if (value === "canceling") return t({ en: "Ending", zh: "即将结束" });
    if (value === "past_due") return t({ en: "Past due", zh: "待付款" });
    if (value === "canceled") return t({ en: "Canceled", zh: "已取消" });
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
    if (capabilitiesState.status !== "ready" || !capabilitiesState.data.billing.enabled) {
      return;
    }
    if (availablePlans.length > 0 && !availablePlans.includes(plan)) {
      setPlan(availablePlans[0]);
    }
  }, [availablePlans, capabilitiesState, plan]);

  useEffect(() => {
    if (selectedLifetime || availableIntervals.length === 0 || availableIntervals.includes(interval)) {
      return;
    }
    setInterval(availableIntervals[0]);
  }, [availableIntervals, interval, selectedLifetime]);

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
    setActionFailure(describeRequestFailure(error, capability));
  }

  function canCommitAction(token: string, generation: number): boolean {
    return readSession()?.token === token
      && isCurrentRequestGeneration(actionGenerationRef, generation);
  }

  async function handleRefreshBillingState() {
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
        setStatus("Billing state refreshed from the backend.");
      }
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleSubscriptionCheckout() {
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
      setStatus(`Checkout session created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription checkout");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleChangeSubscription() {
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
      setStatus(res.message);
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

  async function handleOpenPortal() {
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
      setStatus(`Credits checkout created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Credits checkout");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleLifetimeCheckout() {
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
      setStatus(`Lifetime checkout created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Lifetime checkout");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleCancel(cancelType: "end_of_period" | "3days") {
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
      const res = await cancelSubscription(requestToken, cancelType);
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(res.message);
      await loadBillingState(requestToken);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription cancellation");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  async function handleReactivate() {
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
      const res = await reactivateSubscription(requestToken);
      if (!canCommitAction(requestToken, actionGeneration)) return;
      setStatus(res.message);
      await loadBillingState(requestToken);
    } catch (err) {
      if (canCommitAction(requestToken, actionGeneration)) handleActionError(err, "Subscription reactivation");
    } finally {
      if (canCommitAction(requestToken, actionGeneration)) setBusy("");
    }
  }

  return (
    <SiteShell
      eyebrow={t({ en: "Billing", zh: "订阅与账单" })}
      title={t({ en: "Plans and payments, without the clutter.", zh: "套餐与支付，一目了然。" })}
      description={t({ en: "Choose a plan, manage your payment method, buy credits, and keep every invoice in one place.", zh: "在一个页面选择套餐、管理支付方式、购买积分并查看全部账单。" })}
      accountMenuData={{ capabilities: capabilitiesState, subscription: subscriptionState }}
      sideTitle={t({ en: "Current billing", zh: "当前订阅" })}
      sideBody={
        <DetailRows
          rows={[
            {
              label: t({ en: "Plan", zh: "套餐" }),
              value: <span>{planName(subscription?.plan)}</span>
            },
            {
              label: t({ en: "Status", zh: "状态" }),
              value: <span>{statusName(subscription?.status)}</span>
            },
            {
              label: t({ en: "Billing cycle", zh: "计费周期" }),
              value: <span>{intervalName(subscription?.billing_cycle)}</span>
            },
            {
              label: t({ en: "Period end", zh: "周期结束" }),
              value: <span>{formatDate(subscription?.current_period_end, dateLocale)}</span>
            }
          ]}
        />
      }
      toc={[
        { id: "subscription-checkout", label: t({ en: "Subscriptions", zh: "选择套餐" }) },
        { id: "credits-checkout", label: t({ en: "Credits", zh: "购买积分" }) },
        { id: "subscription-state", label: t({ en: "Subscription state", zh: "订阅状态" }) },
        { id: "invoices", label: t({ en: "Invoices", zh: "账单记录" }) }
      ]}
    >
      <div className="page-grid">
        {sessionReady && !session ? (
          <div className="span-12">
            <SignInRequired message={t({ en: "Sign in to view your subscription, invoices, and available checkout options.", zh: "登录后即可查看订阅、账单和可购买的套餐。" })} />
          </div>
        ) : null}

        {billingCapabilityFailure ? (
          <div className="span-12">
            <ResourceFailure
              failure={billingCapabilityFailure}
              onRetry={capabilitiesState.status === "error" ? () => void loadCapabilitiesState() : undefined}
            />
          </div>
        ) : null}

        {checkoutReturn === "success" ? (
          <div className="span-12">
            <Notice tone={checkoutRefreshComplete ? "success" : "default"}>
              {checkoutRefreshActive
                ? t({ en: `Checkout completed. Waiting for Stripe confirmation (${checkoutRefreshAttempt}/${CHECKOUT_REFRESH_DELAYS_MS.length}).`, zh: `支付已完成，正在等待 Stripe 确认并刷新账单（${checkoutRefreshAttempt}/${CHECKOUT_REFRESH_DELAYS_MS.length}）。` })
                : checkoutRefreshComplete
                  ? t({ en: "Checkout completed and billing was refreshed. If Stripe is still processing, refresh again in a moment.", zh: "支付已完成，账单状态已刷新。如 Stripe 仍在处理中，请稍后再次刷新。" })
                  : session
                    ? t({ en: "Checkout completed. Billing will refresh as soon as it is available.", zh: "支付已完成，账单状态将在可用后自动刷新。" })
                    : t({ en: "Checkout completed. Sign in to refresh billing for this account.", zh: "支付已完成，请登录后刷新此账户的账单状态。" })}
            </Notice>
          </div>
        ) : checkoutReturn === "cancelled" ? (
          <div className="span-12"><Notice>{t({ en: "Checkout was cancelled. No new purchase was confirmed.", zh: "支付已取消，没有产生新的购买。" })}</Notice></div>
        ) : null}

        <Panel className="span-7" title={t({ en: "Choose your plan", zh: "选择套餐" })} subtitle={t({ en: "Start, upgrade, or schedule a plan change with the exact cost shown before confirmation.", zh: "开通、升级或预约变更，确认前会清楚展示费用。" })}>
          <div id="subscription-checkout" />
          <div className="field-grid">
            <div className="field">
              <label htmlFor="plan">{t({ en: "Plan", zh: "套餐" })}</label>
              <select
                id="plan"
                value={availablePlans.includes(plan) ? plan : ""}
                disabled={!billingReady || availablePlans.length === 0}
                onChange={(event) => setPlan(event.target.value)}
              >
                {availablePlans.length === 0 ? <option value="">{t({ en: "No configured offers", zh: "暂无可购买套餐" })}</option> : null}
                {availablePlans.map((availablePlan) => (
                  <option value={availablePlan} key={availablePlan}>{planName(availablePlan)}</option>
                ))}
              </select>
            </div>
            {!selectedLifetime ? (
              <div className="field">
                <label htmlFor="interval">{t({ en: "Billing interval", zh: "计费周期" })}</label>
                <select
                  id="interval"
                  value={availableIntervals.includes(interval) ? interval : ""}
                  disabled={!billingReady || availableIntervals.length === 0}
                  onChange={(event) => setInterval(event.target.value)}
                >
                  {availableIntervals.length === 0 ? <option value="">{t({ en: "No configured intervals", zh: "暂无可用周期" })}</option> : null}
                  {availableIntervals.map((availableInterval) => (
                    <option value={availableInterval} key={availableInterval}>{intervalName(availableInterval)}</option>
                  ))}
                </select>
              </div>
            ) : null}
          </div>
          <Notice>
            {t({ en: "Checkout opens securely with Stripe. Existing subscriptions show a price preview before any change is confirmed.", zh: "支付将在 Stripe 安全页面完成；已有订阅在变更前会先展示价格预览。" })}
            {referralCode ? <> {t({ en: "Your saved invitation will be applied automatically when eligible.", zh: "符合条件时，已保存的邀请码会自动应用。" })}</> : null}
          </Notice>
          <div className="button-row">
            {selectedLifetime && !hasLifetime ? (
              ongoingRecurringSubscription ? (
                <div className="button-row">
                  <button className="button" disabled={baseActionDisabled} onClick={handleOpenPortal}>
                    {busy === "portal" ? t({ en: "Opening...", zh: "正在打开..." }) : t({ en: "Open Billing Portal", zh: "打开账单管理" })}
                  </button>
                </div>
              ) : (
                <button className="button primary" disabled={lifetimePurchaseDisabled} onClick={handleLifetimeCheckout}>
                  {busy === "subscription" ? t({ en: "Creating...", zh: "正在创建..." }) : t({ en: "Buy Lifetime", zh: "购买终身版" })}
                </button>
              )
            ) : ongoingRecurringSubscription ? (
              <>
                <button
                  className="button primary"
                  disabled={purchaseDisabled || !previewMatchesSelection}
                  onClick={handleChangeSubscription}
                >
                  {busy === "change" ? t({ en: "Updating...", zh: "更新中..." }) : t({ en: "Change Plan", zh: "变更套餐" })}
                </button>
                <button className="button" disabled={baseActionDisabled} onClick={handleOpenPortal}>
                  {busy === "portal" ? t({ en: "Opening...", zh: "正在打开..." }) : t({ en: "Open Billing Portal", zh: "打开账单管理" })}
                </button>
              </>
            ) : (
              <div className="button-row">
                {!hasLifetime ? (
                  <>
                    <button className="button primary" disabled={purchaseDisabled} onClick={handleSubscriptionCheckout}>
                      {busy === "subscription" ? t({ en: "Creating...", zh: "正在创建..." }) : t({ en: "Continue to checkout", zh: "前往支付" })}
                    </button>
                    {billingCapability?.offers.lifetime ? (
                      <button className="button" disabled={lifetimePurchaseDisabled} onClick={handleLifetimeCheckout}>
                        {busy === "subscription" ? t({ en: "Creating...", zh: "正在创建..." }) : t({ en: "Buy Lifetime", zh: "购买终身版" })}
                      </button>
                    ) : null}
                  </>
                ) : (
                  <Notice tone="success">{t({ en: "This account already has lifetime access.", zh: "当前账户已拥有终身权益。" })}</Notice>
                )}
              </div>
            )}
          </div>
          {selectedLifetime && ongoingRecurringSubscription ? (
            <Notice>
              {t({ en: "End the recurring subscription before buying Lifetime. Open Billing Portal, cancel it, and wait until the current paid period ends to avoid overlapping charges.", zh: "购买终身版前，请先在账单管理中取消周期订阅，并等待当前付费周期结束，以免产生重叠扣款。" })}
            </Notice>
          ) : selectedLifetime ? (
            <Notice>
              {t({ en: "Lifetime is a one-time purchase. Billing intervals and proration do not apply.", zh: "终身版为一次性购买，不涉及计费周期或按比例结算。" })}
            </Notice>
          ) : preview ? (
            <Notice>
              {t({ en: "Change timing", zh: "变更方式" })}: <span className="inline-code">{changeModeLabel(preview.change_mode)}</span>
              <br />
              {t({ en: "Amount due now", zh: "当前应付" })}: <span className="inline-code">{formatCurrencyUSD(preview.amount_due_now)}</span>
              <br />
              {t({ en: "Current period end", zh: "当前周期结束" })}: <span className="inline-code">{formatDate(preview.current_period_end, dateLocale)}</span>
              <br />
              {t({ en: "Next billing", zh: "下次计费" })}: <span className="inline-code">{formatDate(preview.next_billing_at, dateLocale)}</span>
              <br />
              {preview.message}
            </Notice>
          ) : previewState.status === "error" ? (
            <ResourceFailure
              failure={previewState.failure}
              onRetry={() => void loadPreviewState(sessionToken, plan, interval)}
              retryLabel={t({ en: "Retry preview", zh: "重新获取预览" })}
            />
          ) : null}
          <p className="footer-note">{t({ en: "Use Billing Portal for payment methods and invoice self-service. Lifetime access is a separate one-time purchase.", zh: "支付方式和发票可在账单管理中自助处理；终身版需单独一次性购买。" })}</p>
        </Panel>

        <Panel className="span-5" title={t({ en: "Add credits", zh: "购买积分" })} subtitle={t({ en: "Top up your account with one or more fixed credit packages.", zh: "按需购买固定积分包，为当前账户补充余额。" })}>
          <div id="credits-checkout" />
          {billingReady && billingCapability?.offers.credits ? (
            <>
              <div className="input-row">
                <div className="field">
                  <label htmlFor="credits-qty">{t({ en: "Package quantity (1–100)", zh: "积分包数量（1–100）" })}</label>
                  <input
                    id="credits-qty"
                    type="number"
                    min={1}
                    max={100}
                    step={1}
                    inputMode="numeric"
                    aria-invalid={parsedCreditsQuantity === null}
                    value={creditsQuantity}
                    onChange={(event) => setCreditsQuantity(event.target.value)}
                  />
                  {parsedCreditsQuantity === null ? (
                    <small className="field-error" role="alert">{t({ en: "Enter a whole number from 1 through 100.", zh: "请输入 1 到 100 之间的整数。" })}</small>
                  ) : null}
                </div>
              </div>
              <div className="button-row">
                <button className="button" disabled={creditsDisabled} onClick={handleCreditsCheckout}>
                  {busy === "credits" ? t({ en: "Creating...", zh: "正在创建..." }) : t({ en: "Buy credit packages", zh: "购买积分包" })}
                </button>
              </div>
              <p className="footer-note">{t({ en: "You will review the total securely in Stripe before payment.", zh: "付款前可在 Stripe 安全页面核对总金额。" })}</p>
            </>
          ) : (
            <EmptyState>{t({ en: "Credit packages are not available yet.", zh: "积分包暂未开放购买。" })}</EmptyState>
          )}
        </Panel>

        <Panel
          className="span-6"
          title={t({ en: "Current subscription", zh: "当前订阅" })}
          subtitle={t({ en: "Live plan status, renewal date, and saved payment method.", zh: "查看套餐状态、续费时间和已保存的支付方式。" })}
          actions={(
            <button
              className="button"
              type="button"
              disabled={!session || !billingReady || busy !== "" || checkoutRefreshActive}
              onClick={() => void handleRefreshBillingState()}
            >
              {busy === "refresh" ? t({ en: "Refreshing...", zh: "刷新中..." }) : t({ en: "Refresh billing", zh: "刷新账单" })}
            </button>
          )}
        >
          <div id="subscription-state" />
          {subscription ? (
            <div className="details-list">
              <div className="details-row">
                <strong>{t({ en: "Plan", zh: "套餐" })}</strong>
                <span>{planName(subscription.plan)}</span>
              </div>
              <div className="details-row">
                <strong>{t({ en: "Status", zh: "状态" })}</strong>
                <span>{statusName(subscription.status)}</span>
              </div>
              <div className="details-row">
                <strong>{t({ en: "Billing cycle", zh: "计费周期" })}</strong>
                <span>{intervalName(subscription.billing_cycle)}</span>
              </div>
              <div className="details-row">
                <strong>{t({ en: "Current period end", zh: "当前周期结束" })}</strong>
                <span>{formatDate(subscription.current_period_end, dateLocale)}</span>
              </div>
              <div className="details-row">
                <strong>{t({ en: "Ends after this period", zh: "周期结束后终止" })}</strong>
                <span>{subscription.cancel_at_period_end ? t({ en: "Yes", zh: "是" }) : t({ en: "No", zh: "否" })}</span>
              </div>
              <div className="details-row">
                <strong>{t({ en: "Payment method", zh: "支付方式" })}</strong>
                <span>
                  {subscription.payment_method
                    ? `${subscription.payment_method.brand} •••• ${subscription.payment_method.last4}`
                    : "-"}
                </span>
              </div>
            </div>
          ) : subscriptionState.status === "error" ? (
            <ResourceFailure
              failure={subscriptionState.failure}
              onRetry={() => session ? void loadSubscriptionState(session.token) : undefined}
            />
          ) : (
            <EmptyState>
              {subscriptionState.status === "loading"
                ? t({ en: "Loading subscription...", zh: "正在加载订阅..." })
                : session
                  ? t({ en: "No subscription data loaded yet.", zh: "暂未加载到订阅信息。" })
                  : t({ en: "Sign in to load billing data.", zh: "登录后即可加载账单信息。" })}
            </EmptyState>
          )}
          {ongoingRecurringSubscription ? (
            <div className="button-row">
              {cancellationPending ? (
                <button className="button" disabled={baseActionDisabled} onClick={handleReactivate}>
                  {t({ en: "Reactivate", zh: "恢复订阅" })}
                </button>
              ) : (
                <>
                  <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("end_of_period")}>
                    {t({ en: "Cancel at period end", zh: "周期结束时取消" })}
                  </button>
                  <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("3days")}>
                    {t({ en: "Cancel in 3 days", zh: "3 天后取消" })}
                  </button>
                </>
              )}
            </div>
          ) : hasLifetime ? (
            <p className="footer-note">{t({ en: "Lifetime access has no recurring cancellation or reactivation flow.", zh: "终身权益没有周期取消或恢复流程。" })}</p>
          ) : null}
        </Panel>

        <Panel className="span-6" title={t({ en: "Invoices", zh: "账单记录" })} subtitle={t({ en: "Your payment history and downloadable receipts.", zh: "查看历史付款记录并下载收据。" })}>
          <div id="invoices" />
          {invoices.length > 0 ? (
            <table className="table">
              <thead>
                <tr>
                  <th>{t({ en: "Period", zh: "账期" })}</th>
                  <th>{t({ en: "Status", zh: "状态" })}</th>
                  <th>{t({ en: "Amount", zh: "金额" })}</th>
                  <th>{t({ en: "Issued", zh: "生成时间" })}</th>
                  <th>{t({ en: "Receipt", zh: "收据" })}</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map((invoice) => {
                  const pdf = invoice.pdf_url || invoice.pdfurl;
                  return (
                    <tr key={invoice.id}>
                      <td>{invoice.period}</td>
                      <td>{invoice.status}</td>
                      <td>{formatCurrencyUSD(invoice.amount_usd)}</td>
                      <td>{formatDate(invoice.created_at, dateLocale)}</td>
                      <td>
                        {pdf ? (
                          <a href={pdf} target="_blank" rel="noreferrer">
                            {t({ en: "Open", zh: "查看" })}
                          </a>
                        ) : (
                          "-"
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : invoiceState.status === "error" ? (
            <ResourceFailure
              failure={invoiceState.failure}
              onRetry={() => session ? void loadInvoiceState(session.token) : undefined}
            />
          ) : (
            <EmptyState>{invoiceState.status === "loading" ? t({ en: "Loading invoices...", zh: "正在加载账单..." }) : t({ en: "No invoices yet.", zh: "暂无账单记录。" })}</EmptyState>
          )}
        </Panel>

        {status ? <div className="span-12"><Notice tone="success">{status}</Notice></div> : null}
        {actionFailure ? (
          <div className="span-12"><ResourceFailure failure={actionFailure} /></div>
        ) : null}
      </div>
    </SiteShell>
  );
}
