import assert from "node:assert/strict";
import { test } from "node:test";
import type { SubscriptionPreview, SubscriptionView } from "../lib/api";
import { cancelRenewalForSubscription, confirmPlanSelection, isScheduledSelection, keepCurrentSubscription, removeScheduledChangeForSelection, subscriptionRenewalAction } from "../lib/subscription-management";

const pro: SubscriptionView = { plan: "pro", status: "active", billing_cycle: "monthly" };
const scheduled = { ...pro, scheduled_change: { plan: "starter", billing_cycle: "monthly" } };
const canceled = { ...pro, cancel_at_period_end: true };
const premium = { plan: "premium", interval: "monthly" };
const price: SubscriptionPreview = { currency: "usd", amount_due_now: 45, target_plan: "premium", target_interval: "monthly", change_mode: "immediate_prorated", immediate_charge: true, effective_at_period_end: false, message: "" };

test("keeping Premium only removes the scheduled Pro downgrade and preserves renewal", async () => {
  const current: SubscriptionView = { plan: "premium", status: "active", billing_cycle: "yearly", current_period_end: "2027-01-01T00:00:00Z" };
  const reviewed = { ...current, scheduled_change: { plan: "pro", billing_cycle: "monthly", effective_at: current.current_period_end } };
  let snapshot: SubscriptionView = reviewed;
  const calls: string[] = [];
  const result = await keepCurrentSubscription(reviewed, {
    load: async () => { calls.push("read"); return snapshot; }, current: () => true,
    removeSchedule: async () => { calls.push("remove schedule"); snapshot = current; }
  });
  assert.deepEqual(result, { state: "kept", subscription: current });
  assert.deepEqual(calls, ["read", "remove schedule", "read"]);
});

test("keeping a current plan refuses a changed subscription or a newly scheduled target", async () => {
  let removes = 0;
  for (const latest of [{ ...scheduled, plan: "starter" }, { ...scheduled, billing_cycle: "yearly" },
    { ...scheduled, scheduled_change: { plan: "premium", billing_cycle: "monthly" } },
    { ...scheduled, pending_payment: { amount_due: 100, currency: "usd" } },
    { ...scheduled, cancel_at_period_end: true }]) {
    const result = await keepCurrentSubscription(scheduled, { load: async () => latest, current: () => true, removeSchedule: async () => { removes++; } });
    assert.equal(result?.state, "review");
  }
  assert.equal(removes, 0);
});

test("keeping a current plan retries a lost successful response without removing again", async () => {
  let snapshot: SubscriptionView = scheduled;
  let removes = 0;
  const deps = { load: async () => snapshot, current: () => true, removeSchedule: async () => { removes++; snapshot = pro; throw new TypeError("fixture response lost"); } };
  await assert.rejects(keepCurrentSubscription(scheduled, deps), TypeError);
  assert.equal((await keepCurrentSubscription(scheduled, deps))?.state, "kept");
  assert.equal(removes, 1);
});

test("failed, stale and unconfirmed schedule removals cannot report the plan kept", async () => {
  const deps = { load: async () => scheduled, current: () => true, removeSchedule: async () => {} };
  assert.equal((await keepCurrentSubscription(scheduled, deps))?.state, "review");
  assert.equal(await keepCurrentSubscription(scheduled, { ...deps, current: () => false, removeSchedule: async () => { assert.fail("stale account"); } }), null);
  await assert.rejects(keepCurrentSubscription(scheduled, { ...deps, removeSchedule: async () => { throw new TypeError("fixture offline"); } }), TypeError);
  let current = true;
  assert.equal(await keepCurrentSubscription(scheduled, { ...deps, current: () => current, removeSchedule: async () => { current = false; } }), null);
});

test("selecting an already scheduled target shows its result without replacing it", async () => {
  const selection = { plan: "starter", interval: "monthly" };
  assert.equal(isScheduledSelection(selection, scheduled), true);
  assert.equal(isScheduledSelection(premium, scheduled), false);
  assert.equal(isScheduledSelection({ ...selection, interval: "yearly" }, scheduled), false);
  let writes = 0;
  const result = await confirmPlanSelection({ selection, subscription: scheduled, preview: { ...price, target_plan: "starter", amount_due_now: 0, change_mode: "period_end", immediate_charge: false, effective_at_period_end: true } }, {
    load: async () => scheduled, current: () => true, preview: async () => price,
    removeSchedule: async () => { writes++; }, submit: async () => { writes++; return "changed"; }
  });
  assert.equal(result?.state, "review");
  assert.equal(writes, 0);
});

test("one confirmation replaces the reviewed schedule with the exact selected plan", async () => {
  let snapshot: SubscriptionView = scheduled;
  const calls: string[] = [];
  const result = await confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, {
    load: async () => { calls.push("read"); return snapshot; }, current: () => true,
    removeSchedule: async () => { calls.push("remove schedule"); snapshot = pro; },
    preview: async () => { calls.push("preview"); return price; },
    submit: async () => { calls.push("submit premium monthly"); return "changed"; }
  });
  assert.deepEqual(result, { state: "submitted", result: "changed" });
  assert.deepEqual(calls, ["read", "remove schedule", "read", "preview", "submit premium monthly"]);
});

test("a newer schedule, payment or current plan is never silently overwritten", async () => {
  let writes = 0;
  for (const latest of [
    { ...scheduled, scheduled_change: { plan: "starter", billing_cycle: "yearly" } },
    { ...scheduled, pending_payment: { amount_due: 4500, currency: "usd" } },
    { ...scheduled, plan: "premium" }, { ...scheduled, current_period_end: "2026-12-01T00:00:00Z" }
  ]) {
    const result = await confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, {
      load: async () => latest, current: () => true, preview: async () => price,
      removeSchedule: async () => { writes++; }, submit: async () => { writes++; }
    });
    assert.equal(result?.state, "review");
  }
  assert.equal(writes, 0);
});

test("a changed amount or effective date requires another review before payment", async () => {
  for (const next of [{ ...price, amount_due_now: 46 }, { ...price, next_billing_at: "2026-12-01T00:00:00Z" }, { ...price, currency: "eur" }]) {
    let submits = 0;
    const result = await confirmPlanSelection({ selection: premium, subscription: pro, preview: price }, {
      load: async () => pro, current: () => true, preview: async () => next,
      removeSchedule: async () => { assert.fail("no schedule to remove"); }, submit: async () => { submits++; }
    });
    assert.equal(result?.state, "review");
    assert.equal(submits, 0);
  }
});

test("failed removal and stale sessions never submit the selected plan", async () => {
  let submits = 0;
  const deps = { load: async () => scheduled, current: () => true, preview: async () => price,
    removeSchedule: async () => { throw new TypeError("fixture offline"); }, submit: async () => { submits++; } };
  await assert.rejects(confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, deps), TypeError);
  assert.equal(await confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, { ...deps, current: () => false }), null);
  let current = true;
  assert.equal(await confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, { ...deps,
    current: () => current, removeSchedule: async () => { current = false; } }), null);
  assert.equal(submits, 0);
});

test("a new pending payment after removal blocks the replacement", async () => {
  let snapshot: SubscriptionView = scheduled;
  const result = await confirmPlanSelection({ selection: premium, subscription: scheduled, preview: price }, {
    load: async () => snapshot, current: () => true, preview: async () => price,
    removeSchedule: async () => { snapshot = { ...pro, pending_payment: { amount_due: 4500, currency: "usd" } }; },
    submit: async () => { assert.fail("payment must be resolved first"); }
  });
  assert.equal(result?.state, "review");
});

test("ongoing subscriptions offer Cancel or Renew regardless of a scheduled plan", () => {
  for (const sub of [pro, scheduled, { ...pro, status: "past_due" }]) assert.equal(subscriptionRenewalAction(sub), "cancel");
  for (const sub of [canceled, { ...pro, status: "canceling" }, { ...pro, effective_cancel_at: "2026-11-01T00:00:00Z" }]) assert.equal(subscriptionRenewalAction(sub), "renew");
  for (const sub of [null, { plan: "free", status: "inactive" }, { plan: "lifetime", status: "active" }, { ...pro, status: "canceled" }]) assert.equal(subscriptionRenewalAction(sub), null);
});

test("confirmed cancellation clears a schedule before canceling renewal", async () => {
  let snapshot: SubscriptionView = scheduled;
  const calls: string[] = [];
  const result = await cancelRenewalForSubscription({
    load: async () => { calls.push("read"); return snapshot; },
    removeSchedule: async () => { calls.push("remove schedule"); snapshot = pro; },
    cancel: async () => { calls.push("cancel renewal"); snapshot = canceled; },
    current: () => true
  });
  assert.equal(result?.state, "canceled");
  assert.deepEqual(calls, ["read", "remove schedule", "read", "cancel renewal", "read"]);
});

test("schedule removal failure never submits cancellation", async () => {
  let cancels = 0;
  await assert.rejects(cancelRenewalForSubscription({ load: async () => scheduled,
    removeSchedule: async () => { throw new TypeError("fixture offline"); },
    cancel: async () => { cancels++; }, current: () => true }), TypeError);
  assert.equal(cancels, 0);
});

test("lost cancellation response can be retried without a duplicate write", async () => {
  let snapshot: SubscriptionView = pro;
  let cancels = 0;
  const deps = { load: async () => snapshot, removeSchedule: async () => {},
    cancel: async () => { cancels++; snapshot = canceled; throw new TypeError("fixture offline"); }, current: () => true };
  await assert.rejects(cancelRenewalForSubscription(deps), TypeError);
  assert.equal((await cancelRenewalForSubscription(deps))?.state, "canceled");
  assert.equal(cancels, 1);
});

test("pending payment and stale sessions cannot trigger cancellation writes", async () => {
  let writes = 0;
  const write = async () => { writes++; };
  const deps = { load: async () => ({ ...scheduled, pending_payment: { amount_due: 4500, currency: "usd" } }), removeSchedule: write, cancel: write, current: () => true };
  assert.equal((await cancelRenewalForSubscription(deps))?.state, "blocked");
  assert.equal(await cancelRenewalForSubscription({ ...deps, load: async () => scheduled, current: () => false }), null);
  let current = true;
  assert.equal(await cancelRenewalForSubscription({ ...deps, load: async () => scheduled,
    removeSchedule: async () => { current = false; }, current: () => current }), null);
  assert.equal(writes, 0);
});

test("an unconfirmed or ended subscription is never reported as canceled", async () => {
  const deps = { load: async () => pro, removeSchedule: async () => {}, cancel: async () => {}, current: () => true };
  assert.equal((await cancelRenewalForSubscription(deps))?.state, "unconfirmed");
  assert.equal((await cancelRenewalForSubscription({ ...deps, load: async () => ({ ...pro, status: "canceled" }) }))?.state, "ended");
});

test("removing a schedule only changes the schedule, preserving the chosen subscription", async () => {
  let snapshot: SubscriptionView = scheduled;
  let removes = 0;
  const deps = { load: async () => snapshot, removeSchedule: async () => { removes++; snapshot = pro; }, current: () => true };
  assert.deepEqual(await removeScheduledChangeForSelection(deps), pro);
  assert.deepEqual(await removeScheduledChangeForSelection(deps), pro);
  assert.equal(removes, 1);
  assert.equal(await removeScheduledChangeForSelection({ ...deps, current: () => false }), null);
});
