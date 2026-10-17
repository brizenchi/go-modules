import { stripeInvoiceURL } from "../lib/billing-state";
import assert from "node:assert/strict";
import { test } from "node:test";
import {
  CHECKOUT_REFRESH_DELAYS_MS,
  checkoutReturnStatus,
  hasOngoingRecurringSubscription,
  subscriptionUpgradeAction,
  parseCreditsQuantity
} from "../lib/billing-state";

test("free and ended memberships upgrade through pricing", () => {
  assert.equal(subscriptionUpgradeAction("free", { plan: "free", status: "inactive" }), "pricing");
  for (const status of ["canceled", "expired", "ended", "inactive"]) {
    assert.equal(subscriptionUpgradeAction("free", { plan: "pro", status }), "pricing", status);
  }
});

test("recurring tiers upgrade through the existing subscription change flow", () => {
  for (const plan of ["starter", "pro", "premium"]) {
    for (const status of ["active", "trialing", "canceling", "past_due"]) {
      assert.equal(subscriptionUpgradeAction(plan, { plan, status }), "change", `${plan}: ${status}`);
    }
  }
});

test("lifetime membership disables upgrades even without a billing subscription", () => {
  assert.equal(subscriptionUpgradeAction("lifetime", null), "lifetime");
  assert.equal(subscriptionUpgradeAction(" Lifetime ", { plan: "free", status: "inactive" }), "lifetime");
  assert.equal(subscriptionUpgradeAction("lifetime", { plan: "pro", status: "active" }), "lifetime");
  assert.equal(subscriptionUpgradeAction(undefined, { plan: "lifetime", status: "active" }), "lifetime");
  assert.equal(subscriptionUpgradeAction("free", { plan: "lifetime", status: "ended" }), "pricing");
});

test("an unknown membership waits instead of presenting a purchase action", () => {
  assert.equal(subscriptionUpgradeAction(undefined, null), "loading");
  assert.equal(subscriptionUpgradeAction("", undefined), "loading");
});

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

 test("payment links accept only Stripe hosted invoices",()=>{
 assert.equal(stripeInvoiceURL("https://invoice.stripe.com/i/test"),"https://invoice.stripe.com/i/test");
 assert.equal(stripeInvoiceURL("javascript:alert(1)"),null);
 assert.equal(stripeInvoiceURL("https://invoice.stripe.com.evil.test/test"),null);
 assert.equal(stripeInvoiceURL("https://user:pass@invoice.stripe.com/test"),null);
 });
