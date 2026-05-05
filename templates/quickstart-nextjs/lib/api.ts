import { apiUrl } from "./env";
import type { AuthSession, SessionUser } from "./auth";

export type ApiEnvelope<T> = {
  code: number;
  msg?: string;
  data: T;
};

export type SendCodeResult = {
  email: string;
  expires_at: string;
  debug_code?: string;
};

export type VerifyResult = AuthSession;

export type WSTicketResult = {
  ticket: string;
  expires_at: string;
};

export type SubscriptionView = {
  plan: string;
  status: string;
  billing_cycle?: string;
  current_period_end?: string;
  cancel_at_period_end?: boolean;
  payment_method?: {
    brand: string;
    last4: string;
    exp_month: number;
    exp_year: number;
  } | null;
};

export type SubscriptionChangeMode =
  | "immediate_prorated"
  | "immediate_reset_cycle"
  | "period_end";

export type SubscriptionPreview = {
  currency: string;
  amount_due_now: number;
  current_period_end?: string;
  next_billing_at?: string;
  target_plan: string;
  target_interval: string;
  change_mode: SubscriptionChangeMode;
  immediate_charge: boolean;
  effective_at_period_end: boolean;
  message: string;
};

export type InvoiceItem = {
  id: string;
  amount_usd: number;
  status: string;
  period: string;
  pdfurl?: string;
  pdf_url?: string;
  created_at: string;
};

export type ReferralCodeResult = {
  code: string;
  link: string;
};

export type ReferralStats = {
  total_referred: number;
  activated: number;
  pending: number;
  total_reward_credits: number;
};

export type ReferralItem = {
  id: number;
  code: string;
  referrer_id: string;
  referee_id: string;
  status: string;
  activated_at?: string | null;
  expires_at?: string | null;
  reward_credits: number;
  created_at: string;
  updated_at: string;
};

export class ApiError extends Error {
  readonly status: number;
  readonly code: number;
  readonly data: unknown;

  constructor(message: string, status: number, code: number, data?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

type ApiRequestOptions = Omit<RequestInit, "body"> & {
  authToken?: string;
  body?: BodyInit | null;
  json?: unknown;
};

function buildHeaders(options: ApiRequestOptions): Headers {
  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");

  if (options.authToken) {
    headers.set("Authorization", `Bearer ${options.authToken}`);
  }

  if (options.json !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  return headers;
}

async function parseEnvelope<T>(response: Response): Promise<ApiEnvelope<T> | null> {
  const contentType = response.headers.get("content-type") || "";
  if (!contentType.includes("application/json")) {
    return null;
  }
  return (await response.json()) as ApiEnvelope<T>;
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const response = await fetch(apiUrl(path), {
    ...options,
    cache: "no-store",
    headers: buildHeaders(options),
    body: options.json !== undefined ? JSON.stringify(options.json) : options.body
  });

  const envelope = await parseEnvelope<T>(response);
  const message =
    envelope?.msg?.trim() ||
    (response.ok ? "request failed" : `http ${response.status}`);
  const bodyCode = typeof envelope?.code === "number" ? envelope.code : response.status;

  if (!response.ok || bodyCode !== 200) {
    throw new ApiError(message, response.status, bodyCode, envelope?.data);
  }

  return (envelope?.data ?? null) as T;
}

export async function sendCode(email: string): Promise<SendCodeResult> {
  return apiRequest<SendCodeResult>("/auth/send-code", {
    method: "POST",
    json: { email }
  });
}

export async function verifyCode(email: string, code: string, referralCode?: string): Promise<VerifyResult> {
  return apiRequest<VerifyResult>("/auth/verify-code", {
    method: "POST",
    json: {
      email,
      code,
      referral_code: referralCode?.trim() || undefined
    }
  });
}

export async function getGoogleAuthorizeURL(): Promise<string> {
  const data = await apiRequest<{ redirect_url: string }>("/auth/google/authorize");
  return data.redirect_url;
}

export async function exchangeToken(code: string, referralCode?: string): Promise<VerifyResult> {
  return apiRequest<VerifyResult>("/auth/exchange-token", {
    method: "POST",
    json: {
      code,
      referral_code: referralCode?.trim() || undefined
    }
  });
}

export async function refreshSession(token: string): Promise<VerifyResult> {
  return apiRequest<VerifyResult>("/auth/refresh", {
    method: "POST",
    authToken: token
  });
}

export async function logout(token: string): Promise<{ ok: boolean }> {
  return apiRequest<{ ok: boolean }>("/auth/logout", {
    method: "POST",
    authToken: token
  });
}

export async function issueWSTicket(token: string): Promise<WSTicketResult> {
  return apiRequest<WSTicketResult>("/websocket/ticket", {
    method: "POST",
    authToken: token,
    json: { scope: { source: "quickstart-nextjs" } }
  });
}

export type CheckoutPayload = {
  plan?: string;
  interval?: string;
  product_type: "subscription" | "credits" | "lifetime";
  price_id?: string;
  quantity?: number;
  success_url: string;
  cancel_url: string;
  metadata?: Record<string, string>;
};

export type CreateTopUpPaymentIntentPayload = {
  amount: number;
  metadata?: Record<string, string>;
};

export type TopUpPaymentIntentResult = {
  payment_intent_id: string;
  client_secret: string;
  amount_cents: number;
  amount_usd: number;
  currency: string;
  credits: number;
};

export async function createCheckoutSession(token: string, payload: CheckoutPayload): Promise<{ session_id: string; checkout_url: string }> {
  return apiRequest<{ session_id: string; checkout_url: string }>("/stripe/checkout/session", {
    method: "POST",
    authToken: token,
    json: payload
  });
}

export async function createTopUpPaymentIntent(
  token: string,
  payload: CreateTopUpPaymentIntentPayload
): Promise<TopUpPaymentIntentResult> {
  return apiRequest<TopUpPaymentIntentResult>("/stripe/topup/payment-intent", {
    method: "POST",
    authToken: token,
    json: payload
  });
}

export async function changeSubscription(
  token: string,
  payload: { plan: string; interval: string; change_mode?: SubscriptionChangeMode }
): Promise<{
  status: string;
  plan: string;
  billing_cycle: string;
  change_mode: SubscriptionChangeMode;
  provider_subscription_id: string;
  message: string;
}> {
  return apiRequest("/stripe/subscription/change", {
    method: "POST",
    authToken: token,
    json: payload
  });
}

export async function previewSubscriptionChange(
  token: string,
  payload: { plan: string; interval: string; change_mode?: SubscriptionChangeMode }
): Promise<SubscriptionPreview> {
  return apiRequest("/stripe/subscription/preview", {
    method: "POST",
    authToken: token,
    json: payload
  });
}

export async function createBillingPortalSession(
  token: string,
  returnURL: string
): Promise<{ url: string }> {
  return apiRequest("/stripe/portal/session", {
    method: "POST",
    authToken: token,
    json: { return_url: returnURL }
  });
}

export async function getSubscription(token: string): Promise<SubscriptionView> {
  const data = await apiRequest<{
    plan: string;
    status: string;
    billing_cycle?: string;
    current_period_end?: string;
    cancel_at_period_end?: boolean;
    payment_method?: {
      Brand?: string;
      Last4?: string;
      ExpMonth?: number;
      ExpYear?: number;
    } | null;
  }>("/stripe/subscription", {
    authToken: token
  });

  return {
    plan: data.plan,
    status: data.status,
    billing_cycle: data.billing_cycle,
    current_period_end: data.current_period_end,
    cancel_at_period_end: data.cancel_at_period_end,
    payment_method: data.payment_method
      ? {
          brand: data.payment_method.Brand || "",
          last4: data.payment_method.Last4 || "",
          exp_month: data.payment_method.ExpMonth || 0,
          exp_year: data.payment_method.ExpYear || 0
        }
      : null
  };
}

export async function listInvoices(token: string, page = 1, limit = 10): Promise<{ items: InvoiceItem[]; total: number; page: number; limit: number }> {
  const data = await apiRequest<{
    items: Array<{
      ID: string;
      AmountUSD: number;
      Status: string;
      Period: string;
      PDFURL?: string;
      CreatedAt: string;
    }>;
    total: number;
    page: number;
    limit: number;
  }>(
    `/stripe/invoices?page=${page}&limit=${limit}`,
    { authToken: token }
  );

  return {
    total: data.total,
    page: data.page,
    limit: data.limit,
    items: data.items.map((item) => ({
      id: item.ID,
      amount_usd: item.AmountUSD,
      status: item.Status,
      period: item.Period,
      pdf_url: item.PDFURL,
      created_at: item.CreatedAt
    }))
  };
}

export async function cancelSubscription(token: string, cancelType: "end_of_period" | "3days"): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/stripe/subscription/cancel", {
    method: "POST",
    authToken: token,
    json: { cancel_type: cancelType }
  });
}

export async function reactivateSubscription(token: string): Promise<{ message: string }> {
  return apiRequest<{ message: string }>("/stripe/subscription/reactivate", {
    method: "POST",
    authToken: token
  });
}

export async function getReferralCode(token: string): Promise<ReferralCodeResult> {
  return apiRequest<ReferralCodeResult>("/referral/code", {
    authToken: token
  });
}

export async function getReferralStats(token: string): Promise<ReferralStats> {
  const data = await apiRequest<{
    TotalReferred: number;
    Activated: number;
    Pending: number;
    TotalRewardCredits: number;
  }>("/referral/stats", {
    authToken: token
  });

  return {
    total_referred: data.TotalReferred,
    activated: data.Activated,
    pending: data.Pending,
    total_reward_credits: data.TotalRewardCredits
  };
}

export async function listReferrals(token: string, page = 1, limit = 20): Promise<{ items: ReferralItem[]; total: number; page: number; limit: number }> {
  const data = await apiRequest<{
    items: Array<{
      ID: number;
      Code: string;
      ReferrerID: string;
      RefereeID: string;
      Status: string;
      ActivatedAt?: string | null;
      ExpiresAt?: string | null;
      RewardCredits: number;
      CreatedAt: string;
      UpdatedAt: string;
    }>;
    total: number;
    page: number;
    limit: number;
  }>(
    `/referral/list?page=${page}&limit=${limit}`,
    { authToken: token }
  );

  return {
    total: data.total,
    page: data.page,
    limit: data.limit,
    items: data.items.map((item) => ({
      id: item.ID,
      code: item.Code,
      referrer_id: item.ReferrerID,
      referee_id: item.RefereeID,
      status: item.Status,
      activated_at: item.ActivatedAt,
      expires_at: item.ExpiresAt,
      reward_credits: item.RewardCredits,
      created_at: item.CreatedAt,
      updated_at: item.UpdatedAt
    }))
  };
}

export function userLabel(user?: SessionUser | null): string {
  if (!user) {
    return "anonymous";
  }
  return user.username?.trim() || user.email || user.id;
}
