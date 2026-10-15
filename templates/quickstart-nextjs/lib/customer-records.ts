import type { CreditTransaction } from "./operations-api";

export const customerTabs = ["overview", "payments", "credits"] as const;
export type CustomerTab = typeof customerTabs[number];

type Copy = { en: string; zh: string };
export const customerTabNames: Record<CustomerTab, Copy> = {
  overview: { en: "Overview", zh: "用户概况" },
  payments: { en: "Payment records", zh: "支付记录" },
  credits: { en: "Credit changes", zh: "积分变更记录" }
};

/** Classification is presentation only; ledger source/idempotency keys stay intact. */
export function creditSourceName(row: Pick<CreditTransaction, "source" | "kind" | "actor_id">): Copy {
  if (row.kind === "refund") return { en: "Credit return", zh: "积分退回" };
  if (row.kind === "expire") return { en: "Expired credits", zh: "积分到期" };
  if (row.kind === "opening") return { en: "Opening balance", zh: "期初余额" };
  if (row.source.startsWith("admin_") || row.actor_id) return { en: "Administrator adjustment", zh: "管理员调整" };
  if (row.source.startsWith("referral")) return { en: "Referral reward", zh: "邀请奖励" };
  const sources: Record<string, Copy> = {
    signup: { en: "Signup bonus", zh: "注册赠送" },
    stripe: { en: "Credit purchase", zh: "充值购买" },
    credit_purchase: { en: "Credit purchase", zh: "充值购买" },
    subscription: { en: "Subscription allowance", zh: "订阅周期发放" },
    subscription_allowance: { en: "Subscription allowance", zh: "订阅周期发放" },
    monthly: { en: "Monthly allowance", zh: "每月发放" },
    campaign: { en: "Campaign reward", zh: "活动赠送" },
    activity_reward: { en: "Campaign reward", zh: "活动赠送" },
    promotion: { en: "Promotion reward", zh: "活动赠送" },
    note_export: { en: "Export usage", zh: "导出消费" }
  };
  return Object.prototype.hasOwnProperty.call(sources, row.source) ? sources[row.source] : { en: row.source || "Other", zh: row.source || "其他" };
}

export function signedCredits(amount: number): string {
  return `${amount > 0 ? "+" : ""}${amount.toLocaleString("en-US")}`;
}
