import assert from "node:assert/strict";
import { test } from "node:test";
import { loadAccountSummary } from "../lib/account-summary";

test("loadAccountSummary returns both values when both backend reads succeed", async () => {
  const summary = await loadAccountSummary("jwt-token", {
    getCapabilities: async () => ({
      account: { enabled: true },
      billing: { enabled: true, provider: "stripe", offers: { subscriptions: [], lifetime: false, credits: false } },
      referral: { enabled: true }
    }),
    getSubscription: async () => ({
      plan: "pro",
      status: "active"
    }),
    getReferralStats: async () => ({
      total_referred: 8,
      activated: 5,
      pending: 3,
      total_reward_credits: 150
    })
  });

  assert.deepEqual(summary, {
    capabilities: {
      status: "ready",
      data: {
        account: { enabled: true },
        billing: { enabled: true, provider: "stripe", offers: { subscriptions: [], lifetime: false, credits: false } },
        referral: { enabled: true }
      },
      failure: null
    },
    subscription: { status: "ready", data: { plan: "pro", status: "active" }, failure: null },
    referralStats: { status: "ready", data: { total_referred: 8, activated: 5, pending: 3, total_reward_credits: 150 }, failure: null }
  });
});

test("loadAccountSummary degrades gracefully when one backend read fails", async () => {
  const summary = await loadAccountSummary("jwt-token", {
    getCapabilities: async () => ({
      account: { enabled: true },
      billing: { enabled: false, provider: "stripe", offers: { subscriptions: [], lifetime: false, credits: false } },
      referral: { enabled: true }
    }),
    getSubscription: async () => {
      throw new Error("stripe unavailable");
    },
    getReferralStats: async () => ({
      total_referred: 2,
      activated: 1,
      pending: 1,
      total_reward_credits: 30
    })
  });

  assert.equal(summary.capabilities.status, "ready");
  assert.equal(summary.subscription.status, "error");
  assert.equal(summary.subscription.failure?.title, "Could not load subscription billing");
  assert.deepEqual(summary.referralStats, {
    status: "ready",
    data: { total_referred: 2, activated: 1, pending: 1, total_reward_credits: 30 },
    failure: null
  });
});
