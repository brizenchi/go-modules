export const adminSections = ["overview", "users", "orders", "subscriptions", "referrals", "credits", "settings", "integrations", "audit"] as const;
export type AdminSection = typeof adminSections[number];
type Copy = { en: string; zh: string };

export const adminSectionNames: Record<AdminSection, Copy> = {
  overview: { en: "Overview", zh: "运营概览" },
  users: { en: "Users", zh: "用户管理" },
  orders: { en: "Payments", zh: "支付记录" },
  subscriptions: { en: "Subscriptions", zh: "订阅管理" },
  referrals: { en: "Referrals", zh: "推荐奖励" },
  credits: { en: "Credits", zh: "积分管理" },
  settings: { en: "Site settings", zh: "网站设置" },
  integrations: { en: "Integrations", zh: "服务集成" },
  audit: { en: "Audit log", zh: "审计日志" }
};

export const adminSectionDescriptions: Record<AdminSection, Copy> = {
  overview: { en: "Monitor registered users, active subscriptions and referral activity.", zh: "查看注册用户、有效订阅与推荐进展。" },
  users: { en: "Find customer accounts, roles and credit balances.", zh: "查询用户账号、角色与积分余额。" },
  orders: { en: "Review checkout and invoice payments. Amounts use the currency’s minor unit, such as cents for USD.", zh: "查看收银台与账单支付记录。金额以货币最小单位显示，例如 USD 使用美分。" },
  subscriptions: { en: "Review customer plans, subscription states and billing periods.", zh: "查看用户套餐、订阅状态与计费周期。" },
  referrals: { en: "Track invitations, activation and recorded referral rewards.", zh: "查看邀请归因、激活状态与已记录的推荐奖励。" },
  credits: { en: "Review credit transactions, grant credits or return a recorded consumption.", zh: "查询积分流水，按需发放积分或退回已记录的消费。" },
  settings: { en: "Maintain service details, support channels and example business settings.", zh: "维护服务资料、支持渠道与示例业务参数。" },
  integrations: { en: "Connect email and payment services. Saved changes require an API restart to take effect.", zh: "管理邮件与支付服务。保存配置后，需要重启后端服务使其生效。" },
  audit: { en: "Review administrator actions, affected resources and recorded outcomes.", zh: "查询管理员操作、关联资源与处理结果。" }
};

export const adminNavigationGroups: ReadonlyArray<{ id: string; label: Copy; sections: readonly AdminSection[] }> = [
  { id: "operations", label: { en: "Operations", zh: "运营管理" }, sections: ["overview", "users", "referrals"] },
  { id: "billing", label: { en: "Billing", zh: "订阅与计费" }, sections: ["subscriptions", "orders", "credits"] },
  { id: "system", label: { en: "Website & system", zh: "网站与系统" }, sections: ["settings", "integrations", "audit"] }
];

export const adminSectionIcons = {
  overview: "overview", users: "account", orders: "billing", subscriptions: "billing", referrals: "referrals",
  credits: "credits", settings: "settings", integrations: "integrations", audit: "audit"
} as const;

export function adminSectionHref(section: AdminSection): string {
  return section === "overview" ? "/admin" : `/admin/${section}`;
}

export function activeAdminSection(pathname: string): AdminSection | null {
  const path = pathname.split(/[?#]/, 1)[0].replace(/\/$/, "");
  return adminSections.find((section) => adminSectionHref(section) === path) ?? null;
}
