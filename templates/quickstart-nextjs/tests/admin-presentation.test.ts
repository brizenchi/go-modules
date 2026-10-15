import assert from "node:assert/strict";
import { test } from "node:test";
import { adminTag } from "../lib/admin-presentation";

test("pending payment, referral and audit statuses retain their own meaning", () => {
  assert.equal(adminTag("orders", "status", "pending")?.label.zh, "待支付");
  assert.equal(adminTag("referrals", "status", "pending")?.label.zh, "待激活");
  assert.equal(adminTag("audit", "status", "pending")?.label.zh, "处理中");
  assert.equal(adminTag("users", "reason", "pending"), null);
  assert.equal(adminTag("audit", "reason", "succeeded"), null);
});

test("known states use semantic tones and unknown codes are not misrepresented", () => {
  assert.equal(adminTag("subscriptions", "status", "active")?.tone, "success");
  assert.equal(adminTag("subscriptions", "status", "canceling")?.tone, "warning");
  assert.equal(adminTag("orders", "status", "failed")?.tone, "danger");
  assert.equal(adminTag("users", "role", "admin")?.label.zh, "管理员");
  assert.equal(adminTag("users", "plan", "pro")?.label.zh, "Pro");
  assert.equal(adminTag("subscriptions", "plan", "lifetime")?.label.zh, "Lifetime");
  assert.equal(adminTag("orders", "livemode", false)?.label.zh, "测试");
  assert.equal(adminTag("orders", "livemode", null)?.label.zh, "未知");
  assert.deepEqual(adminTag("orders", "status", "future_status"), { label: { en: "future_status", zh: "future_status" }, tone: "neutral" });
  assert.equal(adminTag("orders", "status", null), null);
});


test("unrecognized enum values cannot resolve inherited object properties", () => {
  for (const value of ["constructor", "__proto__", "toString"]) {
    assert.deepEqual(adminTag("subscriptions", "status", value), { label: { en: value, zh: value }, tone: "neutral" });
  }
});
