import type { SubscriptionView } from "./api";
import { hasOngoingRecurringSubscription } from "./billing-state";

export function subscriptionRenewalAction(subscription: SubscriptionView | null): "cancel" | "renew" | null {
  if (!subscription || !hasOngoingRecurringSubscription(subscription)) return null;
  return subscription.cancel_at_period_end || subscription.effective_cancel_at || subscription.status.trim().toLowerCase() === "canceling" ? "renew" : "cancel";
}

type ScheduleDependencies = {
  load: () => Promise<SubscriptionView>;
  removeSchedule: () => Promise<unknown>;
  current: () => boolean;
};

/** Removing a future change is explicit and never submits a replacement plan. */
export async function removeScheduledChangeForSelection(deps: ScheduleDependencies): Promise<SubscriptionView | null> {
  const latest = await deps.load();
  if (!deps.current()) return null;
  if (!latest.scheduled_change || latest.pending_payment) return latest;
  await deps.removeSchedule();
  if (!deps.current()) return null;
  const refreshed = await deps.load();
  return deps.current() ? refreshed : null;
}

type CancellationResult = {
  state: "canceled" | "ended" | "blocked" | "unconfirmed";
  subscription: SubscriptionView;
};

/** Stop renewal only after explicit confirmation, clearing a conflicting future plan first. */
export async function cancelRenewalForSubscription(deps: ScheduleDependencies & { cancel: () => Promise<unknown> }): Promise<CancellationResult | null> {
  let latest = await deps.load();
  if (!deps.current()) return null;
  if (subscriptionRenewalAction(latest) === "renew") return { state: "canceled", subscription: latest };
  if (!subscriptionRenewalAction(latest)) return { state: "ended", subscription: latest };
  if (latest.pending_payment) return { state: "blocked", subscription: latest };
  if (latest.scheduled_change) {
    await deps.removeSchedule();
    if (!deps.current()) return null;
    latest = await deps.load();
    if (!deps.current()) return null;
    if (latest.pending_payment || latest.scheduled_change) return { state: "blocked", subscription: latest };
    if (subscriptionRenewalAction(latest) === "renew") return { state: "canceled", subscription: latest };
    if (!subscriptionRenewalAction(latest)) return { state: "ended", subscription: latest };
  }
  await deps.cancel();
  if (!deps.current()) return null;
  latest = await deps.load();
  if (!deps.current()) return null;
  return { state: subscriptionRenewalAction(latest) === "renew" ? "canceled" : !subscriptionRenewalAction(latest) ? "ended" : "unconfirmed", subscription: latest };
}
