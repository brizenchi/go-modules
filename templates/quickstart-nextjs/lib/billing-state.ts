import type { SubscriptionView } from "./api";

const ENDED_SUBSCRIPTION_STATUSES = new Set([
  "canceled",
  "inactive",
  "expired",
  "ended",
  "incomplete_expired"
]);

export const CHECKOUT_REFRESH_DELAYS_MS = [0, 1000, 2000, 4000, 8000] as const;

export type CheckoutReturnStatus = "success" | "cancelled" | null;

export function checkoutReturnStatus(search: string): CheckoutReturnStatus {
  const value = new URLSearchParams(search).get("checkout")?.trim().toLowerCase();
  return value === "success" || value === "cancelled" ? value : null;
}

export function hasOngoingRecurringSubscription(
  subscription: Pick<SubscriptionView, "plan" | "status"> | null | undefined
): boolean {
  if (!subscription) return false;

  const plan = subscription.plan.trim().toLowerCase();
  if (!plan || plan === "free" || plan === "lifetime") return false;

  const status = subscription.status.trim().toLowerCase();
  return !ENDED_SUBSCRIPTION_STATUSES.has(status);
}

export function parseCreditsQuantity(value: string): number | null {
  const normalized = value.trim();
  if (!/^\d+$/.test(normalized)) return null;

  const quantity = Number(normalized);
  return Number.isSafeInteger(quantity) && quantity >= 1 && quantity <= 100
    ? quantity
    : null;
}
