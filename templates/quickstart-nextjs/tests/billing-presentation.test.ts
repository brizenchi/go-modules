import assert from "node:assert/strict";
import { test } from "node:test";
import type { SubscriptionPreview, SubscriptionView } from "../lib/api";
import { billingPrimaryAction, billingPrimaryText, billingText, lifetimePurchaseAction, planActionText, subscriptionRestrictionText, subscriptionStatusText, changeDialogTitle, changeConfirmationText, changeSubmitText, changeTimingText, changePaymentText, changeResultText, billingFailureText } from "../lib/billing-presentation";
import { subscriptionChangeBlocker, subscriptionPlanIntent } from "../lib/subscription-change";

const pro: SubscriptionView = { plan: "pro", status: "active", billing_cycle: "monthly" };
const preview: SubscriptionPreview = { currency: "usd", amount_due_now: 45, target_plan: "premium", target_interval: "monthly", change_mode: "immediate_prorated", immediate_charge: true, effective_at_period_end: false, message: "" };

test("billing primary actions match free, ended, active, restricted and lifetime accounts", () => {
  const cases: [string | undefined, SubscriptionView | null, ReturnType<typeof billingPrimaryAction>][] = [
    [undefined, null, "loading"],
    ["free", { plan: "free", status: "inactive" }, "upgrade"],
    ["free", { ...pro, status: "canceled" }, "resubscribe"],
    ["free", { ...pro, status: "incomplete_expired" }, "resubscribe"],
    ["pro", pro, "change"],
    ["premium", { ...pro, plan: "premium", status: "trialing" }, "change"],
    ["pro", { ...pro, status: "canceling" }, "resume"],
    ["pro", { ...pro, cancel_at_period_end: true }, "resume"],
    ["pro", { ...pro, effective_cancel_at: "2026-11-01T00:00:00Z" }, "resume"],
    ["pro", { ...pro, status: "past_due" }, "payment"],
    ["pro", { ...pro, pending_payment: { amount_due: 100, currency: "usd" } }, "payment"],
    ["pro", { ...pro, scheduled_change: { plan: "starter", billing_cycle: "monthly" } }, "scheduled"],
    ["pro", { ...pro, status: "paused" }, "review"],
    ["lifetime", null, "lifetime"],
    ["lifetime", { plan: "free", status: "inactive" }, "lifetime"]
  ];
  for (const [membership, subscription, expected] of cases) {
    assert.equal(billingPrimaryAction(membership, subscription), expected, JSON.stringify(subscription));
    assert.ok(billingPrimaryText(expected).en);
    assert.ok(billingPrimaryText(expected).zh);
  }
});

test("pricing distinguishes resubscription from renewal recovery and exact billing interval", () => {
  assert.equal(planActionText("subscribe", "Pro", "monthly", true).en, "Resubscribe to Pro");
  assert.equal(planActionText("subscribe", "Pro", "monthly").zh, "订阅 Pro");
  assert.equal(subscriptionPlanIntent({ plan: "pro", interval: "monthly" }, pro), "current");
  assert.equal(planActionText(subscriptionPlanIntent({ plan: "pro", interval: "yearly" }, pro), "Pro", "yearly").en, "Switch to yearly");
  for (const current of [pro, { ...pro, status: "canceling" }, { ...pro, pending_payment: { amount_due: 100, currency: "usd" } }]) {
    assert.equal(planActionText(subscriptionPlanIntent({ plan: "starter", interval: "monthly" }, current), "Starter", "monthly").en, "Downgrade to Starter");
  }
});

test("each restriction gives a localized next step, without claiming payment success", () => {
  for (const blocker of ["payment", "scheduled", "cancellation", "unavailable"] as const) {
    const copy = subscriptionRestrictionText(blocker);
    assert.ok(copy?.en && copy.zh);
  }
  assert.equal(subscriptionRestrictionText(null), null);
  assert.equal(subscriptionChangeBlocker({ ...pro, status: "past_due", cancel_at_period_end: true }), "payment");
  assert.equal(subscriptionStatusText("internal_enum").en, "Needs attention");
  assert.ok(!billingText.checkoutChecking.en.includes("completed"));
  assert.ok(!billingText.checkoutWaiting.en.includes("successful"));
});

test("payment confirmation names the immediate amount or a future scheduled change", () => {
  assert.equal(changeDialogTitle("downgrade").en, "Downgrade plan");
  assert.equal(changeDialogTitle("interval").zh, "更改计费周期");
  assert.equal(changeConfirmationText(null, "").en, "Review the price first");
  assert.equal(changeConfirmationText(preview, "$45.00").en, "Confirm and pay $45.00");
  const scheduled = { ...preview, change_mode: "period_end" as const, amount_due_now: 0 };
  assert.equal(changeConfirmationText(scheduled, "$0.00").zh, "确认下期变更");
  assert.equal(changeTimingText(scheduled).label.en, "Next billing cycle");
  assert.ok(!changeTimingText({ ...preview, amount_due_now: 0 }).label.en.includes("payment"));
  assert.equal(changeConfirmationText({ ...preview, amount_due_now: 0 }, "$0.00").en, "Confirm change");
});

test("disabled dialog actions explain current selection, billing restrictions, loading and errors", () => {
  const state = { choice: "upgrade" as const, configured: true, phase: "loading" as const, preview: null, amount: "" };
  assert.equal(changeSubmitText(state).en, "Loading price…");
  assert.equal(changeSubmitText({ ...state, choice: "current" }).en, "Current plan");
  assert.equal(changeSubmitText({ ...state, choice: "manage" }).zh, "请先处理当前订阅");
  assert.equal(changeSubmitText({ ...state, configured: false }).en, "Choose an available plan");
  assert.equal(changeSubmitText({ ...state, phase: "error" }).en, "Price unavailable");
  assert.equal(changeSubmitText({ ...state, phase: "ready" }).en, "Price unavailable");
  assert.equal(changeSubmitText({ ...state, phase: "ready", preview, amount: "$45.00" }).en, "Confirm and pay $45.00");
});

test("immediate paid changes disclose saved-method collection without claiming payment success", () => {
  for (const change_mode of ["immediate_prorated", "immediate_reset_cycle"] as const) {
    const copy = changePaymentText({ ...preview, change_mode });
    assert.match(copy?.en || "", /saved payment method/);
    assert.match(copy?.zh || "", /已保存的付款方式/);
    assert.match(copy?.en || "", /verification/);
    assert.doesNotMatch(copy?.en || "", /paid|successful|completed/i);
  }
  assert.equal(changePaymentText({ ...preview, change_mode: "period_end" }), null);
  assert.equal(changePaymentText({ ...preview, amount_due_now: 0 }), null);
});

test("billing failures explain user actions instead of infrastructure details", () => {
  assert.equal(billingFailureText("auth").title.zh, "请重新登录");
  assert.equal(billingFailureText("network").title.en, "Connection interrupted");
  assert.equal(billingFailureText("unknown", "subscription_change_pending").title.zh, "订阅状态已变化");
  for (const kind of ["configuration", "disabled", "unavailable", "unknown"] as const) {
    const copy = billingFailureText(kind);
    assert.ok(!/API|redeploy|backend|credential|secret/i.test(JSON.stringify(copy)));
  }
});

test("Lifetime protects existing access and prevents a conflicting recurring purchase", () => {
  assert.equal(lifetimePurchaseAction({ plan: "lifetime", status: "active" }), "owned");
  for (const status of ["active", "trialing", "canceling", "past_due", "incomplete"]) {
    assert.equal(lifetimePurchaseAction({ ...pro, status }), "blocked", status);
  }
  for (const status of ["canceled", "expired", "ended", "incomplete_expired"]) {
    assert.equal(lifetimePurchaseAction({ ...pro, status }), "buy", status);
  }
  assert.equal(lifetimePurchaseAction({ plan: "free", status: "inactive" }), "buy");
});

test("pending payment, scheduled changes and an applied change have distinct outcomes", () => {
  const payment = changeResultText({ pending_payment: { amount_due: 4500, currency: "usd" } });
  assert.equal(payment.state, "payment");
  assert.equal(payment.title.en, "Payment required");
  assert.ok(!payment.body.zh.includes("已生效"));
  const scheduled = changeResultText({ scheduled_change: { plan: "starter", billing_cycle: "monthly" } });
  assert.equal(scheduled.state, "scheduled");
  assert.ok(scheduled.body.en.includes("current plan continues"));
  assert.equal(changeResultText({}).state, "updated");
});
