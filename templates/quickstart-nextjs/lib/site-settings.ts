import { apiUrl, appEnv } from "./env";

export type PublicSiteSettings = {
  brand_name: string;
  description: string;
  support_email: string;
  support_url: string;
  export_credit_cost: number;
};

export const SITE_SETTINGS_EVENT = "go-modules.quickstart-nextjs.site-settings-change";

export function safeSupportEmail(value: unknown): string {
  if (typeof value !== "string") return "";
  const trimmed = value.trim();
  return trimmed.length <= 254 && /^[^\s@?&#%]+@[^\s@?&#%]+\.[^\s@?&#%]+$/u.test(trimmed) ? trimmed : "";
}

export function safeSupportURL(value: unknown): string {
  if (typeof value !== "string" || !value.trim()) return "";
  try {
    const url = new URL(value.trim());
    return url.protocol === "https:" && !url.username && !url.password ? url.toString() : "";
  } catch { return ""; }
}

export const publicSiteSettingsFallback: PublicSiteSettings = {
  brand_name: appEnv.appName,
  description: "Launch your SaaS with authentication, billing and referrals.",
  support_email: safeSupportEmail(process.env.NEXT_PUBLIC_SUPPORT_EMAIL),
  support_url: safeSupportURL(process.env.NEXT_PUBLIC_SUPPORT_URL),
  export_credit_cost: 1
};

export function normalizePublicSiteSettings(value: unknown): PublicSiteSettings {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Invalid site settings response.");
  const data = value as Record<string, unknown>;
  return {
    brand_name: typeof data.brand_name === "string" && data.brand_name.trim() ? data.brand_name.trim() : publicSiteSettingsFallback.brand_name,
    description: typeof data.description === "string" ? data.description.trim() : publicSiteSettingsFallback.description,
    support_email: safeSupportEmail(data.support_email),
    support_url: safeSupportURL(data.support_url),
    export_credit_cost: typeof data.export_credit_cost === "number" && Number.isSafeInteger(data.export_credit_cost) && data.export_credit_cost >= 1 && data.export_credit_cost <= 1000000 ? data.export_credit_cost : 1
  };
}

export async function getPublicSiteSettings(signal?: AbortSignal): Promise<PublicSiteSettings> {
  const response = await fetch(apiUrl("/site/settings"), { signal, cache: "no-store" });
  if (!response.ok) throw new Error("Site settings are temporarily unavailable.");
  const envelope: unknown = await response.json();
  if (!envelope || typeof envelope !== "object" || !("code" in envelope) || envelope.code !== 200 || !("data" in envelope)) {
    throw new Error("Invalid site settings response.");
  }
  return normalizePublicSiteSettings(envelope.data);
}
