function normalizeOrigin(value: string | undefined, fallback: string): string {
  const raw = value?.trim() || fallback;
  return raw.replace(/\/+$/, "");
}

function normalizePath(value: string | undefined, fallback: string): string {
  const raw = value?.trim() || fallback;
  return raw.startsWith("/") ? raw : `/${raw}`;
}

function parsePositiveInt(value: string | undefined, fallback: number): number {
  const parsed = Number.parseInt((value || "").trim(), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function parsePositiveFloat(value: string | undefined, fallback: number): number {
  const parsed = Number.parseFloat((value || "").trim());
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

export const appEnv = {
  appName: (process.env.NEXT_PUBLIC_APP_NAME || "Clawmesh Quickstart Frontend").trim(),
  appUrl: normalizeOrigin(process.env.NEXT_PUBLIC_APP_URL, "http://localhost:3000"),
  apiBaseUrl: normalizeOrigin(process.env.NEXT_PUBLIC_API_BASE_URL, "http://localhost:8080/api/v1"),
  stripePublishableKey: (process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY || "").trim(),
  defaultPlan: (process.env.NEXT_PUBLIC_DEFAULT_PLAN || "pro").trim(),
  defaultInterval: (process.env.NEXT_PUBLIC_DEFAULT_INTERVAL || "monthly").trim(),
  defaultCreditsQuantity: parsePositiveInt(process.env.NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY, 1),
  defaultTopUpAmountUSD: parsePositiveFloat(process.env.NEXT_PUBLIC_DEFAULT_TOPUP_AMOUNT_USD, 25),
  creditsPriceId: (process.env.NEXT_PUBLIC_CREDITS_PRICE_ID || "").trim(),
  stripeSuccessPath: normalizePath(process.env.NEXT_PUBLIC_STRIPE_SUCCESS_PATH, "/billing?checkout=success"),
  stripeCancelPath: normalizePath(process.env.NEXT_PUBLIC_STRIPE_CANCEL_PATH, "/billing?checkout=cancelled")
};

export function apiUrl(path: string): string {
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return `${appEnv.apiBaseUrl}${suffix}`;
}

export function appUrl(path: string): string {
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return `${appEnv.appUrl}${suffix}`;
}
