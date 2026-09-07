import { integrationFields, integrationSecrets, type IntegrationPatch, type IntegrationSource, type IntegrationView } from "./integrations-api";

export type IntegrationDraft = {
  source: IntegrationSource;
  enabled: boolean;
  fields: Record<string, string>;
  secrets: Record<string, string>;
  clearSecrets: Record<string, boolean>;
};
export type IntegrationIssue = "reason" | "email" | "name" | "key" | "missing_key" | "mode" | "price" | "duplicate_price" | "missing_price" | "trial" | "credits";

export function integrationDraft(view: IntegrationView): IntegrationDraft {
  const fields = { ...view.fields };
  if (view.provider === "stripe") {
    fields.mode ||= "test";
    fields.trial_days ||= "0";
    fields.credits_per_package ||= "100";
  }
  return { source: view.source, enabled: view.enabled, fields, secrets: {}, clearSecrets: {} };
}

export function buildIntegrationPatch(view: IntegrationView, draft: IntegrationDraft, reason: string): IntegrationPatch {
  const base = { source: draft.source, version: view.version, reason: reason.trim() };
  if (draft.source === "environment") return base;
  const fields = Object.fromEntries(integrationFields[view.provider].map((key) => [key, (draft.fields[key] || "").trim()]));
  if (view.provider === "resend" && !draft.enabled) fields.email_auth_enabled = "false";
  const secrets: Record<string, string> = {};
  for (const key of integrationSecrets[view.provider]) {
    if (!draft.enabled && draft.clearSecrets[key]) secrets[key] = "";
    else if (draft.secrets[key]) secrets[key] = draft.secrets[key];
  }
  return { ...base, enabled: draft.enabled, fields, ...(Object.keys(secrets).length ? { secrets } : {}) };
}

const bytes = (value: string) => new TextEncoder().encode(value).length;
const priceKeys = ["starter_monthly", "starter_yearly", "pro_monthly", "pro_yearly", "premium_monthly", "premium_yearly", "lifetime"];
export function integrationIssues(view: IntegrationView, draft: IntegrationDraft, reason: string): IntegrationIssue[] {
  const issues = new Set<IntegrationIssue>();
  const reasonLength = Array.from(reason.trim()).length;
  if (reasonLength < 3 || reasonLength > 500) issues.add("reason");
  if (draft.source === "environment") return [...issues];
  const fields = draft.fields;
  const configured = (key: string) => Boolean(draft.secrets[key] || ((draft.enabled || !draft.clearSecrets[key]) && view.secrets[key]?.configured));
  for (const key of integrationSecrets[view.provider]) {
    const value = draft.secrets[key];
    if (!value || (!draft.enabled && draft.clearSecrets[key])) continue;
    const prefix = view.provider === "resend" ? /^re_[A-Za-z0-9_-]+$/ : key === "webhook_secret" ? /^whsec_[A-Za-z0-9]+$/ : new RegExp(`^(?:sk|rk)_${fields.mode}_[A-Za-z0-9]+$`);
    if (bytes(value) > 4096 || !prefix.test(value)) issues.add("key");
  }
  if (draft.enabled && integrationSecrets[view.provider].some((key) => !configured(key))) issues.add("missing_key");
  if (view.provider === "resend") {
    const email = (fields.sender_email || "").trim();
    if ((draft.enabled || email) && (!/^[^\s@<>]+@[^\s@<>]+\.[^\s@<>]+$/.test(email) || bytes(email) > 254)) issues.add("email");
    if (Array.from((fields.sender_name || "").trim()).length > 100) issues.add("name");
  } else {
    if (fields.mode !== "test" && fields.mode !== "live") issues.add("mode");
    const publishable = (fields.publishable_key || "").trim();
    if (publishable && !new RegExp(`^pk_${fields.mode}_[A-Za-z0-9]+$`).test(publishable)) issues.add("key");
    const creditPriceValue = (fields.credit_price_ids || "").trim();
    const creditPrices = creditPriceValue ? creditPriceValue.split(",").map((value) => value.trim()) : [];
    const prices = [...priceKeys.map((key) => (fields[key] || "").trim()).filter(Boolean), ...creditPrices];
    if (creditPrices.length > 20 || prices.some((value) => !/^price_[A-Za-z0-9]+$/.test(value) || bytes(value) > 2048)) issues.add("price");
    if (new Set(prices).size !== prices.length) issues.add("duplicate_price");
    if (draft.enabled && prices.length === 0) issues.add("missing_price");
    if (!/^\d+$/.test(fields.trial_days || "") || Number(fields.trial_days) > 730) issues.add("trial");
    if (!/^\d+$/.test(fields.credits_per_package || "") || Number(fields.credits_per_package) < 1 || Number(fields.credits_per_package) > 1000000) issues.add("credits");
  }
  return [...issues];
}
