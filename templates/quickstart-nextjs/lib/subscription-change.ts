import type { SubscriptionPreview, SubscriptionView } from "./api";
import { hasOngoingRecurringSubscription } from "./billing-state";

export type SubscriptionSelection = { plan: string; interval: string };
export type SubscriptionPlanAction = "subscribe" | "current" | "upgrade" | "downgrade" | "interval" | "manage" | "lifetime";

const ranks: Record<string, number> = { starter: 1, pro: 2, premium: 3 };

/** The existing preview endpoint returns major units; purchases use minor units. */
export function subscriptionPreviewAmount(preview: Pick<SubscriptionPreview, "currency" | "amount_due_now">, locale: string): string {
  return new Intl.NumberFormat(locale, { style: "currency", currency: preview.currency.toUpperCase() }).format(preview.amount_due_now);
}

export function subscriptionPlanAction(selection: SubscriptionSelection, current: SubscriptionView | null): SubscriptionPlanAction {
  if (current?.plan === "lifetime") return "lifetime";
  if (!hasOngoingRecurringSubscription(current)) return "subscribe";
  if (!current) return "subscribe";
  if (current.plan === selection.plan && current.billing_cycle === selection.interval) return "current";
  if (current.pending_payment || current.scheduled_change || current.cancel_at_period_end || current.effective_cancel_at
    || !["active", "trialing"].includes(current.status) || !ranks[current.plan] || !ranks[selection.plan]) return "manage";
  if (ranks[selection.plan] > ranks[current.plan]) return "upgrade";
  if (ranks[selection.plan] < ranks[current.plan]) return "downgrade";
  return "interval";
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
    && state.preview?.target_plan === state.selection.plan
    && state.preview.target_interval === state.selection.interval;
}
