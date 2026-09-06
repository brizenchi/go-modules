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
          title: "Subscription billing needs configuration",
          message: `${capabilitiesState.data.billing.provider || "Stripe"} is selected, but the backend has no complete billing configuration or purchasable offers.`,
          retryable: false
        }
      : subscriptionState.status === "error"
          && (subscriptionState.failure.kind === "disabled" || subscriptionState.failure.kind === "configuration")
        ? subscriptionState.failure
        : null;
  const sessionToken = session?.token || "";

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
        return "Immediate switch, restart billing cycle";
      case "period_end":
        return "Takes effect next billing cycle";
      case "immediate_prorated":
        return "Immediate switch with proration";
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
      eyebrow="Subscription Console"
      title="Operate reliable Stripe Checkout flows from one console."
      description="Subscriptions, lifetime access, and fixed credit packages use hosted Stripe Checkout. The backend and signed webhooks remain the only billing truth."
      accountMenuData={{ capabilities: capabilitiesState, subscription: subscriptionState }}
      sideTitle="Stripe callback split"
      sideBody={
        <DetailRows
          rows={[
            {
              label: "Frontend success URL",
              value: <span className="inline-code">{appUrl(appEnv.stripeSuccessPath)}</span>
            },
            {
              label: "Frontend cancel URL",
              value: <span className="inline-code">{appUrl(appEnv.stripeCancelPath)}</span>
            },
            {
              label: "Stripe webhook",
              value: <span className="inline-code">https://api.example.com/api/v1/stripe/webhook</span>
            },
            {
              label: "Client never verifies Stripe",
              value: <span>All billing truth comes from backend reads and webhook processing.</span>
            }
          ]}
        />
      }
      toc={[
        { id: "subscription-checkout", label: "Subscriptions" },
        { id: "credits-checkout", label: "Package" },
        { id: "subscription-state", label: "Subscription state" },
        { id: "invoices", label: "Invoices" }
      ]}
    >
      <div className="page-grid">
        {sessionReady && !session ? (
          <div className="span-12">
            <SignInRequired message="Sign in to view your subscription, invoices, and available checkout options." />
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
                ? `Checkout completed. Waiting for Stripe webhook confirmation and refreshing billing (${checkoutRefreshAttempt}/${CHECKOUT_REFRESH_DELAYS_MS.length}).`
                : checkoutRefreshComplete
                  ? "Checkout completed and billing state was refreshed. If Stripe is still processing the payment, use Refresh billing state in a moment."
                  : session
                    ? "Checkout completed. Billing state will refresh as soon as billing is available."
                    : "Checkout completed. Sign in to refresh the billing state for this account."}
            </Notice>
          </div>
        ) : checkoutReturn === "cancelled" ? (
          <div className="span-12"><Notice>Checkout was cancelled. No new purchase was confirmed.</Notice></div>
        ) : null}

        <Panel className="span-7" title="Subscription tiers" subtitle="Starter, Pro, Premium, and Lifetime. First purchase uses Checkout; existing recurring subscriptions change in-place.">
          <div id="subscription-checkout" />
          <div className="field-grid">
            <div className="field">
              <label htmlFor="plan">Plan</label>
              <select
                id="plan"
                value={availablePlans.includes(plan) ? plan : ""}
                disabled={!billingReady || availablePlans.length === 0}
                onChange={(event) => setPlan(event.target.value)}
              >
                {availablePlans.length === 0 ? <option value="">No configured offers</option> : null}
                {availablePlans.map((availablePlan) => (
                  <option value={availablePlan} key={availablePlan}>{availablePlan}</option>
                ))}
              </select>
            </div>
            {!selectedLifetime ? (
              <div className="field">
                <label htmlFor="interval">Interval</label>
                <select
                  id="interval"
                  value={availableIntervals.includes(interval) ? interval : ""}
                  disabled={!billingReady || availableIntervals.length === 0}
                  onChange={(event) => setInterval(event.target.value)}
                >
                  {availableIntervals.length === 0 ? <option value="">No configured intervals</option> : null}
                  {availableIntervals.map((availableInterval) => (
                    <option value={availableInterval} key={availableInterval}>{availableInterval}</option>
                  ))}
                </select>
              </div>
            ) : null}
          </div>
          <Notice>
            Success URL: <span className="inline-code">{appUrl(appEnv.stripeSuccessPath)}</span>
            <br />
            Cancel URL: <span className="inline-code">{appUrl(appEnv.stripeCancelPath)}</span>
            <br />
            Supported here: <span className="inline-code">starter / pro / premium / lifetime</span>
            <br />
            Optional referral metadata carried from browser: <span className="inline-code">{referralCode || "-"}</span>
          </Notice>
          <div className="button-row">
            {selectedLifetime && !hasLifetime ? (
              ongoingRecurringSubscription ? (
                <div className="button-row">
                  <button className="button" disabled={baseActionDisabled} onClick={handleOpenPortal}>
                    {busy === "portal" ? "Opening..." : "Open Billing Portal"}
                  </button>
                </div>
              ) : (
                <button className="button primary" disabled={lifetimePurchaseDisabled} onClick={handleLifetimeCheckout}>
                  {busy === "subscription" ? "Creating..." : "Buy Lifetime"}
                </button>
              )
            ) : ongoingRecurringSubscription ? (
              <>
                <button
                  className="button primary"
                  disabled={purchaseDisabled || !previewMatchesSelection}
                  onClick={handleChangeSubscription}
                >
                  {busy === "change" ? "Updating..." : "Change Plan"}
                </button>
                <button className="button" disabled={baseActionDisabled} onClick={handleOpenPortal}>
                  {busy === "portal" ? "Opening..." : "Open Billing Portal"}
                </button>
              </>
            ) : (
              <div className="button-row">
                {!hasLifetime ? (
                  <>
                    <button className="button primary" disabled={purchaseDisabled} onClick={handleSubscriptionCheckout}>
                      {busy === "subscription" ? "Creating..." : "Start Subscription Checkout"}
                    </button>
                    {billingCapability?.offers.lifetime ? (
                      <button className="button" disabled={lifetimePurchaseDisabled} onClick={handleLifetimeCheckout}>
                        {busy === "subscription" ? "Creating..." : "Buy Lifetime"}
                      </button>
                    ) : null}
                  </>
                ) : (
                  <Notice tone="success">This account already has lifetime access.</Notice>
                )}
              </div>
            )}
          </div>
          {selectedLifetime && ongoingRecurringSubscription ? (
            <Notice>
              End the recurring subscription before buying Lifetime. Open the Billing Portal, cancel it, and wait until the current paid period has ended; this prevents overlapping recurring charges.
            </Notice>
          ) : selectedLifetime ? (
            <Notice>
              Lifetime is a one-time buyout flow. Interval and subscription proration preview do not apply.
            </Notice>
          ) : preview ? (
            <Notice>
              Mode: <span className="inline-code">{changeModeLabel(preview.change_mode)}</span>
              <br />
              Amount due now: <span className="inline-code">{formatCurrencyUSD(preview.amount_due_now)}</span>
              <br />
              Current period end: <span className="inline-code">{formatDate(preview.current_period_end)}</span>
              <br />
              Next billing: <span className="inline-code">{formatDate(preview.next_billing_at)}</span>
              <br />
              {preview.message}
            </Notice>
          ) : previewState.status === "error" ? (
            <ResourceFailure
              failure={previewState.failure}
              onRetry={() => void loadPreviewState(sessionToken, plan, interval)}
              retryLabel="Retry preview"
            />
          ) : null}
          <p className="footer-note">
            Professional default: existing subscriptions change in place with proration; card updates and invoice self-service go through Stripe Billing Portal. Lifetime is a separate one-time buyout path.
          </p>
        </Panel>

        <Panel className="span-5" title="Package credits checkout" subtitle="Hosted Checkout uses the backend's configured credit package and a quantity from 1 to 100.">
          <div id="credits-checkout" />
          {billingReady && billingCapability?.offers.credits ? (
            <>
              <div className="input-row">
                <div className="field">
                  <label htmlFor="credits-qty">Package quantity (1–100)</label>
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
                    <small className="field-error" role="alert">Enter a whole number from 1 through 100.</small>
                  ) : null}
                </div>
              </div>
              <div className="button-row">
                <button className="button" disabled={creditsDisabled} onClick={handleCreditsCheckout}>
                  {busy === "credits" ? "Creating..." : "Buy Package Credits"}
                </button>
              </div>
              <p className="footer-note">The backend selects and validates its configured credit package before creating hosted Checkout.</p>
            </>
          ) : (
            <EmptyState>Credit packages are not configured on this backend.</EmptyState>
          )}
        </Panel>

        <Panel
          className="span-6"
          title="Current subscription"
          subtitle="Loaded from GET /stripe/subscription."
          actions={(
            <button
              className="button"
              type="button"
              disabled={!session || !billingReady || busy !== "" || checkoutRefreshActive}
              onClick={() => void handleRefreshBillingState()}
            >
              {busy === "refresh" ? "Refreshing..." : "Refresh billing state"}
            </button>
          )}
        >
          <div id="subscription-state" />
          {subscription ? (
            <div className="details-list">
              <div className="details-row">
                <strong>Plan</strong>
                <span>{subscription.plan}</span>
              </div>
              <div className="details-row">
                <strong>Status</strong>
                <span>{subscription.status}</span>
              </div>
              <div className="details-row">
                <strong>Billing cycle</strong>
                <span>{subscription.billing_cycle || "-"}</span>
              </div>
              <div className="details-row">
                <strong>Current period end</strong>
                <span>{formatDate(subscription.current_period_end)}</span>
              </div>
              <div className="details-row">
                <strong>Cancel at period end</strong>
                <span>{subscription.cancel_at_period_end ? "true" : "false"}</span>
              </div>
              <div className="details-row">
                <strong>Payment method</strong>
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
                ? "Loading subscription..."
                : session
                  ? "No subscription payload loaded yet."
                  : "Sign in to load billing data."}
            </EmptyState>
          )}
          {ongoingRecurringSubscription ? (
            <div className="button-row">
              {cancellationPending ? (
                <button className="button" disabled={baseActionDisabled} onClick={handleReactivate}>
                  Reactivate
                </button>
              ) : (
                <>
                  <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("end_of_period")}>
                    Cancel End Of Period
                  </button>
                  <button className="button danger" disabled={baseActionDisabled} onClick={() => void handleCancel("3days")}>
                    Cancel In 3 Days
                  </button>
                </>
              )}
            </div>
          ) : hasLifetime ? (
            <p className="footer-note">Lifetime access has no recurring cancellation or reactivation flow.</p>
          ) : null}
        </Panel>

        <Panel className="span-6" title="Invoices" subtitle="Loaded from GET /stripe/invoices.">
          <div id="invoices" />
          {invoices.length > 0 ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Period</th>
                  <th>Status</th>
                  <th>Amount</th>
                  <th>Issued</th>
                  <th>PDF</th>
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
                      <td>{formatDate(invoice.created_at)}</td>
                      <td>
                        {pdf ? (
                          <a href={pdf} target="_blank" rel="noreferrer">
                            Open
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
            <EmptyState>{invoiceState.status === "loading" ? "Loading invoices..." : "No invoices yet."}</EmptyState>
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
