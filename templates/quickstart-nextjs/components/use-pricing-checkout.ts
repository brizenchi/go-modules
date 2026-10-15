"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { createCheckoutSession, getCapabilities, getSubscription } from "@/lib/api";
import { readReferralCode, readSession, type AuthSession } from "@/lib/auth";
import { checkoutReturnStatus, type CheckoutReturnStatus } from "@/lib/billing-state";
import { type BillingIntent } from "@/lib/billing-intent";
import { appUrl } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import { localizeHref } from "@/lib/locale";
import { preparePricingCheckout, pricingCheckoutFailure, pricingPurchaseHref, readPricingPurchase } from "@/lib/pricing-checkout";

const RESUME_KEY = "go-modules.pricing-purchase";
export function usePricingCheckout() {
  const { locale } = useI18n();
  const [pending, setPending] = useState<{ intent: BillingIntent; quantity: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [lastIntent, setLastIntent] = useState<BillingIntent | null>(null);
  const [returnStatus, setReturnStatus] = useState<CheckoutReturnStatus>(null);
  const [failure, setFailure] = useState<ReturnType<typeof pricingCheckoutFailure> | null>(null);
  const [restored, setRestored] = useState<ReturnType<typeof readPricingPurchase>>(null);
  const mounted = useRef(false);
  const inFlight = useRef(false);
  const route = localizeHref("/pricing", locale);

  const run = useCallback(async function run(intent: BillingIntent, quantity: number, session: AuthSession) {
    if (inFlight.current) return;
    inFlight.current = true; setBusy(true); setLastIntent(intent); setFailure(null);
    const current = () => mounted.current && readSession()?.token === session.token;
    try {
      const result = await preparePricingCheckout(intent, quantity, {
        success: appUrl(`${route}?checkout=success`), cancel: appUrl(`${route}?checkout=cancelled`)
      }, readReferralCode(), {
        capabilities: getCapabilities,
        subscription: () => getSubscription(session.token),
        create: payload => createCheckoutSession(session.token, payload),
        current
      });
      if (!current()) return;
      if (result.kind === "checkout") window.location.assign(result.url);
      else if (result.kind === "manage") setFailure({ messages: { en: "You already have a subscription. Manage or change your plan from your billing page.", zh: "你已有订阅，请在订阅管理中变更套餐或查看待付款状态。" }, requestId: "", manage: true });
      else if (result.kind === "unavailable") setFailure({ messages: { en: "This offer is not currently available for purchase.", zh: "此商品暂未开放购买，请稍后再试。" }, requestId: "", manage: false });
    } catch (error) { if (current()) setFailure(pricingCheckoutFailure(error)); }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }, [route]);

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
  return { purchase, close, signedIn, busy, lastIntent, failure, restored, pending, returnStatus, returnTo: pending ? pricingPurchaseHref(pending.intent, locale, pending.quantity) : route };
}
