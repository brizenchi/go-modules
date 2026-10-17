import assert from "node:assert/strict";
import { test } from "node:test";
import type { SubscriptionPreview, SubscriptionView } from "../lib/api";
import { subscriptionPlanAction, subscriptionPlanIntent, subscriptionChangeBlocker, canConfirmSubscriptionChange, subscriptionPreviewAmount } from "../lib/subscription-change";

const pro: SubscriptionView = { plan: "pro", status: "active", billing_cycle: "monthly" };
const selection = { plan: "premium", interval: "monthly" };
const preview: SubscriptionPreview = { currency: "usd", amount_due_now: 45, target_plan: "premium", target_interval: "monthly", change_mode: "immediate_prorated", immediate_charge: true, effective_at_period_end: false, message: "" };

test("subscription preview amounts retain the API's major currency units", () => {
  assert.equal(subscriptionPreviewAmount(preview, "en-US"), "$45.00");
  assert.equal(subscriptionPreviewAmount({ ...preview, amount_due_now: 30.45 }, "en-US"), "$30.45");
  assert.equal(subscriptionPreviewAmount({ ...preview, amount_due_now: 0 }, "en-US"), "$0.00");
  assert.equal(subscriptionPreviewAmount(preview, "zh-CN"), "US$45.00");
});

test("pricing distinguishes the current offer, upgrades, downgrades and interval changes", () => {
  assert.equal(subscriptionPlanAction({ plan: "pro", interval: "monthly" }, pro), "current");
  assert.equal(subscriptionPlanAction(selection, pro), "upgrade");
  assert.equal(subscriptionPlanAction({ plan: "starter", interval: "monthly" }, pro), "downgrade");
  assert.equal(subscriptionPlanAction({ plan: "pro", interval: "yearly" }, pro), "interval");
  assert.equal(subscriptionPlanAction(selection, { ...pro, status: "canceled" }), "subscribe");
  assert.equal(subscriptionPlanAction(selection, null), "subscribe");
  assert.equal(subscriptionPlanAction(selection, { plan: "lifetime", status: "active" }), "lifetime");
});

test("all recurring tiers and both intervals keep the same upgrade and downgrade meaning", () => {
  const choices = {
    starter: { starter: "current", pro: "upgrade", premium: "upgrade" },
    pro: { starter: "downgrade", pro: "current", premium: "upgrade" },
    premium: { starter: "downgrade", pro: "downgrade", premium: "current" }
  } as const;
  for (const currentPlan of ["starter", "pro", "premium"] as const) {
    for (const billing_cycle of ["monthly", "yearly"]) {
      for (const plan of ["starter", "pro", "premium"] as const) {
        for (const interval of ["monthly", "yearly"]) {
          const expected = currentPlan === plan && billing_cycle !== interval ? "interval" : choices[currentPlan][plan];
          assert.equal(subscriptionPlanAction({ plan, interval }, { plan: currentPlan, billing_cycle, status: "active" }), expected, `${currentPlan}/${billing_cycle} -> ${plan}/${interval}`);
        }
      }
    }
  }
});

test("pending payments, scheduled changes and canceled renewal require management first", () => {
  for (const current of [
    { ...pro, pending_payment: { amount_due: 100, currency: "usd" } },
    { ...pro, scheduled_change: { plan: "starter", billing_cycle: "monthly" } },
    { ...pro, cancel_at_period_end: true },
    { ...pro, effective_cancel_at: "2026-11-01T00:00:00Z" },
    { ...pro, status: "incomplete" },
    { ...pro, status: "past_due" }
  ]) assert.equal(subscriptionPlanAction(selection, current), "manage");
});

test("canceled renewal keeps each card's intended action and exposes one cancellation blocker", () => {
  const ending = { ...pro, status: "canceling", cancel_at_period_end: true };
  assert.equal(subscriptionPlanIntent({ plan: "starter", interval: "yearly" }, ending), "downgrade");
  assert.equal(subscriptionPlanIntent({ plan: "pro", interval: "yearly" }, ending), "interval");
  assert.equal(subscriptionPlanIntent({ plan: "premium", interval: "yearly" }, ending), "upgrade");
  assert.equal(subscriptionPlanIntent({ plan: "pro", interval: "monthly" }, ending), "current");
  assert.equal(subscriptionChangeBlocker(ending), "cancellation");
  assert.equal(subscriptionChangeBlocker({ ...pro, effective_cancel_at: "2026-11-01T00:00:00Z" }), "cancellation");
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "canceling" }), "cancellation");
  assert.equal(subscriptionPlanAction(selection, ending), "manage");
  assert.equal(canConfirmSubscriptionChange({ selection, subscription: ending, preview, ready: true, busy: false }), false);
});

test("payment and scheduled-change blockers remain distinct from the selected target", () => {
  const pending = { ...pro, pending_payment: { amount_due: 100, currency: "usd" } };
  const scheduled = { ...pro, scheduled_change: { plan: "starter", billing_cycle: "monthly" } };
  assert.equal(subscriptionChangeBlocker(pending), "payment");
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "past_due" }), "payment");
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "incomplete" }), "payment");
  assert.equal(subscriptionChangeBlocker(scheduled), "scheduled");
  for (const current of [pending, scheduled]) {
    assert.equal(subscriptionPlanIntent(selection, current), "upgrade");
    assert.equal(subscriptionPlanAction(selection, current), "manage");
  }
  assert.equal(subscriptionChangeBlocker(pro), null);
  assert.equal(subscriptionChangeBlocker(null), null);
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "canceled" }), null);
  assert.equal(subscriptionChangeBlocker({ plan: "lifetime", status: "active" }), null);
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "paused" }), "unavailable");
});

test("confirmation needs a successful preview for the exact selection and a changeable subscription", () => {
  const state = { selection, subscription: pro, preview, ready: true, busy: false };
  assert.equal(canConfirmSubscriptionChange(state), true);
  assert.equal(canConfirmSubscriptionChange({ ...state, ready: false }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, busy: true }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: null }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, selection: { ...selection, interval: "yearly" } }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, selection: { ...selection, plan: "starter" } }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, subscription: { ...pro, pending_payment: { amount_due: 100, currency: "usd" } } }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, selection: { plan: "pro", interval: "monthly" }, preview: { ...preview, target_plan: "pro" } }), false);
});

test("an invalid price preview cannot enable a payment confirmation", () => {
  const state = { selection, subscription: pro, preview, ready: true, busy: false };
  for (const amount_due_now of [NaN, Infinity, -1]) {
    assert.equal(canConfirmSubscriptionChange({ ...state, preview: { ...preview, amount_due_now } }), false);
  }
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: { ...preview, currency: "" } }), false);
  const scheduled: SubscriptionPreview = { ...preview, amount_due_now: 0, change_mode: "period_end", immediate_charge: false, effective_at_period_end: true };
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: scheduled }), true);
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: { ...scheduled, amount_due_now: 10 } }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: { ...scheduled, immediate_charge: true } }), false);
  assert.equal(canConfirmSubscriptionChange({ ...state, preview: { ...preview, effective_at_period_end: true } }), false);
});
