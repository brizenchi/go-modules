import { ApiError, type CapabilitiesView, type CheckoutPayload, type SubscriptionView } from "./api";
import { billingIntentHref, readBillingIntent, type BillingIntent } from "./billing-intent";
import { hasOngoingRecurringSubscription, parseCreditsQuantity, type CheckoutReturnStatus } from "./billing-state";
import { isBillingIntentAvailable } from "./billing-selection";
import { localizeHref, type Locale } from "./locale";
import { subscriptionPlanIntent, type SubscriptionSelection } from "./subscription-change";

/** Keep the return marker so billing can quietly wait for server-side fulfillment. */
export function pricingCheckoutResultHref(status: CheckoutReturnStatus, product?: string, sessionID?: string): string {
  if (product === "credits") return status === "success"
    ? `/credits?checkout=success${sessionID ? `&session_id=${encodeURIComponent(sessionID)}` : ""}` : "/credits";
  return status === "success" ? "/billing?checkout=success" : "/billing";
}

export function pricingCheckoutURLs(intent: BillingIntent, locale: Locale, quantity = 1): { success: string; cancel: string } {
  const cancel = pricingPurchaseHref(intent, locale, quantity);
  return {
    success: intent.product === "credits"
      ? `${localizeHref("/credits", locale)}?checkout=success&session_id={CHECKOUT_SESSION_ID}`
      : `${localizeHref("/billing", locale)}?checkout=success`,
    cancel: cancel.replace("?", "?checkout=cancelled&")
  };
}

export function pricingPurchaseHref(intent: BillingIntent, locale: Locale, quantity = 1): string {
  const selection = new URL(billingIntentHref(intent), "https://example.test");
  if (intent.product === "credits") selection.searchParams.set("quantity", String(quantity));
  return `${localizeHref("/pricing", locale)}?${selection.searchParams.toString()}#${intent.product === "credits" ? "one-time" : "plans"}`;
}

export function readPricingPurchase(search: string): { intent: BillingIntent; quantity: number } | null {
  const intent = readBillingIntent(search);
  if (!intent) return null;
  const params = new URLSearchParams(search);
  if (params.getAll("quantity").length > 1) return null;
  const quantity = intent.product === "credits" ? parseCreditsQuantity(params.get("quantity") || "1") : 1;
  return quantity === null ? null : { intent, quantity };
}

type CheckoutDependencies = {
  capabilities: () => Promise<CapabilitiesView>;
  subscription: () => Promise<SubscriptionView>;
  create: (payload: CheckoutPayload) => Promise<{ checkout_url: string }>;
  current: () => boolean;
};
export type PricingCheckoutResult = { kind: "checkout"; url: string }
  | { kind: "change"; selection: SubscriptionSelection; subscription: SubscriptionView; offers: CapabilitiesView["billing"]["offers"]["subscriptions"] }
  | { kind: "manage"; subscription: SubscriptionView }
  | { kind: "unavailable" | "stale" | "current" };

export async function preparePricingCheckout(intent: BillingIntent, quantity: number, urls: { success: string; cancel: string }, referralCode: string, deps: CheckoutDependencies): Promise<PricingCheckoutResult> {
  const capabilities = await deps.capabilities();
  if (!deps.current()) return { kind: "stale" };
  if (!isBillingIntentAvailable(intent, capabilities.billing)) return { kind: "unavailable" };
  if (intent.product !== "credits") {
    const subscription = await deps.subscription();
    if (!deps.current()) return { kind: "stale" };
    if (intent.product === "subscription" && hasOngoingRecurringSubscription(subscription)) {
      const selection = { plan: intent.plan, interval: intent.interval };
      const action = subscriptionPlanIntent(selection, subscription);
      if (action === "current") return { kind: "current" };
      if (["upgrade", "downgrade", "interval"].includes(action)) return { kind: "change", selection, subscription, offers: capabilities.billing.offers.subscriptions };
      return { kind: "manage", subscription };
    }
    if (subscription.plan === "lifetime") return { kind: "manage", subscription };
  }
  if (intent.product === "credits" && parseCreditsQuantity(String(quantity)) === null) return { kind: "unavailable" };
  const payload: CheckoutPayload = {
    product_type: intent.product,
    ...(intent.product === "subscription" ? { plan: intent.plan, interval: intent.interval } : {}),
    ...(intent.product === "credits" ? { quantity } : {}),
    success_url: urls.success, cancel_url: urls.cancel,
    ...(referralCode ? { metadata: { referral_code: referralCode } } : {})
  };
  const checkout = await deps.create(payload);
  if (!deps.current()) return { kind: "stale" };
  const url = new URL(checkout.checkout_url);
  if (url.protocol !== "https:" || url.hostname !== "checkout.stripe.com" || url.username || url.password) throw new Error("Invalid checkout destination");
  return { kind: "checkout", url: url.href };
}

export function pricingCheckoutFailure(error: unknown) {
  const status = error instanceof ApiError ? error.status : 0;
  const messages = status === 401 ? { en: "Please sign in again, then retry your purchase.", zh: "登录已失效，请重新登录后再购买。" }
    : status === 409 ? { en: "Your billing status has changed. Review billing before trying again.", zh: "订阅状态已变更，请先查看账单后重试。" }
    : status === 503 || status === 404 ? { en: "Payments are currently unavailable. Please try again later.", zh: "付款暂时不可用，请稍后再试。" }
    : { en: "We couldn't open checkout. Please try again.", zh: "暂时无法打开结账，请重试。" };
  return { messages, requestId: status >= 500 && error instanceof ApiError ? error.requestId : "", manage: status === 409 };
}

/** Identify the card that owns progress or feedback without unlocking other purchases. */
export function pricingIntentMatches(current: BillingIntent | null, selected: BillingIntent): boolean {
  if (!current || current.product !== selected.product) return false;
  if (current.product === "subscription" && selected.product === "subscription") {
    return current.plan === selected.plan && current.interval === selected.interval;
  }
  return true;
}
