import assert from "node:assert/strict";
import { test } from "node:test";
import { buildIntegrationPatch, integrationDraft, integrationIssues } from "../lib/integration-form";
import type { IntegrationView } from "../lib/integrations-api";

function view(provider: "resend" | "stripe"): IntegrationView {
  return {
    provider, source: "database", version: 7, enabled: true,
    fields: provider === "resend" ? { sender_email: "hello@example.test", sender_name: "Demo", email_auth_enabled: "true" } : { mode: "test", trial_days: "0", credits_per_package: "100", starter_monthly: "price_monthly" },
    secrets: provider === "resend" ? { api_key: { configured: true } } : { secret_key: { configured: true }, webhook_secret: { configured: true } },
    active_enabled: false, active_source: "environment", restart_required: true
  };
}

test("empty key inputs keep configured values, replacements are explicit, and draft construction never copies read secrets", () => {
  const saved = view("resend");
  const draft = integrationDraft(saved);
  assert.deepEqual(draft.secrets, {});
  assert.equal(buildIntegrationPatch(saved, draft, " change sender ").secrets, undefined);
  assert.equal(buildIntegrationPatch(saved, draft, " change sender ").version, 7);
  draft.secrets.api_key = "re_Replacement123";
  assert.deepEqual(buildIntegrationPatch(saved, draft, "change key").secrets, { api_key: "re_Replacement123" });
  assert.deepEqual(integrationIssues(saved, draft, "change key"), []);
});

test("environment fallback sends only source, version and reason even when a draft contains entered secrets", () => {
  const saved = view("stripe");
  const draft = integrationDraft(saved);
  draft.source = "environment";
  draft.secrets.secret_key = "sk_test_discard";
  draft.fields.credits_per_package = "invalid hidden field";
  assert.deepEqual(buildIntegrationPatch(saved, draft, " reset to env "), { source: "environment", version: 7, reason: "reset to env" });
  assert.deepEqual(integrationIssues(saved, draft, "reset to env"), []);
});

test("clearing credentials requires an explicit disabled-service change; disabling Resend also disables email-code sign-in", () => {
  const saved = view("resend");
  const draft = integrationDraft(saved);
  draft.clearSecrets.api_key = true;
  assert.equal(buildIntegrationPatch(saved, draft, "leave enabled").secrets, undefined);
  assert.deepEqual(integrationIssues(saved, draft, "leave enabled"), []);
  draft.enabled = false;
  const patch = buildIntegrationPatch(saved, draft, "retire service");
  assert.deepEqual(patch.secrets, { api_key: "" });
  assert.equal(patch.fields?.email_auth_enabled, "false");
});

test("enabling services requires credentials and relevant sender/price fields; disabled drafts can be incomplete", () => {
  for (const provider of ["resend", "stripe"] as const) {
    const saved = view(provider);
    saved.secrets = Object.fromEntries(Object.keys(saved.secrets).map((key) => [key, { configured: false }]));
    const draft = integrationDraft(saved);
    if (provider === "resend") draft.fields.sender_email = "";
    else draft.fields.starter_monthly = "";
    assert.ok(integrationIssues(saved, draft, "enable provider").includes("missing_key"));
    assert.ok(integrationIssues(saved, draft, "enable provider").includes(provider === "resend" ? "email" : "missing_price"));
    draft.enabled = false;
    assert.deepEqual(integrationIssues(saved, draft, "save draft"), []);
  }
});

test("Stripe validation catches mixed modes, duplicate offers, invalid prices and integer limits", () => {
  const saved = view("stripe");
  const draft = integrationDraft(saved);
  draft.fields.mode = "live";
  draft.secrets.secret_key = "sk_test_WrongMode";
  draft.fields.publishable_key = "pk_test_WrongMode";
  draft.fields.pro_yearly = "price_monthly";
  draft.fields.credit_price_ids = "invalid, price_monthly";
  draft.fields.trial_days = "731";
  draft.fields.credits_per_package = "1.5";
  const issues = integrationIssues(saved, draft, "configure payments");
  for (const issue of ["key", "duplicate_price", "price", "trial", "credits"]) assert.ok(issues.includes(issue as typeof issues[number]));
  draft.secrets.secret_key = "rk_live_Correct123";
  draft.fields.publishable_key = "pk_live_Correct123";
  draft.fields.pro_yearly = "price_proyearly";
  draft.fields.credit_price_ids = " price_credita , price_creditb ";
  draft.fields.trial_days = "730";
  draft.fields.credits_per_package = "1000000";
  assert.deepEqual(integrationIssues(saved, draft, "configure payments"), []);
});

test("service validation counts reason and sender name in characters and rejects credential whitespace", () => {
  const saved = view("resend");
  const draft = integrationDraft(saved);
  draft.secrets.api_key = "re_Credential\n";
  draft.fields.sender_name = "名".repeat(101);
  const issues = integrationIssues(saved, draft, "更改");
  assert.ok(issues.includes("reason"));
  assert.ok(issues.includes("name"));
  assert.ok(issues.includes("key"));
});

test("Stripe credit price lists reject empty entries instead of sending values the backend rejects", () => {
  const saved = view("stripe");
  const draft = integrationDraft(saved);
  for (const prices of ["price_A,", "price_A,,price_B", ",price_A", ","]) {
    draft.fields.credit_price_ids = prices;
    assert.ok(integrationIssues(saved, draft, "edit packages").includes("price"));
  }
  draft.fields.credit_price_ids = " ";
  assert.deepEqual(integrationIssues(saved, draft, "edit packages"), []);
});
