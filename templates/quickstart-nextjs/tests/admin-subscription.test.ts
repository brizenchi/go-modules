import assert from "node:assert/strict";
import { test } from "node:test";
import { adminSubscriptionPresentation } from "../lib/admin-subscription";

const subscription = {
  plan: "premium", status: "active", billing_interval: "monthly", period_end: "2026-11-17T04:25:00Z",
  scheduled_change: { plan: "starter", billing_cycle: "monthly", effective_at: "2026-11-17T04:25:00Z" }
};

test("admin subscription distinguishes current plan and scheduled downgrade in both languages", () => {
  const view = adminSubscriptionPresentation(subscription);
  assert.equal(view?.plan.en, "Premium · Monthly");
  assert.equal(view?.upcoming?.plan.en, "Starter · Monthly");
  assert.equal(view?.upcoming?.plan.zh, "Starter · 月付");
  assert.equal(view?.upcoming?.effectiveAt, subscription.period_end);
  assert.equal(view?.endLabel.zh, "当前套餐保留至");
  assert.equal(view?.status.label.en, "Active");
});

test("admin shows cancellation and payment problems instead of a confirmed upcoming plan", () => {
  const canceling = adminSubscriptionPresentation({ ...subscription, cancel_at_period_end: true });
  assert.equal(canceling?.status.label.zh, "到期取消");
  assert.equal(canceling?.upcoming, null);
  assert.equal(canceling?.endLabel.en, "Access ends");
  const pending = adminSubscriptionPresentation({ ...subscription, payment_pending: true });
  assert.equal(pending?.status.label.en, "Payment pending");
  assert.equal(pending?.upcoming, null);
  const ended = adminSubscriptionPresentation({ ...subscription, status: "canceled" });
  assert.equal(ended?.upcoming, null);
  assert.equal(ended?.endLabel.en, "Period ended");
  assert.equal(adminSubscriptionPresentation(null), null);
});

test("ordinary renewal and lifetime avoid inventing a scheduled change or date", () => {
  assert.equal(adminSubscriptionPresentation({ ...subscription, scheduled_change: null })?.endLabel.en, "Next renewal");
  assert.equal(adminSubscriptionPresentation({ plan: "lifetime", status: "active" })?.periodEnd, null);
  assert.equal(adminSubscriptionPresentation({ ...subscription, scheduled_change: { plan: "starter", billing_cycle: "monthly" } })?.upcoming?.effectiveAt, null);
});
