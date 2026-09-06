import type { AccountProfile, SubscriptionView } from "./api";

export type DashboardProfile = Pick<AccountProfile, "email_verified" | "username" | "avatar_url">;

export type DashboardChecklistItem = {
  key: "profile" | "billing" | "referral";
  href: string;
  complete: boolean;
};

type DashboardChecklistInput = {
  profile: DashboardProfile | null;
  subscription: Pick<SubscriptionView, "plan" | "status"> | null;
  referralLink: string;
  billingEnabled: boolean;
  referralEnabled: boolean;
};

export function profileCompletion(profile: DashboardProfile | null): number {
  if (!profile) {
    return 0;
  }
  const completed = [
    profile.email_verified,
    profile.username.trim() !== "",
    profile.avatar_url.trim() !== ""
  ].filter(Boolean).length;
  return Math.min(100, Math.ceil((completed / 3) * 100));
}

export function humanizePlan(plan: string): string {
  const normalized = plan.trim().replace(/[-_]+/g, " ").replace(/\s+/g, " ");
  if (!normalized) {
    return "Free";
  }
  return normalized.charAt(0).toUpperCase() + normalized.slice(1).toLowerCase();
}

export function buildDashboardChecklist(input: DashboardChecklistInput): DashboardChecklistItem[] {
  const items: DashboardChecklistItem[] = [{
    key: "profile",
    href: "/account",
    complete: Boolean(input.profile?.email_verified && input.profile.username.trim())
  }];

  if (input.billingEnabled) {
    const status = input.subscription?.status.trim().toLowerCase() || "";
    items.push({
      key: "billing",
      href: "/billing",
      complete: status === "active" || status === "trialing"
    });
  }

  if (input.referralEnabled) {
    items.push({
      key: "referral",
      href: "/referrals",
      complete: input.referralLink.trim() !== ""
    });
  }

  return items;
}
