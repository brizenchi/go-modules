import { adminTag, type StatusTone } from "./admin-presentation";

type Text = { en: string; zh: string };
function record(value: unknown): Record<string, unknown> | null {
  return typeof value === "object" && value !== null ? value as Record<string, unknown> : null;
}
function timestamp(value: unknown): string | null {
  return typeof value === "string" && Number.isFinite(new Date(value).getTime()) ? value : null;
}
function planText(plan: string, interval: unknown): Text {
  const name = ({ free: "Free", starter: "Starter", pro: "Pro", premium: "Premium", lifetime: "Lifetime" } as Record<string, string>)[plan] || plan;
  return interval === "monthly" ? { en: `${name} · Monthly`, zh: `${name} · 月付` }
    : interval === "yearly" ? { en: `${name} · Yearly`, zh: `${name} · 年付` } : { en: name, zh: name };
}

/** Admin tables use persisted billing facts, independently of manual membership access. */
export function adminSubscriptionPresentation(value: unknown) {
  const subscription = record(value);
  if (!subscription || typeof subscription.plan !== "string" || !subscription.plan || typeof subscription.status !== "string" || !subscription.status) return null;
  const ongoing = ["active", "trialing", "past_due", "canceling"].includes(subscription.status) && subscription.plan !== "lifetime";
  const cancelAt = timestamp(subscription.cancel_effective_at);
  const canceling = ongoing && (subscription.cancel_at_period_end === true || Boolean(cancelAt) || subscription.status === "canceling");
  const pending = ongoing && subscription.payment_pending === true;
  const schedule = ongoing && !canceling && !pending ? record(subscription.scheduled_change) : null;
  const upcoming = schedule && typeof schedule.plan === "string" && ["starter", "pro", "premium"].includes(schedule.plan)
    && (schedule.billing_cycle === "monthly" || schedule.billing_cycle === "yearly")
    ? { plan: planText(schedule.plan, schedule.billing_cycle), effectiveAt: timestamp(schedule.effective_at) } : null;
  let periodEnd = subscription.plan === "lifetime" ? null : timestamp(subscription.period_end);
  if (canceling && cancelAt && (!periodEnd || new Date(cancelAt) < new Date(periodEnd))) periodEnd = cancelAt;
  const status = pending ? { label: { en: "Payment pending", zh: "待完成付款" }, tone: "warning" as StatusTone }
    : adminTag("subscriptions", "status", canceling ? "canceling" : subscription.status) || { label: { en: subscription.status, zh: subscription.status }, tone: "neutral" as StatusTone };
  const endLabel: Text = canceling ? { en: "Access ends", zh: "权益截止" }
    : upcoming ? { en: "Current plan until", zh: "当前套餐保留至" }
      : subscription.status === "active" || subscription.status === "trialing" ? { en: "Next renewal", zh: "下次续费" }
        : subscription.status === "canceled" || subscription.status === "incomplete_expired" ? { en: "Period ended", zh: "本期已结束" }
          : { en: "Period end", zh: "本期截止" };
  return { plan: planText(subscription.plan, subscription.billing_interval), status, periodEnd, endLabel, upcoming };
}
