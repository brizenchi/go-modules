export type WorkspaceIcon = "billing" | "referrals" | "account" | "credits" | "notes" | "files";

export type WorkspaceNavItem = {
  href: string;
  icon: WorkspaceIcon;
  label: { en: string; zh: string };
  description: { en: string; zh: string };
};

export const workspaceNavItems: WorkspaceNavItem[] = [
  {
    href: "/account",
    icon: "account",
    label: { en: "Settings", zh: "账户设置" },
    description: { en: "Profile and security", zh: "资料与安全" }
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
  }
];

// Personal product pages retain the account layout but are not primary account
// navigation. Site-wide operations use the separate /admin layout.
export const auxiliaryWorkspaceItems: WorkspaceNavItem[] = [
  { href: "/credits", icon: "credits", label: { en: "Credits", zh: "积分明细" }, description: { en: "Balance and history", zh: "余额、消费与奖励" } },
  { href: "/notes", icon: "notes", label: { en: "Notes", zh: "我的笔记" }, description: { en: "Write and export", zh: "记录与积分导出" } },
  { href: "/files", icon: "files", label: { en: "Files", zh: "我的文件" }, description: { en: "Private image library", zh: "私有图片空间" } }
];

export function activeWorkspaceHref(pathname: string): string {
  const normalized = pathname.split(/[?#]/, 1)[0] || "/";
  return [...workspaceNavItems, ...auxiliaryWorkspaceItems].find((item) =>
    normalized === item.href || normalized.startsWith(`${item.href}/`)
  )?.href || "";
}

export function isWorkspacePath(pathname: string): boolean {
  return activeWorkspaceHref(pathname) !== "";
}
