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
    description: { en: "Account and billing services", zh: "账户与计费服务" }
  },
  {
    href: "/billing",
    icon: "billing",
    label: siteLinks.billing.label,
    description: { en: "Plans and payments", zh: "套餐与支付" }
  },
  {
    href: "/referrals",
    icon: "referrals",
    label: siteLinks.referrals.label,
    description: { en: "Invite and earn", zh: "邀请与奖励" }
  }
];

// Personal product pages retain the account layout but are not primary account
// navigation. Site-wide operations use the separate /admin layout.
export const auxiliaryWorkspaceItems: WorkspaceNavItem[] = [
  { href: "/credits", icon: "credits", label: siteLinks.credits.label, description: { en: "Balance and transaction history", zh: "余额、消费与奖励流水" } }
];

export const accountServiceItems: WorkspaceNavItem[] = [workspaceNavItems[1], auxiliaryWorkspaceItems[0], workspaceNavItems[2]];

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
