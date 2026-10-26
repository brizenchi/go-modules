import { apiUrl } from "./env";
import { clearSessionIfToken, type AuthSession, type SessionUser } from "./auth";
import { prepareOAuthBrowserFlow } from "./oauth-flow";
import type { ContentLocale } from "./locale";

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

export type CapabilitiesView = {
  // Optional only for compatibility with an older quickstart API that exposed
  // account/billing/referral capabilities before auth method discovery.
  auth?: {
    email_enabled: boolean;
    oauth_providers: OAuthProvider[];
    admin_password_enabled?: boolean;
  };
  account: { enabled: boolean };
  billing: {
    enabled: boolean;
    provider: string;
    offers: {
      subscriptions: Array<{ plan: string; intervals: string[] }>;
      lifetime: boolean;
      credits: boolean;
    };
  };
  referral: { enabled: boolean };
};

export type AccountProfile = {
  id: string;
  email: string;
  email_verified: boolean;
  username: string;
  avatar_url: string;
  role: string;
  credits: number;
  created_at: string;
  updated_at: string;
};

export type UpdateAccountProfilePayload = {
  username?: string;
  avatar_url?: string;
};

export type SubscriptionView = {
  plan: string;
  status: string;
  billing_cycle?: string;
  current_period_end?: string;
  cancel_at_period_end?: boolean;
  effective_cancel_at?: string;
  scheduled_change?: { plan: string; billing_cycle: string; effective_at?: string } | null;
  pending_payment?: { expires_at?: string; invoice_url?: string; amount_due: number; currency: string } | null;
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
  /** Major currency units from the existing preview API, unlike payment-record amounts. */
  amount_due_now: number;
  /** Unused-time credit in minor currency units; null/absent when unavailable. */
  unused_credit_amount?: number | null;
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
  /** X-Request-ID of the failed response; show it so users can report the error. */
  readonly requestId: string;

  constructor(message: string, status: number, code: number, data?: unknown, requestId = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.data = data;
    this.requestId = requestId;
  }

  /** Stable machine-readable error reason from `data.reason`, or "" (see docs/standards/API_STANDARD.md). */
  get reason(): string {
    const reason = (this.data as { reason?: unknown } | null | undefined)?.reason;
    return typeof reason === "string" ? reason : "";
  }
}

type ApiRequestOptions = Omit<RequestInit, "body"> & {
  authToken?: string;
  next?: { revalidate: number };
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
    cache: options.authToken ? "no-store" : options.cache || "no-store",
    headers: buildHeaders(options),
    body: options.json !== undefined ? JSON.stringify(options.json) : options.body
  });

  const envelope = await parseEnvelope<T>(response);
  const message =
    envelope?.msg?.trim() ||
    (response.ok ? "request failed" : `http ${response.status}`);
  const bodyCode = typeof envelope?.code === "number" ? envelope.code : response.status;

  if (!response.ok || bodyCode !== 200) {
    const error = new ApiError(message, response.status, bodyCode, envelope?.data, response.headers.get("x-request-id") || "");
    if ((response.status === 401 || bodyCode === 401) && options.authToken) {
      clearSessionIfToken(options.authToken);
    }
    throw error;
  }

  return (envelope?.data ?? null) as T;
}

export async function getCapabilities(signal?: AbortSignal): Promise<CapabilitiesView> {
  return apiRequest<CapabilitiesView>("/capabilities", { signal });
}

export async function loginAdmin(email: string, password: string, signal?: AbortSignal): Promise<AuthSession> {
  return apiRequest<AuthSession>("/auth/admin/login", {
    method: "POST",
    json: { email: email.trim(), password },
    signal
  });
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

export type OAuthProvider = "google" | "github";

export async function getOAuthAuthorizeURL(provider: OAuthProvider, returnTo?: string): Promise<string> {
	const challenge = await prepareOAuthBrowserFlow(provider, returnTo);
  const data = await apiRequest<{ redirect_url: string }>(
		`/auth/${provider}/authorize?challenge=${encodeURIComponent(challenge)}`,
		{ credentials: "include" }
	);
  return data.redirect_url;
}

export async function exchangeToken(code: string, referralCode?: string, oauthVerifier?: string): Promise<VerifyResult> {
  return apiRequest<VerifyResult>("/auth/exchange-token", {
    method: "POST",
    json: {
      code,
		oauth_verifier: oauthVerifier?.trim() || undefined,
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
  try {
    const result = await apiRequest<{ ok: boolean }>("/auth/logout", {
      method: "POST",
      authToken: token
    });
    if (!result.ok) throw new Error("Sign-out was not confirmed. Please try again.");
    clearSessionIfToken(token);
    return result;
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      clearSessionIfToken(token);
      return { ok: true };
    }
    throw error;
  }
}

export async function issueWSTicket(token: string): Promise<WSTicketResult> {
  return apiRequest<WSTicketResult>("/websocket/ticket", {
    method: "POST",
    authToken: token,
    json: { scope: { source: "quickstart-nextjs" } }
  });
}

export async function getAccountProfile(token: string): Promise<AccountProfile> {
  return apiRequest<AccountProfile>("/account/profile", {
    authToken: token
  });
}

export async function updateAccountProfile(
  token: string,
  payload: UpdateAccountProfilePayload
): Promise<AccountProfile> {
  return apiRequest<AccountProfile>("/account/profile", {
    method: "PATCH",
    authToken: token,
    json: payload
  });
}

export type CheckoutPayload = {
  plan?: string;
  interval?: string;
  product_type: "subscription" | "credits" | "lifetime";
  quantity?: number;
  success_url: string;
  cancel_url: string;
  metadata?: Record<string, string>;
};

export async function createCheckoutSession(token: string, payload: CheckoutPayload): Promise<{ session_id: string; checkout_url: string }> {
  return apiRequest<{ session_id: string; checkout_url: string }>("/stripe/checkout/session", {
    method: "POST",
    authToken: token,
    json: payload
  });
}

export type CreditCheckoutResult = {
  state: "pending" | "processing" | "fulfilled" | "failed" | "expired";
  credits_granted: number;
};

export async function getCreditCheckoutResult(token: string, sessionID: string): Promise<CreditCheckoutResult> {
  return apiRequest(`/billing/credit-checkouts/${encodeURIComponent(sessionID)}`, { authToken: token });
}

export async function confirmCreditCheckout(token: string, sessionID: string): Promise<CreditCheckoutResult> {
  return apiRequest(`/billing/credit-checkouts/${encodeURIComponent(sessionID)}/confirmation`, {
    method: "POST", authToken: token, headers: { "Idempotency-Key": sessionID }
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
  scheduled_change?: SubscriptionView["scheduled_change"];
  pending_payment?: SubscriptionView["pending_payment"];
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
  effective_cancel_at?: string;
  scheduled_change?: { plan: string; billing_cycle: string; effective_at?: string } | null;
  pending_payment?: { expires_at?: string; invoice_url?: string; amount_due: number; currency: string } | null;
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
    effective_cancel_at: data.effective_cancel_at,
    scheduled_change: data.scheduled_change,
    pending_payment: data.pending_payment,
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

export function cancelScheduledChange(token: string): Promise<{status:string}> {
 return apiRequest("/stripe/subscription/scheduled-change/cancel", {method:"POST",authToken:token});
}
export type PurchaseItem = { product_type?:string; id:string; status:string; amount:number|null; currency:string; processed:boolean; created_at:string };
export function listPurchases(token:string,page=1,limit=10):Promise<{items:PurchaseItem[];total:number;page:number;limit:number}> {
 return apiRequest(`/billing/purchases?page=${page}&limit=${limit}`,{authToken:token});
}
export type MembershipView = {plan:string;active:boolean;valid_until?:string;reason:string;source?:string};
export function getMembership(token:string):Promise<MembershipView> {
 return apiRequest("/account/membership",{authToken:token});
}

export type MembershipOverride = { user_id: string; plan: string; expires_at: string | null; updated_at: string };
export type OperatorSubscription = {
 id?: number; user_id?: string; email?: string; provider?: string; provider_subscription_id?: string;
 plan: string; status: string; billing_interval?: string; period_end?: string | null;
 cancel_at_period_end?: boolean; cancel_effective_at?: string | null;
 scheduled_change?: SubscriptionView["scheduled_change"]; payment_pending?: boolean;
 created_at?: string; updated_at?: string;
};
export type OperatorAccess = { membership: MembershipView; billing: MembershipView; override: MembershipOverride | null; user: OperatorUser; subscription: OperatorSubscription | null };
export type OperatorMetricDay = { date: string; registrations: number; referrals: number; payments: number; new_paying_users: number };
export type OperatorOverview = {
 users: number; subscriptions: number; active_subscriptions: number; referrals: number;
 pending_referrals: number; activated_referrals: number; billing_events: number;
 today_registrations: number; today_referrals: number; today_payments: number; paid_users: number;
 timezone: string; mode: "live" | "test"; as_of: string; series: OperatorMetricDay[];
};
export type OperatorReportFilters = { days: 7 | 30; timezone: "Asia/Shanghai" | "UTC"; mode: "live" | "test" };
export type MembershipPatch = { plan: "automatic" | "free" | "starter" | "pro" | "premium" | "lifetime"; expires_at?: string; reason: string };

export type OperatorUser = { id: string; email: string; username: string; role: string; plan: string; membership: MembershipView; subscription?: OperatorSubscription | null; credits: number; email_verified: boolean; created_at: string; last_login_at?: string };
export type OperatorAuthorization = { authorized: true };

export type ContentTranslation = { title: string; summary: string; body: string; seo_title: string; seo_description: string };
/** AI writing (POST /admin/articles/drafts, /admin/articles/translations): nothing is saved until the editor saves the draft. */
export type ArticleImageSuggestion = { section: string; alt: string; description: string };
export type ArticleDraft = { path: string; translation: ContentTranslation; images: ArticleImageSuggestion[] };
export type ArticleBrief = { kind: Exclude<ContentKind, "page">; keyword: string; notes: string; locale: ContentLocale };
export type ArticleTranslations = { translations: Partial<Record<ContentLocale, ContentTranslation>> };
/** An article image from POST /admin/site-images; `url` is public. */
export type SiteImage = { id: string; url: string; width: number; height: number; content_type: string; size: number };
export type ContentPublication = ContentTranslation & { published_at: string };
export type ContentKind = "page" | "blog" | "docs" | "features" | "use-cases" | "compare";
export type ContentArticle = { id: string; path: string; kind: ContentKind; position: number; version: number; translations: Partial<Record<ContentLocale, ContentTranslation>>; published: Partial<Record<ContentLocale, ContentPublication>>; updated_at: string };
export type ContentSave = Pick<ContentArticle, "path" | "kind" | "position" | "version" | "translations"> & { reason: string };
export type PublishedArticle = { references?: string[]; id: string; path: string; kind: ContentKind; position: number; version: number; translation: ContentTranslation; published_at: string; updated_at: string; locales: ContentLocale[] };
/** Sections a header item or footer link may name; mirrors the backend allowlist. */
export type NavigationSection = "home" | "features" | "use-cases" | "compare" | "docs" | "blog" | "resources" | "updates" | "pricing" | "examples" | "about" | "contact";
export type NavigationCopy = { en: string; zh: string };
/** `auto` lists the section's published pages, `manual` lists `items` in order, `none` shows no menu. An empty label uses the section's default name. */
export type HeaderNavigationItem = { section: NavigationSection; label: NavigationCopy; menu: "auto" | "manual" | "none"; items: string[] };
export type FooterNavigationGroup = { title: NavigationCopy; links: NavigationSection[] };
export type SiteNavigation = { version: 1; header: HeaderNavigationItem[]; footer: FooterNavigationGroup[] };
export type WebsiteSettings = { brand_name: string; logo_url?: string; description: string; description_zh?: string; headline_en?: string; headline_zh?: string; sender_email?: string; sender_name?: string; support_email: string; support_url: string; export_credit_cost: number; navigation?: SiteNavigation | null; presentation_version?: number };

export type OperatorOrder = { id: string; user_id: string; email?: string; provider: string; provider_event_id: string; event_type: string; processed: boolean; created_at: string; amount: number | null; currency: string; status: string; record_type: "invoice" | "checkout" | "payment_event"; product_type?: string; plan?: string; billing_interval?: string; quantity?: number; amount_due?: number | null; checkout_session_id?: string; paid_at?: string; expires_at?: string; livemode: boolean | null };

export type ContentImportResult = { applied: boolean; created: number; skipped: number; items: Array<{ path: string; kind: ContentKind; title: string; status: "create" | "skip" }> };
