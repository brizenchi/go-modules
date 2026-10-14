import { billingIntentHref, type BillingIntent } from "./billing-intent";
import { hasOngoingRecurringSubscription } from "./billing-state";

type BillingCatalog = {
  enabled: boolean;
  offers: {
    subscriptions: Array<{ plan: string; intervals: string[] }>;
    lifetime: boolean;
    credits: boolean;
  };
};

/** An incoming choice must never silently become a different purchasable offer. */
export function isBillingIntentAvailable(intent: BillingIntent, billing?: BillingCatalog): boolean {
  if (!billing?.enabled) return false;
  if (intent.product === "subscription") {
    return billing.offers.subscriptions.some(offer => offer.plan === intent.plan && offer.intervals.includes(intent.interval));
  }
  return billing.offers[intent.product];
}

/** Update only purchase parameters; checkout-return state still belongs to its consumer. */
export function billingSelectionPath(search: string, intent: BillingIntent, billing?: BillingCatalog): string | null {
  if (!isBillingIntentAvailable(intent, billing)) return null;
  const selection = new URL(billingIntentHref(intent), "https://example.test");
  const params = new URLSearchParams(search);
  for (const key of ["product", "plan", "interval"]) params.delete(key);
  selection.searchParams.forEach((value, key) => params.set(key, value));
  return `/billing?${params.toString()}${selection.hash}`;
}

/** A purchase link takes precedence over the default invitation to choose a plan. */
export function initialBillingSection(intent: BillingIntent | null, subscription?: { plan: string; status: string } | null): "plans" | "credits" | null {
  if (intent) return intent.product === "credits" ? "credits" : "plans";
  if (!subscription || subscription.plan === "lifetime" || hasOngoingRecurringSubscription(subscription)) return null;
  return "plans";
}

/** Missing data is not evidence that an authenticated customer needs to sign in. */
export function billingPlaceholderState(state: {
  sessionReady: boolean;
  signedIn: boolean;
  capabilities: "idle" | "loading" | "ready" | "error";
  billingEnabled: boolean;
  resource: "idle" | "loading" | "ready" | "error";
}): "loading" | "signed_out" | "unavailable" | "empty" {
  if (!state.sessionReady) return "loading";
  if (!state.signedIn) return "signed_out";
  if (state.capabilities === "error" || (state.capabilities === "ready" && !state.billingEnabled)) return "unavailable";
  if (state.capabilities === "idle" || state.capabilities === "loading" || state.resource === "idle" || state.resource === "loading") return "loading";
  return state.resource === "error" ? "unavailable" : "empty";
}
