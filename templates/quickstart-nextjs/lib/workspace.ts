import { siteLinks } from "../content/navigation";

export type WorkspaceIcon = "billing" | "referrals" | "account" | "credits";

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
    label: siteLinks.account.label,
    description: { en: "Identity and membership", zh: "身份与会员等级" }
  },
  {
    href: "/billing",
    icon: "billing",
    label: siteLinks.billing.label,
    description: { en: "Plans and payments", zh: "套餐与支付" }
  },
  {
    href: "/credits",
    icon: "credits",
    label: siteLinks.credits.label,
    description: { en: "Balance and transaction history", zh: "余额、消费与奖励流水" }
  },
  {
    href: "/referrals",
    icon: "referrals",
    label: siteLinks.referrals.label,
    description: { en: "Invite and earn", zh: "邀请与奖励" }
  }
];

// Reserved for product-owned pages that share the customer layout.
// Site-wide operations use the separate /admin layout.
export const auxiliaryWorkspaceItems: WorkspaceNavItem[] = [];

export function activeWorkspaceHref(pathname: string): string {
  const normalized = pathname.split(/[?#]/, 1)[0] || "/";
  return [...workspaceNavItems, ...auxiliaryWorkspaceItems].find((item) =>
    normalized === item.href || normalized.startsWith(`${item.href}/`)
  )?.href || "";
}

export function isWorkspacePath(pathname: string): boolean {
  return activeWorkspaceHref(pathname) !== "";
}

export function customerSignInDestination(currentPath: string): string {
  return isWorkspacePath(currentPath) ? currentPath : "/account";
}
