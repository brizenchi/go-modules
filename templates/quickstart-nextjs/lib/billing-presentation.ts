import type { SubscriptionPreview, SubscriptionView } from "./api";
import { hasOngoingRecurringSubscription, subscriptionUpgradeAction } from "./billing-state";
import { subscriptionChangeBlocker, type SubscriptionPlanAction } from "./subscription-change";
import type { RequestFailure } from "./request-state";

type Text = { en: string; zh: string };
export type BillingRequestFailure = RequestFailure & { reason?: string };
export const billingText = {
  viewBilling: { en: "Go to billing", zh: "前往订阅与账单" },
  current: { en: "Current plan", zh: "当前套餐" },
  lifetime: { en: "Lifetime active", zh: "已享终身权益" },
  loading: { en: "Checking your plan…", zh: "正在确认套餐…" },
  canceledRenewal: { en: "Renewal is canceled. Your plan stays available until the access end date. Existing credits are kept under their original expiry rules.", zh: "已取消续费，权益保留至截止日期。已有积分保留，原有效期照常。" },
  checkoutChecking: { en: "Confirming your payment. Your plan and payment history will update when confirmation arrives.", zh: "正在确认付款，确认后会自动更新套餐和支付记录。" },
  checkoutWaiting: { en: "Review your plan and payment history below. If a paid purchase is missing, contact support before paying again.", zh: "请核对下方套餐与支付记录。若已付款但未到账，请联系支持，先不要重复付款。" },
  checkoutCanceled: { en: "Checkout was canceled. You can choose a plan again.", zh: "已退出结账，可以重新选择套餐。" }
} satisfies Record<string, Text>;

export function subscriptionRestrictionText(blocker: ReturnType<typeof subscriptionChangeBlocker>): Text | null {
  return blocker ? {
    payment: { en: "Complete the pending payment before changing plans.", zh: "完成待付款后，才可更改套餐。" },
    scheduled: { en: "Cancel the scheduled change before confirming another plan.", zh: "确认其他套餐前，请先撤销当前预约变更。" },
    cancellation: { en: "Renewal is canceled. Resume renewal before confirming a plan change.", zh: "已取消续费，请先恢复续费，再确认套餐变更。" },
    unavailable: { en: "Your subscription needs attention. Review it in billing before changing plans.", zh: "当前订阅需要处理，请先前往订阅与账单查看。" }
  }[blocker] : null;
}

export function isEndedSubscription(subscription: SubscriptionView | null): boolean {
  return Boolean(subscription && !["free", "lifetime", ""].includes(subscription.plan) && !hasOngoingRecurringSubscription(subscription));
}

export function lifetimePurchaseAction(subscription: SubscriptionView | null): "owned" | "buy" {
  return subscription?.plan === "lifetime" ? "owned" : "buy";
}

export function planActionText(action: SubscriptionPlanAction, name: string, _interval: string, ended = false): Text {
  switch (action) {
    case "current": return billingText.current;
    case "upgrade":
    case "downgrade":
    case "interval": return { en: "Change plan", zh: "更改套餐" };
    case "lifetime": return billingText.lifetime;
    case "manage": return billingText.viewBilling;
    default: return ended ? { en: `Resubscribe to ${name}`, zh: `重新订阅 ${name}` } : { en: `Subscribe to ${name}`, zh: `订阅 ${name}` };
  }
}

export function billingPrimaryAction(membershipPlan: string | undefined, subscription: SubscriptionView | null) {
  const action = subscriptionUpgradeAction(membershipPlan, subscription);
  if (action === "loading" || action === "lifetime") return action;
  if (action === "change") return "change";
  return isEndedSubscription(subscription) ? "resubscribe" : "upgrade";
}

export function billingPrimaryText(action: ReturnType<typeof billingPrimaryAction>): Text {
  return {
    loading: billingText.loading,
    lifetime: billingText.lifetime,
    upgrade: { en: "Upgrade", zh: "升级套餐" },
    resubscribe: { en: "Subscribe again", zh: "重新订阅" },
    change: { en: "Change plan", zh: "更改套餐" },
  }[action];
}

export function subscriptionStatusText(status: string): Text {
  return ({
    active: { en: "Active", zh: "生效中" },
    trialing: { en: "Trial", zh: "试用中" },
    canceling: { en: "Renewal canceled", zh: "已取消续费" },
    past_due: { en: "Payment overdue", zh: "付款逾期" },
    incomplete: { en: "Payment required", zh: "待付款" },
    unpaid: { en: "Payment required", zh: "待付款" },
    payment_failed: { en: "Payment failed", zh: "付款失败" },
    canceled: { en: "Ended", zh: "已结束" },
    inactive: { en: "Not subscribed", zh: "未订阅" },
    expired: { en: "Ended", zh: "已结束" },
    ended: { en: "Ended", zh: "已结束" },
    incomplete_expired: { en: "Checkout expired", zh: "付款已过期" }
  } as Record<string, Text>)[status] || { en: "Needs attention", zh: "待处理" };
}

export function changeDialogTitle(choice: SubscriptionPlanAction): Text {
  return choice === "upgrade" ? { en: "Upgrade plan", zh: "升级套餐" }
    : choice === "downgrade" ? { en: "Downgrade plan", zh: "降级套餐" }
      : choice === "interval" ? { en: "Change billing interval", zh: "更改计费周期" }
        : { en: "Change plan", zh: "更改套餐" };
}

export function changeConfirmationText(preview: SubscriptionPreview | null, amount: string): Text {
  if (!preview) return { en: "Review the price first", zh: "请先核对费用" };
  if (preview.change_mode === "period_end") return { en: "Confirm scheduled change", zh: "确认下期变更" };
  if (preview.amount_due_now > 0) return { en: `Confirm and pay ${amount}`, zh: `确认并支付 ${amount}` };
  return { en: "Confirm change", zh: "确认变更" };
}

export function changeSubmitText(state: { choice: SubscriptionPlanAction; configured: boolean; phase: "idle" | "loading" | "ready" | "error"; preview: SubscriptionPreview | null; amount: string }): Text {
  if (state.choice === "current") return billingText.current;
  if (!state.configured) return { en: "Choose an available plan", zh: "请选择可用套餐" };
  if (!["upgrade", "downgrade", "interval"].includes(state.choice)) return { en: "Review billing first", zh: "请先处理当前订阅" };
  if (state.phase === "error" || (state.phase === "ready" && !state.preview)) return { en: "Price unavailable", zh: "暂无法确认费用" };
  if (!state.preview) return { en: "Loading price…", zh: "正在获取费用…" };
  return changeConfirmationText(state.preview, state.amount);
}

export function changeTimingText(preview: SubscriptionPreview): { label: Text; detail: Text } {
  const mode = preview.change_mode;
  if (mode === "period_end") return {
    label: { en: "Next billing cycle", zh: "下个计费周期" },
    detail: { en: "Your current plan continues until the change takes effect. No payment is due now.", zh: "变更生效前继续使用当前套餐，本次无需付款。" }
  };
  if (mode === "immediate_reset_cycle") return {
    label: preview.amount_due_now > 0 ? { en: "After payment, new billing cycle", zh: "付款确认后，开启新计费周期" } : { en: "On confirmation, new billing cycle", zh: "确认后，开启新计费周期" },
    detail: { en: "A new billing cycle starts when the change takes effect. The amount shown includes the adjustment for your current plan.", zh: "变更生效后开始新的计费周期，当前应付已包含原套餐的费用调整。" }
  };
  return {
    label: preview.amount_due_now > 0 ? { en: "After payment confirmation", zh: "付款确认后生效" } : { en: "On confirmation", zh: "确认后生效" },
    detail: { en: "The amount shown includes the adjustment for the remaining time in your current billing period.", zh: "当前应付已包含本期剩余时间的费用调整。" }
  };
}

export function changePaymentText(preview: SubscriptionPreview): Text | null {
  if (preview.change_mode === "period_end" || preview.amount_due_now <= 0) return null;
  return {
    en: "Confirming authorizes Stripe to charge your saved payment method. Additional verification may be required.",
    zh: "确认后，Stripe 将从已保存的付款方式扣款；如需验证，会提示继续付款。"
  };
}

export function changeResultText(result: Pick<SubscriptionView, "pending_payment" | "scheduled_change">) {
  if (result.pending_payment) return { state: "payment" as const, title: { en: "Payment required", zh: "等待付款" }, body: { en: "Complete payment before the new plan takes effect.", zh: "完成付款后，新套餐才会生效。" } };
  if (result.scheduled_change) return { state: "scheduled" as const, title: { en: "Plan change scheduled", zh: "已预约套餐变更" }, body: { en: "Your current plan continues until the scheduled change takes effect.", zh: "预约变更生效前，继续使用当前套餐。" } };
  return { state: "updated" as const, title: { en: "Plan updated", zh: "套餐已更新" }, body: { en: "Your plan change has taken effect.", zh: "套餐变更已生效。" } };
}

/** Never echo provider messages or infrastructure instructions into billing UI. */
export function billingFailureText(kind: RequestFailure["kind"], reason = ""): { title: Text; message: Text } {
  if (reason === "subscription_change_pending") return { title: { en: "Subscription changed", zh: "订阅状态已变化" }, message: { en: "Check the latest billing status before trying another change.", zh: "请先查看最新账单状态，再进行其他变更。" } };
  if (kind === "auth") return { title: { en: "Sign in again", zh: "请重新登录" }, message: { en: "Your session has expired. Sign in to continue.", zh: "登录已失效，请重新登录后继续。" } };
  if (kind === "network") return { title: { en: "Connection interrupted", zh: "连接中断" }, message: { en: "Check your connection and try again.", zh: "请检查网络连接后重试。" } };
  return { title: { en: "Billing is temporarily unavailable", zh: "账单服务暂不可用" }, message: { en: "Try again later. Contact support if you need help with a payment.", zh: "请稍后重试；如需处理付款问题，请联系支持。" } };
}
