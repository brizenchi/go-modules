import assert from "node:assert/strict";
import { test } from "node:test";
import type { SubscriptionView } from "../lib/api";
import { cancelRenewalForSubscription, removeScheduledChangeForSelection, subscriptionRenewalAction } from "../lib/subscription-management";

const pro: SubscriptionView = { plan: "pro", status: "active", billing_cycle: "monthly" };
const scheduled = { ...pro, scheduled_change: { plan: "starter", billing_cycle: "monthly" } };
const canceled = { ...pro, cancel_at_period_end: true };

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
