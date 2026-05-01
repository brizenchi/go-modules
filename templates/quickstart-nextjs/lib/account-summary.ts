import {
  getReferralStats,
  getSubscription,
  type ReferralStats,
  type SubscriptionView
} from "./api";

export type AccountSummary = {
  subscription: SubscriptionView | null;
  referralStats: ReferralStats | null;
};

type SummaryDeps = {
  getSubscription: typeof getSubscription;
  getReferralStats: typeof getReferralStats;
};

const defaultDeps: SummaryDeps = {
  getSubscription,
  getReferralStats
};

export async function loadAccountSummary(
  token: string,
  deps: SummaryDeps = defaultDeps
): Promise<AccountSummary> {
  const [subscription, referralStats] = await Promise.allSettled([
    deps.getSubscription(token),
    deps.getReferralStats(token)
  ]);

  return {
    subscription: subscription.status === "fulfilled" ? subscription.value : null,
    referralStats: referralStats.status === "fulfilled" ? referralStats.value : null
  };
}
