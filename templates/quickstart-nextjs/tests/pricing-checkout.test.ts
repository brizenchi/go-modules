import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiError, type CapabilitiesView, type CheckoutPayload, type SubscriptionView } from "../lib/api";
import { preparePricingCheckout, pricingPurchaseHref, readPricingPurchase, pricingCheckoutFailure, pricingCheckoutResultHref, pricingCheckoutURLs } from "../lib/pricing-checkout";
import { checkoutReturnStatus } from "../lib/billing-state";

const capabilities: CapabilitiesView = { account: { enabled: true }, referral: { enabled: true }, billing: { enabled: true, provider: "stripe", offers: { subscriptions: [{ plan: "pro", intervals: ["monthly", "yearly"] }], lifetime: true, credits: true } } };
const intent = { product: "subscription", plan: "pro", interval: "yearly" } as const;
const urls = { success: "https://example.test/zh/pricing?checkout=success", cancel: "https://example.test/zh/pricing?checkout=cancelled" };

test("credit purchases use the private credit route and keep the language on pricing cancellation", () => {
 assert.deepEqual(pricingCheckoutURLs({product:"credits"}, "zh", 3), {
  success:"/credits?checkout=success&session_id={CHECKOUT_SESSION_ID}",
  cancel:"/zh/pricing?checkout=cancelled&product=credits&quantity=3#one-time"
 });
 assert.equal(pricingCheckoutURLs(intent,"en").success,"/billing?checkout=success");
 assert.equal(pricingCheckoutResultHref("success", "credits", "cs_example"), "/credits?checkout=success&session_id=cs_example");
});

test("the pricing purchase-result link starts billing confirmation after checkout", () => {
 const destination = new URL(pricingCheckoutResultHref("success"), "https://example.test");
 assert.equal(destination.pathname, "/billing");
 assert.equal(checkoutReturnStatus(destination.search), "success");
 for (const status of [null, "cancelled"] as const) {
  assert.equal(checkoutReturnStatus(new URL(pricingCheckoutResultHref(status), "https://example.test").search), null);
 }
});
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
test("existing paid subscriptions review a plan change without creating another checkout",async()=>{
 for (const status of ["active", "trialing"]) {
  const { deps, payloads } = fixture({ plan: "pro", status, billing_cycle: "monthly" });
  const result = await preparePricingCheckout(intent, 1, urls, "", deps);
  assert.equal(result.kind, "change");
  if (result.kind === "change") assert.deepEqual(result.selection, { plan: "pro", interval: "yearly" });
  assert.equal(payloads.length, 0);
 }
 const { deps, payloads } = fixture({ plan: "pro", status: "active", billing_cycle: "yearly" });
 assert.equal((await preparePricingCheckout(intent, 1, urls, "", deps)).kind, "current");
 assert.equal(payloads.length, 0);
});
test("same tier interval changes preserve the selected interval even with a scheduled downgrade", async () => {
 for (const billing_cycle of ["monthly", "yearly"]) {
  const subscription = {plan:"pro",status:"active",billing_cycle,scheduled_change:{plan:"starter",billing_cycle:"monthly"}};
  const {deps,payloads}=fixture(subscription);
  const result = await preparePricingCheckout({...intent,interval:billing_cycle==="monthly"?"yearly":"monthly"},1,urls,"",deps);
  assert.equal(result.kind,"change");
  if (result.kind === "change") assert.deepEqual(result.selection,{plan:"pro",interval:billing_cycle==="monthly"?"yearly":"monthly"});
  assert.equal(payloads.length,0);
 }
});
test("existing subscription restrictions do not prevent choosing a target in the change dialog",async()=>{
 for(const subscription of [
  {plan:"pro",status:"incomplete"},
  {plan:"pro",status:"canceling",billing_cycle:"monthly",cancel_at_period_end:true},
  {plan:"pro",status:"active",billing_cycle:"monthly",pending_payment:{amount_due:100,currency:"usd"}},
  {plan:"pro",status:"active",billing_cycle:"monthly",scheduled_change:{plan:"starter",billing_cycle:"monthly"}}
 ]) {
  const {deps,payloads}=fixture(subscription);
  const result = await preparePricingCheckout(intent,1,urls,"",deps);
  assert.equal(result.kind,"change");
  if (result.kind === "change") assert.deepEqual(result.subscription, subscription);
  assert.equal(payloads.length,0);
 }
 const lifetime=fixture({plan:"lifetime",status:"active"});
 assert.equal((await preparePricingCheckout(intent,1,urls,"",lifetime.deps)).kind,"manage");
 assert.equal(lifetime.payloads.length,0);
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
test("Pro can buy Lifetime through a full one-time Checkout without changing its current subscription", async () => {
 const {deps,payloads}=fixture({plan:"pro",status:"active",billing_cycle:"monthly"});
 assert.equal((await preparePricingCheckout({product:"lifetime"},1,urls,"",deps)).kind,"checkout");
 assert.deepEqual(payloads,[{product_type:"lifetime",success_url:urls.success,cancel_url:urls.cancel}]);
 const owned=fixture({plan:"lifetime",status:"active"});
 assert.equal((await preparePricingCheckout({product:"lifetime"},1,urls,"",owned.deps)).kind,"manage");
 assert.equal(owned.payloads.length,0);
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
test("invalid catalog prices explain purchase unavailability without suggesting retries",()=>{
 const error = new ApiError("private provider detail",503,503,{reason:"billing_price_configuration"},"fixture-request");
 const failure = pricingCheckoutFailure(error);
 assert.match(failure.messages.en,/contact support/i);
 assert.doesNotMatch(failure.messages.en,/try again|retry|recurring|Stripe/i);
 assert.equal(failure.requestId,"fixture-request");
 assert.ok(!JSON.stringify(failure).includes("private provider detail"));
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

test("Premium monthly opens the Premium annual confirmation without a new subscription", async () => {
 for (const scheduled_change of [undefined, {plan:"starter",billing_cycle:"monthly"}]) {
  const {deps,payloads}=fixture({plan:"premium",status:"active",billing_cycle:"monthly",scheduled_change});
  deps.capabilities=async()=>({...capabilities,billing:{...capabilities.billing,offers:{...capabilities.billing.offers,subscriptions:[{plan:"premium",intervals:["monthly","yearly"]}]}}});
  const result=await preparePricingCheckout({product:"subscription",plan:"premium",interval:"yearly"},1,urls,"",deps);
  assert.equal(result.kind,"change");
  if(result.kind==="change") assert.deepEqual(result.selection,{plan:"premium",interval:"yearly"});
  assert.equal(payloads.length,0);
 }
});
