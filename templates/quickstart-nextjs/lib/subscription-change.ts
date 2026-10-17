import type { SubscriptionPreview, SubscriptionView } from "./api";
import { hasOngoingRecurringSubscription } from "./billing-state";

export type SubscriptionSelection = { plan: string; interval: string };
export type SubscriptionPlanAction = "subscribe" | "current" | "upgrade" | "downgrade" | "interval" | "manage" | "lifetime";

const ranks: Record<string, number> = { starter: 1, pro: 2, premium: 3 };

/** Explain a restriction once, without replacing every card's intended action. */
export function subscriptionChangeBlocker(current: SubscriptionView | null): "payment" | "scheduled" | "cancellation" | "unavailable" | null {
  if (!current || !hasOngoingRecurringSubscription(current)) return null;
  if (current.pending_payment || ["past_due", "incomplete", "unpaid", "payment_failed"].includes(current.status)) return "payment";
  if (current.scheduled_change) return "scheduled";
  if (current.cancel_at_period_end || current.effective_cancel_at || current.status === "canceling") return "cancellation";
  if (!["active", "trialing"].includes(current.status) || !ranks[current.plan]) return "unavailable";
  return null;
}

/** The existing preview endpoint returns major units; purchases use minor units. */
export function subscriptionPreviewAmount(preview: Pick<SubscriptionPreview, "currency" | "amount_due_now">, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "currency", currency: preview.currency.toUpperCase() }).format(preview.amount_due_now);
}

export function subscriptionPlanIntent(selection: SubscriptionSelection, current: SubscriptionView | null): SubscriptionPlanAction {
  if (current?.plan === "lifetime") return "lifetime";
  if (!hasOngoingRecurringSubscription(current)) return "subscribe";
  if (!current) return "subscribe";
  if (current.plan === selection.plan && current.billing_cycle === selection.interval) return "current";
  if (!ranks[current.plan] || !ranks[selection.plan]) return "manage";
  if (ranks[selection.plan] > ranks[current.plan]) return "upgrade";
  if (ranks[selection.plan] < ranks[current.plan]) return "downgrade";
  return "interval";
}

/** Selecting a target does not submit a change or override billing restrictions. */
export function canSelectSubscriptionPlan(selection: SubscriptionSelection, current: SubscriptionView | null, state: { busy: boolean; loading: boolean; failed: boolean }): boolean {
  return !state.busy && !state.loading && !state.failed
    && !(hasOngoingRecurringSubscription(current) && current?.plan === selection.plan)
    && !["current", "lifetime", "manage"].includes(subscriptionPlanIntent(selection, current));
}

/** Explicitly restore renewal only; a separate reviewed action changes the plan. */
export async function resumeRenewalForChange(deps: {
  load: () => Promise<SubscriptionView>;
  resume: () => Promise<unknown>;
  current: () => boolean;
}): Promise<SubscriptionView | null> {
  const latest = await deps.load();
  if (!deps.current()) return null;
  // A retry may follow a lost response after renewal was already restored.
  if (subscriptionChangeBlocker(latest) !== "cancellation") return latest;
  await deps.resume();
  if (!deps.current()) return null;
  const refreshed = await deps.load();
  return deps.current() ? refreshed : null;
}

/** Execution still requires an unrestricted subscription, even when showing a target. */
export function subscriptionPlanAction(selection: SubscriptionSelection, current: SubscriptionView | null): SubscriptionPlanAction {
  const intent = subscriptionPlanIntent(selection, current);
  return ["upgrade", "downgrade", "interval"].includes(intent) && subscriptionChangeBlocker(current) ? "manage" : intent;
}

export function isValidSubscriptionPreview(preview: SubscriptionPreview | null): preview is SubscriptionPreview {
  return preview !== null && Number.isFinite(preview.amount_due_now) && preview.amount_due_now >= 0
    && /^[a-z]{3}$/i.test(preview.currency)
    && ["period_end", "immediate_prorated", "immediate_reset_cycle"].includes(preview.change_mode)
    && (preview.change_mode === "period_end"
      ? preview.amount_due_now === 0 && !preview.immediate_charge && preview.effective_at_period_end
      : !preview.effective_at_period_end);
}

/** Never submit a different selection from the successfully reviewed price. */
export function canConfirmSubscriptionChange(state: {
  selection: SubscriptionSelection;
  subscription: SubscriptionView;
  preview: SubscriptionPreview | null;
  ready: boolean;
  busy: boolean;
}): boolean {
  const action = subscriptionPlanAction(state.selection, state.subscription);
  return state.ready && !state.busy && ["upgrade", "downgrade", "interval"].includes(action)
    && isValidSubscriptionPreview(state.preview)
    && state.preview?.target_plan === state.selection.plan
    && state.preview.target_interval === state.selection.interval;
}
