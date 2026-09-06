export type WorkspaceIcon = "overview" | "billing" | "referrals" | "account";

export type WorkspaceNavItem = {
  href: string;
  icon: WorkspaceIcon;
  label: { en: string; zh: string };
  description: { en: string; zh: string };
};

export const workspaceNavItems: WorkspaceNavItem[] = [
  {
    href: "/dashboard",
    icon: "overview",
    label: { en: "Overview", zh: "工作台" },
    description: { en: "Account at a glance", zh: "账户总览" }
  },
  {
    href: "/billing",
    icon: "billing",
    label: { en: "Subscription", zh: "订阅与账单" },
    description: { en: "Plans and payments", zh: "套餐与支付" }
  },
  {
    href: "/referrals",
    icon: "referrals",
    label: { en: "Referrals", zh: "推荐奖励" },
    description: { en: "Invite and earn", zh: "邀请与奖励" }
  },
  {
    href: "/account",
    icon: "account",
    label: { en: "Account", zh: "账户设置" },
    description: { en: "Profile and security", zh: "资料与安全" }
  }
];

export function activeWorkspaceHref(pathname: string): string {
  const normalized = pathname.split(/[?#]/, 1)[0] || "/";
  return workspaceNavItems.find((item) =>
    normalized === item.href || normalized.startsWith(`${item.href}/`)
  )?.href || "";
}

export function isWorkspacePath(pathname: string): boolean {
  return activeWorkspaceHref(pathname) !== "";
}
