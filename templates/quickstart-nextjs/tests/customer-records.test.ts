import assert from "node:assert/strict";
import { test } from "node:test";
import { customerTabs, creditSourceName, signedCredits } from "../lib/customer-records";

test("customer records have exactly three separate destinations", () => {
 assert.deepEqual(customerTabs, ["overview", "payments", "credits"]);
});
test("one ledger identifies admin, referral, purchase, allowance and campaign sources", () => {
 for (const [source, name] of [["admin_grant", "管理员调整"], ["admin_deduct", "管理员调整"], ["referral_signup_inviter", "邀请奖励"], ["referral_invitee", "邀请奖励"], ["stripe", "充值购买"], ["monthly", "每月发放"], ["subscription", "订阅周期发放"], ["campaign", "活动赠送"], ["signup", "注册赠送"]]) {
  assert.equal(creditSourceName({ source, kind: "grant" }).zh, name);
 }
 assert.equal(creditSourceName({ source: "custom-product", kind: "grant" }).en, "custom-product");
 assert.equal(creditSourceName({ source: "activity_reward", kind: "grant" }).zh, "活动赠送");
 assert.equal(creditSourceName({ source: "refund", kind: "refund", actor_id: "admin" }).zh, "积分退回");
 assert.equal(creditSourceName({ source: "expiry", kind: "expire" }).zh, "积分到期");
 assert.equal(signedCredits(1000), "+1,000");
 assert.equal(signedCredits(-50), "-50");
});
