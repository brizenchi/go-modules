import assert from "node:assert/strict";
import { test } from "node:test";
import { loadAccountSummary } from "../lib/account-summary";

test("loadAccountSummary returns both values when both backend reads succeed", async () => {
  const summary = await loadAccountSummary("jwt-token", {
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
    subscription: {
      plan: "pro",
      status: "active"
    },
    referralStats: {
      total_referred: 8,
      activated: 5,
      pending: 3,
      total_reward_credits: 150
    }
  });
});

test("loadAccountSummary degrades gracefully when one backend read fails", async () => {
  const summary = await loadAccountSummary("jwt-token", {
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

  assert.deepEqual(summary, {
    subscription: null,
    referralStats: {
      total_referred: 2,
      activated: 1,
      pending: 1,
      total_reward_credits: 30
    }
  });
});
