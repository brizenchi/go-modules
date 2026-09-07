import { apiRequest, ApiError } from "./api";

export const integrationProviders = ["resend", "stripe"] as const;
export type IntegrationProvider = typeof integrationProviders[number];
export type IntegrationSource = "environment" | "database";
export const integrationFields: Record<IntegrationProvider, readonly string[]> = {
  resend: ["sender_email", "sender_name", "email_auth_enabled"],
  stripe: ["mode", "publishable_key", "starter_monthly", "starter_yearly", "pro_monthly", "pro_yearly", "premium_monthly", "premium_yearly", "lifetime", "credit_price_ids", "trial_days", "credits_per_package"]
};
export const integrationSecrets: Record<IntegrationProvider, readonly string[]> = {
  resend: ["api_key"], stripe: ["secret_key", "webhook_secret"]
};
export type IntegrationView = {
  provider: IntegrationProvider;
  source: IntegrationSource;
  version: number;
  enabled: boolean;
  fields: Record<string, string>;
  secrets: Record<string, { configured: boolean }>;
  active_enabled: boolean;
  active_source: IntegrationSource;
  restart_required: boolean;
};
export type IntegrationsView = { storage_ready: boolean; local_preview: boolean; providers: IntegrationView[] };
export type IntegrationPatch = {
  version: number;
  source: IntegrationSource;
  enabled?: boolean;
  fields?: Record<string, string>;
  secrets?: Record<string, string>;
  reason: string;
};
export type IntegrationFailure = "auth" | "forbidden" | "conflict" | "invalid" | "unavailable" | "storage";

// API responses can contain provider errors. Only this allowlisted category
// crosses into form state; neither provider messages nor request keys do.
export class IntegrationError extends Error {
  constructor(readonly kind: IntegrationFailure) { super(kind); this.name = "IntegrationError"; }
}
function safeFailure(error: unknown): IntegrationError {
  if (error instanceof IntegrationError) return error;
  if (error instanceof ApiError) {
    if (error.status === 401) return new IntegrationError("auth");
    if (error.status === 403) return new IntegrationError("forbidden");
    if (error.status === 409) return new IntegrationError("conflict");
    if (error.status === 400 || error.status === 422) return new IntegrationError("invalid");
    if (error.status === 503) return new IntegrationError("storage");
  }
  return new IntegrationError("unavailable");
}
function record(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new IntegrationError("unavailable");
  return value as Record<string, unknown>;
}
function source(value: unknown): IntegrationSource {
  if (value !== "environment" && value !== "database") throw new IntegrationError("unavailable");
  return value;
}
function providerView(value: unknown, expected?: IntegrationProvider): IntegrationView {
  const data = record(value);
  if (data.provider !== "resend" && data.provider !== "stripe") throw new IntegrationError("unavailable");
  const provider = data.provider;
  if (expected && provider !== expected) throw new IntegrationError("unavailable");
  if (typeof data.version !== "number" || !Number.isSafeInteger(data.version) || data.version < 0
    || typeof data.enabled !== "boolean" || typeof data.active_enabled !== "boolean" || typeof data.restart_required !== "boolean") throw new IntegrationError("unavailable");
  const fields = record(data.fields);
  const secrets = record(data.secrets);
  // Project the read DTO explicitly. Saved secret values are never used to
  // populate an input, even if a future server accidentally includes them.
  return {
    provider, source: source(data.source), version: data.version, enabled: data.enabled,
    fields: Object.fromEntries(integrationFields[provider].map((key) => [key, typeof fields[key] === "string" ? fields[key] : ""])),
    secrets: Object.fromEntries(integrationSecrets[provider].map((key) => [key, { configured: record(secrets[key]).configured === true }])),
    active_enabled: data.active_enabled, active_source: source(data.active_source), restart_required: data.restart_required
  };
}

export async function getAdminIntegrations(token: string, signal?: AbortSignal): Promise<IntegrationsView> {
  try {
    const data = record(await apiRequest<unknown>("/admin/integrations", { authToken: token, signal }));
    if (typeof data.storage_ready !== "boolean" || typeof data.local_preview !== "boolean" || !Array.isArray(data.providers)) throw new IntegrationError("unavailable");
    const providers = data.providers.map((value) => providerView(value));
    if (providers.length !== 2 || new Set(providers.map((item) => item.provider)).size !== 2) throw new IntegrationError("unavailable");
    return { storage_ready: data.storage_ready, local_preview: data.local_preview, providers };
  } catch (error) { throw safeFailure(error); }
}

export async function saveAdminIntegration(token: string, provider: IntegrationProvider, patch: IntegrationPatch, key: string, signal?: AbortSignal): Promise<IntegrationView> {
  try {
    const result = await apiRequest<unknown>(`/admin/integrations/${provider}`, {
      method: "PATCH", authToken: token, signal, headers: { "Idempotency-Key": key }, json: patch
    });
    return providerView(result, provider);
  } catch (error) { throw safeFailure(error); }
}
