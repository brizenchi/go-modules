import type { AdminSection } from "./admin-navigation";

export type StatusTone = "neutral" | "success" | "warning" | "danger" | "info";
type Tag = { label: { en: string; zh: string }; tone: StatusTone };
const tag = (en: string, zh: string, tone: StatusTone = "neutral"): Tag => ({ label: { en, zh }, tone });
const subscriptionStates: Record<string, Tag> = {
  active: tag("Active", "有效", "success"), trialing: tag("Trialing", "试用中", "info"),
  canceling: tag("Canceling", "到期取消", "warning"), canceled: tag("Canceled", "已取消"),
  inactive: tag("Inactive", "未生效"), incomplete: tag("Incomplete", "待完成", "warning"),
  incomplete_expired: tag("Expired", "已过期"), past_due: tag("Past due", "付款逾期", "warning"),
  unpaid: tag("Unpaid", "未支付", "danger"), paused: tag("Paused", "已暂停", "warning")
};
const paymentStates: Record<string, Tag> = {
  succeeded: tag("Succeeded", "支付成功", "success"), paid: tag("Paid", "已支付", "success"),
  failed: tag("Failed", "支付失败", "danger"), expired: tag("Expired", "已过期"), pending: tag("Pending", "待支付", "warning"),
  processing: tag("Processing", "处理中", "info"), unpaid: tag("Unpaid", "未支付", "warning"),
  no_payment_required: tag("No payment required", "无需支付"), refunded: tag("Refunded", "已退款")
};
const referralStates: Record<string, Tag> = {
  pending: tag("Pending", "待激活", "warning"), activated: tag("Activated", "已激活", "success"), expired: tag("Expired", "已过期")
};
const auditStates: Record<string, Tag> = {
  pending: tag("Processing", "处理中", "warning"), succeeded: tag("Succeeded", "已完成", "success"), failed: tag("Failed", "失败", "danger")
};
const creditKinds: Record<string, Tag> = {
  opening: tag("Opening balance", "期初余额"), grant: tag("Grant", "发放", "success"),
  consume: tag("Consumption", "消费"), refund: tag("Return", "退回", "info"), expire: tag("Expiry", "过期")
};
const plans: Record<string, Tag> = {
  free: tag("Free", "免费版"), starter: tag("Starter", "入门版"), pro: tag("Pro", "Pro", "info"), premium: tag("Premium", "Premium", "success"), lifetime: tag("Lifetime", "Lifetime", "success")
};

/** Enum labels are scoped to their domain and field, never applied to reasons or IDs. */
export function adminTag(section: AdminSection, field: string, value: unknown): Tag | null {
  if (section === "orders" && field === "livemode") return typeof value === "boolean" ? value ? tag("Live", "正式", "info") : tag("Test", "测试") : tag("Unknown", "未知");
  if (value === null || value === undefined || value === "") return null;
  const code = String(value);
  let values: Record<string, Tag> | undefined;
  if (field === "status") values = section === "orders" ? paymentStates : section === "subscriptions" ? subscriptionStates : section === "referrals" ? referralStates : section === "audit" ? auditStates : undefined;
  else if (field === "kind" && section === "credits") values = creditKinds;
  else if (field === "plan" && (section === "subscriptions" || section === "users")) values = plans;
  else if (field === "role" && section === "users") values = { admin: tag("Administrator", "管理员", "info"), user: tag("User", "用户") };
  return values ? Object.prototype.hasOwnProperty.call(values, code) ? values[code] : tag(code, code) : null;
}

const actions: Record<string,{en:string;zh:string}> = {
 "settings.update":{en:"Update website settings",zh:"修改网站设置"},"membership.update":{en:"Adjust membership",zh:"调整会员级别"},"credits.grant":{en:"Grant credits",zh:"发放积分"},"credits.refund":{en:"Return credits",zh:"退回积分"},"credits.deduct":{en:"Deduct credits",zh:"扣减积分"},"referral.retry_reward":{en:"Reconcile referral reward",zh:"核对推荐奖励"},"content.save":{en:"Save article draft",zh:"保存文章草稿"},"content.publish":{en:"Publish article",zh:"发布文章"},"content.unpublish":{en:"Unpublish article",zh:"下架文章"},"integration.write.denied":{en:"Credential change blocked",zh:"拦截密钥修改"},"admin.request.failed":{en:"Administrative request failed",zh:"管理操作失败"}
};
export function adminActionLabel(action:string):{en:string;zh:string}{
 if(Object.prototype.hasOwnProperty.call(actions,action))return actions[action];
 const base=action.replace(/\.failed$/,"");if(base!==action&&Object.prototype.hasOwnProperty.call(actions,base)){const label=actions[base];return{en:`${label.en} failed`,zh:`${label.zh}失败`};}
 return{en:action,zh:action};
}
