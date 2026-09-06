import {
  getCapabilities,
  getReferralStats,
  getSubscription,
  type CapabilitiesView,
  type ReferralStats,
  type SubscriptionView
} from "./api";
import { settleResource, type ResourceState } from "./request-state";

export type AccountSummary = {
  capabilities: ResourceState<CapabilitiesView>;
  subscription: ResourceState<SubscriptionView>;
  referralStats: ResourceState<ReferralStats>;
};

type SummaryDeps = {
  getCapabilities: typeof getCapabilities;
  getSubscription: typeof getSubscription;
  getReferralStats: typeof getReferralStats;
};

const defaultDeps: SummaryDeps = {
  getCapabilities,
  getSubscription,
  getReferralStats
};

export async function loadAccountSummary(
  token: string,
  deps: SummaryDeps = defaultDeps
): Promise<AccountSummary> {
  const [capabilities, subscription, referralStats] = await Promise.all([
    settleResource(deps.getCapabilities(), "API capabilities"),
    settleResource(deps.getSubscription(token), "Subscription billing"),
    settleResource(deps.getReferralStats(token), "Referral center")
  ]);

  return {
    capabilities,
    subscription,
    referralStats
  };
}
