export const adminSections = ["overview", "users", "articles", "orders", "subscriptions", "referrals", "credits", "settings", "integrations", "audit"] as const;
export type AdminSection = typeof adminSections[number];
type Copy = { en: string; zh: string };

export const adminSectionNames: Record<AdminSection, Copy> = {
  overview: { en: "Overview", zh: "运营概览" },
  users: { en: "Users", zh: "用户管理" },
 articles: {en:"Articles",zh:"文章管理"},
  orders: { en: "Payments", zh: "支付记录" },
  subscriptions: { en: "Subscriptions", zh: "订阅管理" },
  referrals: { en: "Referrals", zh: "推荐奖励" },
  credits: { en: "Credit history", zh: "积分流水" },
  settings: { en: "Site settings", zh: "网站设置" },
  integrations: { en: "Integrations", zh: "服务集成" },
  audit: { en: "Audit log", zh: "审计日志" }
};

export const adminSectionDescriptions: Record<AdminSection, Copy> = {
  articles: {en:"Write, translate and publish website articles.",zh:"上传、编辑并按语言发布网站文章。"},
  overview: { en: "Track daily acquisition, payments and customer conversion.", zh: "查看每日注册、邀请、支付和用户转化趋势。" },
  users: { en: "Manage customer memberships and credits without leaving the list.", zh: "查询用户，在弹窗中调整会员级别和积分。" },
  orders: { en: "Review payments, amounts and their live or test environment.", zh: "查看支付结果、实际金额及所属的正式或测试环境。" },
  subscriptions: { en: "Review customer plans, subscription states and billing periods.", zh: "查看用户套餐、订阅状态与计费周期。" },
  referrals: { en: "Track invitations, activation and recorded referral rewards.", zh: "查看邀请归因、激活状态与已记录的推荐奖励。" },
  credits: { en: "Review credit transactions, grant credits or return a recorded consumption.", zh: "查询积分流水，按需发放积分或退回已记录的消费。" },
  settings: { en: "Maintain service details, support channels and example business settings.", zh: "维护服务资料、支持渠道与示例业务参数。" },
  integrations: { en: "Connect email and payment services. Saved changes require an API restart to take effect.", zh: "管理邮件与支付服务。保存配置后，需要重启后端服务使其生效。" },
  audit: { en: "Review administrator actions, affected resources and recorded outcomes.", zh: "查询管理员操作、关联资源与处理结果。" }
};

export const adminNavigationGroups: ReadonlyArray<{ id: string; label: Copy; sections: readonly AdminSection[] }> = [
 {id:"management",label:{en:"Administration",zh:"网站管理"},sections:["overview","users","articles","settings","audit"]}
];
export const adminPrimarySection = (section: AdminSection | null): AdminSection | null => section && ["orders","subscriptions","referrals","credits"].includes(section) ? "users" : section === "integrations" ? "settings" : section;

export const adminSectionIcons = {
  articles: "notes", overview: "overview", users: "account", orders: "billing", subscriptions: "billing", referrals: "referrals",
  credits: "credits", settings: "settings", integrations: "integrations", audit: "audit"
} as const;

export function adminSectionHref(section: AdminSection): string {
  return section === "overview" ? "/admin" : `/admin/${section}`;
}

export function activeAdminSection(pathname: string): AdminSection | null {
  const path = pathname.split(/[?#]/, 1)[0].replace(/\/$/, "");
  if(/^\/admin\/articles\/[^/]+$/.test(path))return "articles";
  return adminSections.find((section) => adminSectionHref(section) === path) ?? null;
}
