import assert from "node:assert/strict";
import { test } from "node:test";
import type { SubscriptionPreview, SubscriptionView } from "../lib/api";
import { subscriptionPlanAction, canConfirmSubscriptionChange, subscriptionPreviewAmount } from "../lib/subscription-change";

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
