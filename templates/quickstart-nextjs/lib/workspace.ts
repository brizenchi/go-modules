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
    label: { en: "My account", zh: "账户中心" },
    description: { en: "Account and billing services", zh: "账户与计费服务" }
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
  { href: "/credits", icon: "credits", label: { en: "Credits", zh: "积分明细" }, description: { en: "Balance and transaction history", zh: "余额、消费与奖励流水" } }
];

export const accountServiceItems: WorkspaceNavItem[] = [workspaceNavItems[1], auxiliaryWorkspaceItems[0], workspaceNavItems[2]];

/** Replaceable developer examples, never default customer tools. */
export const exampleWorkspaceItems: WorkspaceNavItem[] = [
  { href: "/notes", icon: "notes", label: { en: "Note export example", zh: "笔记导出示例" }, description: { en: "Example of saving and credit consumption", zh: "演示内容保存与积分扣费" } },
  { href: "/files", icon: "files", label: { en: "Private upload example", zh: "私有上传示例" }, description: { en: "Example of storage and ownership", zh: "演示存储接入与资源归属" } }
];

export function activeWorkspaceHref(pathname: string): string {
  const normalized = pathname.split(/[?#]/, 1)[0] || "/";
  return [...workspaceNavItems, ...auxiliaryWorkspaceItems, ...exampleWorkspaceItems].find((item) =>
    normalized === item.href || normalized.startsWith(`${item.href}/`)
  )?.href || "";
}

export function isWorkspacePath(pathname: string): boolean {
  return activeWorkspaceHref(pathname) !== "";
}

export function customerSignInDestination(currentPath: string): string {
  return isWorkspacePath(currentPath) ? currentPath : "/account";
}
