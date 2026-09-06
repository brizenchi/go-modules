import assert from "node:assert/strict";
import test from "node:test";
import {
  buildDashboardChecklist,
  humanizePlan,
  profileCompletion
} from "../lib/dashboard";

test("profile completion reflects only real persisted profile fields", () => {
  assert.equal(profileCompletion(null), 0);
  assert.equal(profileCompletion({ email_verified: true, username: "", avatar_url: "" }), 34);
  assert.equal(profileCompletion({ email_verified: true, username: "Ada", avatar_url: "" }), 67);
  assert.equal(profileCompletion({ email_verified: true, username: "Ada", avatar_url: "https://example.com/a.png" }), 100);
});

test("plan names are readable without inventing product names", () => {
  assert.equal(humanizePlan("starter"), "Starter");
  assert.equal(humanizePlan("pro_yearly"), "Pro yearly");
  assert.equal(humanizePlan(""), "Free");
});

test("dashboard checklist omits disabled capabilities and uses real readiness", () => {
  const items = buildDashboardChecklist({
    profile: { email_verified: true, username: "Ada", avatar_url: "" },
    subscription: { plan: "pro", status: "active" },
    referralLink: "https://app.example.com/invite?ref=INV123",
    billingEnabled: true,
    referralEnabled: true
  });
  assert.deepEqual(items.map((item) => [item.key, item.complete]), [
    ["profile", true],
    ["billing", true],
    ["referral", true]
  ]);

  const accountOnly = buildDashboardChecklist({
    profile: { email_verified: false, username: "", avatar_url: "" },
    subscription: null,
    referralLink: "",
    billingEnabled: false,
    referralEnabled: false
  });
  assert.deepEqual(accountOnly.map((item) => item.key), ["profile"]);
  assert.equal(accountOnly[0]?.complete, false);
});
