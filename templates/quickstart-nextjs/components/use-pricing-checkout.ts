"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createCheckoutSession, getCapabilities, getSubscription } from "@/lib/api";
import { readReferralCode, readSession, type AuthSession } from "@/lib/auth";
import { checkoutReturnStatus, type CheckoutReturnStatus } from "@/lib/billing-state";
import { type BillingIntent } from "@/lib/billing-intent";
import { appUrl } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import { localizeHref } from "@/lib/locale";
import { subscriptionRestrictionText } from "@/lib/billing-presentation";
import { subscriptionChangeBlocker } from "@/lib/subscription-change";
import { preparePricingCheckout, pricingCheckoutFailure, pricingCheckoutURLs, pricingPurchaseHref, readPricingPurchase, type PricingCheckoutResult } from "@/lib/pricing-checkout";
import { useConsoleResource, useConsoleSession } from "./console-kit";
import { usePageReturn } from "./use-page-return";

const RESUME_KEY = "go-modules.pricing-purchase";
export function usePricingCheckout() {
  const { locale } = useI18n();
  const [pending, setPending] = useState<{ intent: BillingIntent; quantity: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [lastIntent, setLastIntent] = useState<BillingIntent | null>(null);
  const [returnStatus, setReturnStatus] = useState<CheckoutReturnStatus>(null);
  const [failure, setFailure] = useState<ReturnType<typeof pricingCheckoutFailure> | null>(null);
  const [restored, setRestored] = useState<ReturnType<typeof readPricingPurchase>>(null);
  const [changeRequest, setChangeRequest] = useState<Extract<PricingCheckoutResult, { kind: "change" }> | null>(null);
  const { session, ready: sessionReady } = useConsoleSession();
  const account = useConsoleResource(session?.token || "", getSubscription);
  const revalidate = account.revalidate;
  usePageReturn(revalidate);
  useEffect(() => { setChangeRequest(null); }, [session?.token]);
  const mounted = useRef(false);
  const inFlight = useRef(false);
  const route = localizeHref("/pricing", locale);

  const run = useCallback(async function run(intent: BillingIntent, quantity: number, session: AuthSession) {
    if (inFlight.current) return;
    inFlight.current = true; setBusy(true); setLastIntent(intent); setFailure(null); setPending(null);
    const current = () => mounted.current && readSession()?.token === session.token;
    try {
      const returnURLs = pricingCheckoutURLs(intent, locale, quantity);
      const result = await preparePricingCheckout(intent, quantity, {
        success: appUrl(returnURLs.success), cancel: appUrl(returnURLs.cancel)
      }, readReferralCode(), {
        capabilities: getCapabilities,
        subscription: () => getSubscription(session.token),
        create: payload => createCheckoutSession(session.token, payload),
        current
      });
      if (!current()) return;
      if (result.kind === "checkout") window.location.assign(result.url);
      else if (result.kind === "change") setChangeRequest(result);
      else if (result.kind === "current") void revalidate();
      else if (result.kind === "manage") setFailure({ messages: result.subscription.plan === "lifetime"
        ? { en: "Lifetime access is already active on your account.", zh: "你的账户已享有终身权益。" }
        : intent.product === "lifetime" ? { en: "Your current subscription must end before you can buy Lifetime.", zh: "当前订阅结束后，可购买终身方案。" }
          : subscriptionRestrictionText(subscriptionChangeBlocker(result.subscription)) || { en: "Review your subscription in billing before continuing.", zh: "请先在订阅与账单中查看当前订阅。" }, requestId: "", manage: true });
      else if (result.kind === "unavailable") setFailure({ messages: { en: "This offer is not currently available for purchase.", zh: "此商品暂未开放购买，请稍后再试。" }, requestId: "", manage: false });
    } catch (error) { if (current()) setFailure(pricingCheckoutFailure(error)); }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }, [locale, revalidate]);

  useEffect(() => {
    mounted.current = true;
    setReturnStatus(checkoutReturnStatus(window.location.search));
    const selection = readPricingPurchase(window.location.search);
    setRestored(selection);
    let resume: string | null = null;
    try { resume = window.sessionStorage.getItem(RESUME_KEY); } catch { /* Selection remains visible when storage is disabled. */ }
    const session = readSession();
    // Only resume a purchase explicitly started in this tab before OAuth login.
    if (selection && session && resume === `${window.location.pathname}${window.location.search}${window.location.hash}`) {
      try { window.sessionStorage.removeItem(RESUME_KEY); } catch { /* Storage is optional. */ }
      void run(selection.intent, selection.quantity, session);
    }
    return () => { mounted.current = false; };
  }, [run]);

  function purchase(intent: BillingIntent, quantity = 1) {
    if (inFlight.current) return;
    const session = readSession();
    if (session) { void run(intent, quantity, session); return; }
    setFailure(null);setPending({ intent, quantity });
    const destination = pricingPurchaseHref(intent, locale, quantity);
    window.history.replaceState(null, "", destination);
    try { window.sessionStorage.setItem(RESUME_KEY, destination); } catch { /* Email login still continues in this page. */ }
  }
  function close() {
    setPending(null);
    try { window.sessionStorage.removeItem(RESUME_KEY); } catch { /* Storage is optional. */ }
  }
  function signedIn(session: AuthSession) {
    if (pending) void run(pending.intent, pending.quantity, session);
  }
  return { purchase, close, signedIn, busy, lastIntent, failure, restored, pending, returnStatus,
    subscription: account.data, accountError: account.error, accountLoading: !sessionReady || Boolean(session && !account.data && !account.error),
    revalidate, token: session?.token || "", changeRequest, setChangeRequest,
    returnTo: pending ? pricingPurchaseHref(pending.intent, locale, pending.quantity) : route };
}
