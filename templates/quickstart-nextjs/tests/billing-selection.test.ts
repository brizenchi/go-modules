import assert from "node:assert/strict";
import { test } from "node:test";
import { billingPlaceholderState, billingSelectionPath, initialBillingSection, isBillingIntentAvailable } from "../lib/billing-selection";

const billing = {
  enabled: true,
  offers: {
    subscriptions: [{ plan: "starter", intervals: ["monthly"] }, { plan: "pro", intervals: ["monthly", "yearly"] }],
    lifetime: false,
    credits: true
  }
};

test("a selected subscription must match both a configured plan and interval", () => {
  assert.equal(isBillingIntentAvailable({ product: "subscription", plan: "pro", interval: "yearly" }, billing), true);
  assert.equal(isBillingIntentAvailable({ product: "subscription", plan: "starter", interval: "yearly" }, billing), false);
  assert.equal(isBillingIntentAvailable({ product: "subscription", plan: "premium", interval: "monthly" }, billing), false);
});

test("one-time purchase availability follows its own offer rather than another enabled product", () => {
  assert.equal(isBillingIntentAvailable({ product: "lifetime" }, billing), false);
  assert.equal(isBillingIntentAvailable({ product: "credits" }, billing), true);
  assert.equal(isBillingIntentAvailable({ product: "credits" }, { ...billing, offers: { ...billing.offers, credits: false } }), false);
});

test("unknown or disabled billing never enables an incoming purchase intent", () => {
  assert.equal(isBillingIntentAvailable({ product: "credits" }, undefined), false);
  assert.equal(isBillingIntentAvailable({ product: "subscription", plan: "pro", interval: "yearly" }, { ...billing, enabled: false }), false);
});

test("changing a selection keeps checkout and unrelated query state for sign-in return", () => {
  const path = billingSelectionPath("?product=subscription&plan=starter&interval=monthly&checkout=success&source=pricing", { product: "subscription", plan: "pro", interval: "yearly" }, billing);
  const url = new URL(path!, "https://example.test");
  assert.equal(url.pathname, "/billing");
  assert.equal(url.searchParams.get("checkout"), "success");
  assert.equal(url.searchParams.get("source"), "pricing");
  assert.equal(url.searchParams.get("plan"), "pro");
  assert.equal(url.searchParams.get("interval"), "yearly");
  assert.equal(url.hash, "#plans");
});

test("one-time selection replaces subscription parameters and unavailable choices do not produce a URL", () => {
  const path = billingSelectionPath("?product=subscription&plan=pro&interval=yearly", { product: "credits" }, billing);
  assert.equal(path, "/billing?product=credits#credits");
  assert.equal(billingSelectionPath("?product=subscription&plan=unknown&interval=monthly", { product: "subscription", plan: "unknown", interval: "monthly" }, billing), null);
});

test("billing disclosure follows the incoming purchase intent before account defaults", () => {
  assert.equal(initialBillingSection({ product: "credits" }, { plan: "free", status: "inactive" }), "credits");
  assert.equal(initialBillingSection({ product: "lifetime" }, { plan: "pro", status: "active" }), "plans");
  assert.equal(initialBillingSection({ product: "subscription", plan: "pro", interval: "yearly" }, { plan: "pro", status: "active" }), "plans");
  assert.equal(initialBillingSection(null, { plan: "free", status: "inactive" }), "plans");
  assert.equal(initialBillingSection(null, { plan: "pro", status: "active" }), null);
  assert.equal(initialBillingSection(null, { plan: "lifetime", status: "active" }), null);
  assert.equal(initialBillingSection(null, undefined), null);
});

test("authenticated billing placeholders distinguish loading from an unavailable service", () => {
  const state = { sessionReady: true, signedIn: true, capabilities: "ready" as const, billingEnabled: false, resource: "idle" as const };
  assert.equal(billingPlaceholderState(state), "unavailable");
  assert.equal(billingPlaceholderState({ ...state, capabilities: "error" }), "unavailable");
  assert.equal(billingPlaceholderState({ ...state, capabilities: "loading" }), "loading");
  assert.equal(billingPlaceholderState({ ...state, capabilities: "idle" }), "loading");
  assert.equal(billingPlaceholderState({ ...state, billingEnabled: true }), "loading");
  assert.equal(billingPlaceholderState({ ...state, signedIn: false }), "signed_out");
  assert.equal(billingPlaceholderState({ ...state, sessionReady: false, signedIn: false }), "loading");
});
