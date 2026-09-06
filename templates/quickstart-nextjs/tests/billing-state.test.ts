import assert from "node:assert/strict";
import { test } from "node:test";
import {
  CHECKOUT_REFRESH_DELAYS_MS,
  checkoutReturnStatus,
  hasOngoingRecurringSubscription,
  parseCreditsQuantity
} from "../lib/billing-state";

test("ongoing recurring subscriptions block lifetime checkout", () => {
  for (const status of ["trialing", "active", "past_due", "canceling", "incomplete", "payment_failed", ""]) {
    assert.equal(hasOngoingRecurringSubscription({ plan: "pro", status }), true, status || "empty status");
  }
});

test("free, lifetime, and ended subscriptions can enter lifetime checkout", () => {
  assert.equal(hasOngoingRecurringSubscription(null), false);
  assert.equal(hasOngoingRecurringSubscription({ plan: "free", status: "inactive" }), false);
  assert.equal(hasOngoingRecurringSubscription({ plan: "lifetime", status: "active" }), false);
  for (const status of ["canceled", "inactive", "expired", "ended", "incomplete_expired"]) {
    assert.equal(hasOngoingRecurringSubscription({ plan: "starter", status }), false, status);
  }
});

test("credits quantity accepts only whole numbers from 1 through 100", () => {
  assert.equal(parseCreditsQuantity("1"), 1);
  assert.equal(parseCreditsQuantity(" 100 "), 100);
  for (const value of ["", "0", "101", "1.5", "1e2", "-1", "not-a-number"]) {
    assert.equal(parseCreditsQuantity(value), null, value);
  }
});

test("checkout return parsing accepts only supported terminal values", () => {
  assert.equal(checkoutReturnStatus("?checkout=success"), "success");
  assert.equal(checkoutReturnStatus("?source=stripe&checkout=CANCELLED"), "cancelled");
  assert.equal(checkoutReturnStatus("?checkout=pending"), null);
  assert.equal(checkoutReturnStatus(""), null);
});

test("checkout refresh uses a bounded backoff window", () => {
  assert.deepEqual(CHECKOUT_REFRESH_DELAYS_MS, [0, 1000, 2000, 4000, 8000]);
  assert.ok(CHECKOUT_REFRESH_DELAYS_MS.length <= 5);
});
