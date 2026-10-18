import assert from "node:assert/strict";
import test from "node:test";
import { createPricingSubscriptionCache } from "../lib/pricing-subscription-cache";

test("pricing revisits reuse recent data and concurrent loads share one request", async () => {
  let now = 0, calls = 0;
  const cache = createPricingSubscriptionCache(async () => { calls++; return { plan: "premium", status: "active" }; }, () => now);
  await Promise.all([cache.load("user-a"), cache.load("user-a")]);
  await cache.load("user-a");
  assert.equal(calls, 1);
  assert.equal(cache.peek("user-a")?.plan, "premium");
  now = 60_000;
  assert.equal(cache.peek("user-a")?.plan, "premium", "keep stale presentation during background update");
  await cache.load("user-a");
  assert.equal(calls, 2);
  await cache.load("user-a", true);
  assert.equal(calls, 3, "plan changes bypass freshness window");
  cache.invalidate();
  assert.equal(cache.peek("user-a")?.plan, "premium", "invalidation preserves presentation");
  await cache.load("user-a");
  assert.equal(calls, 4);
});

test("logout and account switches never reuse another account's snapshot", async () => {
  const cache = createPricingSubscriptionCache(async token => ({ plan: token, status: "active" }));
  await cache.load("user-a");
  assert.equal(cache.peek("user-b"), null);
  await cache.load("user-b");
  assert.equal(cache.peek("user-a"), null);
  cache.clear();
  assert.equal(cache.peek("user-b"), null);
});

test("late responses after clearing cannot restore a previous login's data", async () => {
  let resolve!: (value: { plan: string; status: string }) => void;
  const cache = createPricingSubscriptionCache(() => new Promise(done => { resolve = done; }));
  const request = cache.load("user-a");
  cache.clear();
  resolve({ plan: "premium", status: "active" });
  await request;
  assert.equal(cache.peek("user-a"), null);
});

test("failed refresh retains the previous snapshot and can retry", async () => {
  let fail = false;
  const cache = createPricingSubscriptionCache(async () => { if (fail) throw new TypeError("offline"); return { plan: "pro", status: "active" }; });
  await cache.load("user-a");
  fail = true;
  await assert.rejects(cache.load("user-a", true));
  assert.equal(cache.peek("user-a")?.plan, "pro");
  fail = false;
  assert.equal((await cache.load("user-a", true)).plan, "pro");
});
