import type { SubscriptionPreview, SubscriptionView } from "./api";
import { hasOngoingRecurringSubscription } from "./billing-state";
import { canConfirmSubscriptionChange, canKeepCurrentPlan, subscriptionChangeBlocker, type SubscriptionSelection } from "./subscription-change";

export function isScheduledSelection(selection: SubscriptionSelection, subscription: SubscriptionView): boolean {
  return subscriptionChangeBlocker(subscription) === "scheduled"
    && subscription.scheduled_change?.plan === selection.plan
    && subscription.scheduled_change.billing_cycle === selection.interval;
}

function subscriptionIdentity(subscription: SubscriptionView, includeSchedule = true): string {
  const schedule = subscription.scheduled_change;
  return JSON.stringify([subscription.plan, subscription.status, subscription.billing_cycle, subscription.current_period_end,
    Boolean(subscription.cancel_at_period_end), subscription.effective_cancel_at,
    includeSchedule && schedule ? [schedule.plan, schedule.billing_cycle, schedule.effective_at] : null]);
}

function priceIdentity(preview: SubscriptionPreview): string {
  return JSON.stringify([preview.target_plan, preview.target_interval, preview.currency.toLowerCase(), preview.amount_due_now, preview.unused_credit_amount,
    preview.change_mode, preview.immediate_charge, preview.effective_at_period_end, preview.current_period_end, preview.next_billing_at]);
}

/** One reviewed confirmation may replace its known schedule; changed billing details need a new review. */
export async function confirmPlanSelection<T>(reviewed: {
  selection: SubscriptionSelection;
  subscription: SubscriptionView;
  preview: SubscriptionPreview;
}, deps: ScheduleDependencies & {
  preview: () => Promise<SubscriptionPreview>;
  submit: () => Promise<T>;
}): Promise<{ state: "submitted"; result: T } | { state: "review"; subscription: SubscriptionView } | null> {
  let latest = await deps.load();
  if (!deps.current()) return null;
  const needsReview = () => ({ state: "review" as const, subscription: latest });
  if (subscriptionIdentity(latest) !== subscriptionIdentity(reviewed.subscription)
    || latest.pending_payment || isScheduledSelection(reviewed.selection, latest)) return needsReview();
  const eligible = (subscription: SubscriptionView) => canConfirmSubscriptionChange({
    selection: reviewed.selection, subscription, preview: reviewed.preview, ready: true, busy: false
  });
  if (!eligible({ ...latest, scheduled_change: null })) return needsReview();
  if (latest.scheduled_change) {
    await deps.removeSchedule();
    if (!deps.current()) return null;
    latest = await deps.load();
    if (!deps.current()) return null;
    if (subscriptionIdentity(latest, false) !== subscriptionIdentity(reviewed.subscription, false)) return needsReview();
  }
  if (!eligible(latest)) return needsReview();
  const freshPrice = await deps.preview();
  if (!deps.current()) return null;
  if (!canConfirmSubscriptionChange({ selection: reviewed.selection, subscription: latest, preview: freshPrice, ready: true, busy: false })
    || priceIdentity(freshPrice) !== priceIdentity(reviewed.preview)) return needsReview();
  const result = await deps.submit();
  return deps.current() ? { state: "submitted", result } : null;
}

export function subscriptionRenewalAction(subscription: SubscriptionView | null): "cancel" | "renew" | null {
  if (!subscription || !hasOngoingRecurringSubscription(subscription)) return null;
  return subscription.cancel_at_period_end || subscription.effective_cancel_at || subscription.status.trim().toLowerCase() === "canceling" ? "renew" : "cancel";
}

type ScheduleDependencies = {
  load: () => Promise<SubscriptionView>;
  removeSchedule: () => Promise<unknown>;
  current: () => boolean;
};

/** Keeping the current plan only removes the reviewed future change; it never purchases a plan. */
export async function keepCurrentSubscription(reviewed: SubscriptionView, deps: ScheduleDependencies): Promise<{ state: "kept" | "review"; subscription: SubscriptionView } | null> {
  let latest = await deps.load();
  if (!deps.current()) return null;
  const unchanged = () => subscriptionIdentity(latest, false) === subscriptionIdentity(reviewed, false)
    && subscriptionChangeBlocker({ ...latest, scheduled_change: null }) === null;
  const outcome = (state: "kept" | "review") => ({ state, subscription: latest });
  if (!canKeepCurrentPlan(reviewed) || !unchanged()) return outcome("review");
  // An exact retry may follow a lost response after the schedule was removed.
  if (!latest.scheduled_change) return outcome("kept");
  if (subscriptionIdentity(latest) !== subscriptionIdentity(reviewed)) return outcome("review");
  await deps.removeSchedule();
  if (!deps.current()) return null;
  latest = await deps.load();
  if (!deps.current()) return null;
  return outcome(unchanged() && !latest.scheduled_change ? "kept" : "review");
}

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
