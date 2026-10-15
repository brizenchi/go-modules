import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiError, type CapabilitiesView, type CheckoutPayload, type SubscriptionView } from "../lib/api";
import { preparePricingCheckout, pricingPurchaseHref, readPricingPurchase, pricingCheckoutFailure } from "../lib/pricing-checkout";

const capabilities: CapabilitiesView = { account: { enabled: true }, referral: { enabled: true }, billing: { enabled: true, provider: "stripe", offers: { subscriptions: [{ plan: "pro", intervals: ["monthly", "yearly"] }], lifetime: true, credits: true } } };
const intent = { product: "subscription", plan: "pro", interval: "yearly" } as const;
const urls = { success: "https://example.test/zh/pricing?checkout=success", cancel: "https://example.test/zh/pricing?checkout=cancelled" };
function fixture(subscription: SubscriptionView = {plan:"free",status:"inactive"}) {
  const payloads: CheckoutPayload[] = [];
  const deps = { capabilities: async () => capabilities, subscription: async () => subscription, create: async (payload: CheckoutPayload) => { payloads.push(payload); return { checkout_url: "https://checkout.stripe.com/c/pay/cs_test_not-a-real-session" }; }, current: () => true };
  return { deps, payloads };
}
test("pricing starts the selected checkout directly and returns to the localized pricing page", async () => {
 const { deps, payloads } = fixture();
 const result = await preparePricingCheckout(intent, 1, urls, "invite-example", deps);
 assert.equal(result.kind, "checkout");
 assert.deepEqual(payloads, [{product_type:"subscription",plan:"pro",interval:"yearly",success_url:urls.success,cancel_url:urls.cancel,metadata:{referral_code:"invite-example"}}]);
});
test("existing subscriptions and lifetime access use management without creating another checkout",async()=>{
 for(const subscription of [{plan:"pro",status:"active"},{plan:"pro",status:"trialing"},{plan:"pro",status:"incomplete"},{plan:"lifetime",status:"active"}]) {
  const {deps,payloads}=fixture(subscription);
  assert.equal((await preparePricingCheckout(intent,1,urls,"",deps)).kind,"manage");
  assert.equal(payloads.length,0);
 }
});
test("disabled offers, stale sessions and failed account reads never start checkout",async()=>{
 const {deps,payloads}=fixture();
 assert.equal((await preparePricingCheckout(intent,1,urls,"",{...deps,capabilities:async()=>({...capabilities,billing:{...capabilities.billing,enabled:false}})})).kind,"unavailable");
 assert.equal((await preparePricingCheckout(intent,1,urls,"",{...deps,current:()=>false})).kind,"stale");
 await assert.rejects(preparePricingCheckout(intent,1,urls,"",{...deps,subscription:async()=>{throw new Error("unreachable");}}));
 assert.equal(payloads.length,0);
});
test("credit checkout validates quantity and does not require an absent recurring subscription",async()=>{
 const {deps,payloads}=fixture({plan:"pro",status:"active"});
 assert.equal((await preparePricingCheckout({product:"credits"},3,urls,"",deps)).kind,"checkout");
 assert.equal(payloads[0].quantity,3);
 assert.equal((await preparePricingCheckout({product:"credits"},0,urls,"",deps)).kind,"unavailable");
 assert.equal(payloads.length,1);
});
test("OAuth destinations retain the selected plan and language and reject invalid quantities",()=>{
 assert.equal(pricingPurchaseHref(intent,"zh"),"/zh/pricing?product=subscription&plan=pro&interval=yearly#plans");
 assert.deepEqual(readPricingPurchase("?product=credits&quantity=3"),{intent:{product:"credits"},quantity:3});
 assert.equal(readPricingPurchase("?product=credits&quantity=101"),null);
 assert.equal(readPricingPurchase("?product=credits&quantity=1&quantity=2"),null);
});
test("checkout failures never expose raw server text and display request ids only for server failures",()=>{
 const conflict = pricingCheckoutFailure(new ApiError("private database detail",409,409,null,"trace-conflict"));
 assert.ok(conflict.manage);assert.equal(conflict.requestId,"");assert.ok(!JSON.stringify(conflict).includes("private database"));
 assert.equal(pricingCheckoutFailure(new ApiError("private error",500,500,null,"trace-server")).requestId,"trace-server");
});
test("provider checkout destinations must be trusted Stripe URLs",async()=>{
 const {deps}=fixture();
 for(const checkout_url of ["https://evil.example/pay","javascript:alert(1)","http://checkout.stripe.com/pay","https://user:pass@checkout.stripe.com/pay"]) await assert.rejects(preparePricingCheckout(intent,1,urls,"",{...deps,create:async()=>({checkout_url})}));
});

test("checkout progress belongs only to the selected product and billing interval", async () => {
 const { pricingIntentMatches } = await import("../lib/pricing-checkout");
 assert.equal(pricingIntentMatches(null, intent), false);
 assert.equal(pricingIntentMatches(intent, intent), true);
 assert.equal(pricingIntentMatches(intent, {...intent, interval:"monthly"}), false);
 assert.equal(pricingIntentMatches(intent, {...intent, plan:"starter"}), false);
 assert.equal(pricingIntentMatches(intent, {product:"lifetime"}), false);
 assert.equal(pricingIntentMatches({product:"credits"}, {product:"credits"}), true);
 assert.equal(pricingIntentMatches({product:"lifetime"}, {product:"lifetime"}), true);
});
