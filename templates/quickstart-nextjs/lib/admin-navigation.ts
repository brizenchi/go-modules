export const adminSections = ["overview", "users", "orders", "subscriptions", "referrals", "credits", "settings", "audit"] as const;
export type AdminSection = typeof adminSections[number];

export const adminSectionNames: Record<AdminSection, { en: string; zh: string }> = {
  overview: { en: "Site overview", zh: "全站概况" },
  users: { en: "Users", zh: "用户管理" },
  orders: { en: "Payments", zh: "支付记录" },
  subscriptions: { en: "Subscriptions", zh: "订阅管理" },
  referrals: { en: "Invitations", zh: "邀请管理" },
  credits: { en: "Credit adjustments", zh: "积分管理" },
  settings: { en: "Site settings", zh: "网站配置" },
  audit: { en: "Audit log", zh: "操作记录" }
};

export function adminSectionHref(section: AdminSection): string {
  return section === "overview" ? "/admin" : `/admin/${section}`;
}

export function activeAdminSection(pathname: string): AdminSection | null {
  const path = pathname.split(/[?#]/, 1)[0].replace(/\/$/, "");
  return adminSections.find((section) => adminSectionHref(section) === path) ?? null;
}
