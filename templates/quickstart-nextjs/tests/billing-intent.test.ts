import assert from "node:assert/strict";
import { test } from "node:test";
import { billingIntentHref, readBillingIntent, type BillingIntent } from "../lib/billing-intent";

test("subscription links retain the chosen plan and interval in the billing workspace", () => {
  const intent: BillingIntent = { product: "subscription", plan: "starter", interval: "yearly" };
  const href = billingIntentHref(intent);
  assert.equal(href, "/billing?product=subscription&plan=starter&interval=yearly#plans");
  assert.deepEqual(readBillingIntent(new URL(href, "https://example.test").search), intent);
});

test("one-time purchases lead to their own billing sections", () => {
  assert.equal(billingIntentHref({ product: "lifetime" }), "/billing?product=lifetime#plans");
  assert.equal(billingIntentHref({ product: "credits" }), "/billing?product=credits#credits");
  assert.deepEqual(readBillingIntent("?product=lifetime"), { product: "lifetime" });
  assert.deepEqual(readBillingIntent("?product=credits"), { product: "credits" });
});

test("safe custom plan slugs are parsed without hard-coding the example catalog", () => {
  assert.deepEqual(readBillingIntent("?product=subscription&plan=team-2026&interval=monthly&source=pricing"), {
    product: "subscription", plan: "team-2026", interval: "monthly"
  });
});

test("missing or unsupported purchase parameters never select a default product", () => {
  for (const search of [
    "", "?source=pricing", "?plan=starter&interval=monthly", "?product=subscription", 
    "?product=subscription&plan=starter", "?product=subscription&interval=monthly",
    "?product=subscription&plan=starter&interval=weekly", "?product=subscription&plan=starter&interval=MONTHLY",
    "?product=unknown", "?product=", "?product=credits&plan=starter", "?product=lifetime&interval=yearly"
  ]) {
    assert.equal(readBillingIntent(search), null, search);
  }
});

test("ambiguous repeated parameters are rejected", () => {
  for (const search of [
    "?product=credits&product=lifetime",
    "?product=subscription&plan=starter&plan=pro&interval=monthly",
    "?product=subscription&plan=starter&interval=monthly&interval=yearly"
  ]) {
    assert.equal(readBillingIntent(search), null, search);
  }
});

test("plan values reject unsafe paths, whitespace, and excessively long input", () => {
  for (const plan of ["", "Starter", " pro", "pro ", "../admin", "https://example.test", "pro&interval=yearly", "-pro", "pro-", "pro--team", "a".repeat(65)]) {
    assert.equal(readBillingIntent(`?product=subscription&plan=${encodeURIComponent(plan)}&interval=monthly`), null, plan);
  }
});
