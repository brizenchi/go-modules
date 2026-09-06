import { apiRequest, ApiError } from "./api";
import { apiUrl } from "./env";
import { clearSessionIfToken } from "./auth";

export type PageResult<T> = { items: T[]; total: number; page: number; limit: number };
export type OperatorOverview = { users: number; subscriptions: number; active_subscriptions: number; referrals: number; pending_referrals: number; activated_referrals: number; billing_events: number };
export type OperatorUser = { id: string; email: string; username: string; role: string; credits: number; email_verified: boolean; created_at: string; last_login_at?: string };
export type OperatorSubscription = { id: number; user_id: string; email?: string; plan: string; status: string; billing_interval?: string; provider: string; provider_subscription_id?: string; period_end?: string; updated_at: string };
export type OperatorOrder = { id: string; user_id: string; email?: string; provider: string; provider_event_id: string; event_type: string; processed: boolean; created_at: string; amount: number | null; currency: string; status: string; record_type: "invoice" | "checkout" | "payment_event"; livemode: boolean | null };
export type OperatorReferral = { id: number; referrer_id: string; referee_id: string; referrer_email?: string; referee_email?: string; code: string; status: string; reward_credits: number; created_at: string; activated_at?: string; expires_at?: string };
export type AuditEntry = { id: number; actor_id: string; action: string; target_id?: string; reason: string; created_at: string; idempotency_key?: string };
export type CreditBalance = { balance: number; expiring_credits: number; next_expiry_at?: string | null };
export type CreditTransaction = { id: number; user_id: string; kind: string; amount: number; balance_after: number; source: string; source_id: string; reason: string; actor_id?: string; expires_at?: string | null; related_transaction_id?: number | null; created_at: string };
export type BusinessSettings = { brand_name: string; description: string; support_email: string; support_url: string; export_credit_cost: number };
export type NoteItem = { id: number; title: string; body: string; created_at: string };
export type NoteExport = { filename: string; content: string; transaction_id: number; balance: number };
export type UploadedImage = { id: string; url: string; content_type: string; size: number; filename: string };

export function queryString(values: Record<string, string | number | undefined>): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value !== undefined && String(value).trim()) query.set(key, String(value).trim());
  }
  return query.toString();
}

export function listOperator<T>(token: string, section: "users" | "subscriptions" | "orders" | "referrals" | "audit", page = 1, query = "", status = "") {
  return apiRequest<PageResult<T>>(`/admin/${section}?${queryString({ page, limit: 20, query, status })}`, { authToken: token });
}
export function getOperatorOverview(token: string) { return apiRequest<OperatorOverview>("/admin/overview", { authToken: token }); }
export function getBusinessSettings(token: string) { return apiRequest<BusinessSettings>("/admin/settings", { authToken: token }); }
export function saveBusinessSettings(token: string, data: BusinessSettings & { reason: string }, key: string) {
  return apiRequest<BusinessSettings>("/admin/settings", { authToken: token, method: "PATCH", headers: { "Idempotency-Key": key }, json: data });
}
export function retryReferralReward(token: string, id: number, reason: string, key: string) {
  return apiRequest(`/admin/referrals/${id}/retry-reward`, { authToken: token, method: "POST", headers: { "Idempotency-Key": key }, json: { reason } });
}
export function getCreditBalance(token: string) { return apiRequest<CreditBalance>("/credits", { authToken: token }); }
export async function listCreditTransactions(token: string, page = 1, userID?: string, admin = false): Promise<PageResult<CreditTransaction>> {
  const result = await apiRequest<{ list: CreditTransaction[]; total: number; page: number; limit: number }>(`${admin ? "/admin" : ""}/credits/transactions?${queryString({ page, limit: 20, user_id: userID })}`, { authToken: token });
  return { ...result, items: result.list };
}
export function grantCredits(token: string, data: { user_id: string; amount: number; expires_at?: string; reason: string; idempotency_key: string }) {
  return apiRequest("/admin/credits/grants", { authToken: token, method: "POST", json: data });
}
export function refundCredits(token: string, data: { transaction_id: number; reason: string; idempotency_key: string }) {
  return apiRequest("/admin/credits/refunds", { authToken: token, method: "POST", json: data });
}
export async function listNotes(token: string): Promise<NoteItem[]> {
  const data = await apiRequest<{list: NoteItem[]}>("/notes", { authToken: token });
  return data.list || [];
}
export function createNote(token: string, title: string, body: string) { return apiRequest<NoteItem>("/notes", { authToken: token, method: "POST", json: { title, body } }); }
export function exportNote(token: string, id: number, key: string, expectedCost: number) {
  return apiRequest<NoteExport>(`/notes/${id}/export`, { authToken: token, method: "POST", json: { idempotency_key: key, expected_cost: expectedCost } });
}
export function uploadImage(token: string, file: File) {
  const body = new FormData();
  body.set("file", file);
  return apiRequest<UploadedImage>("/uploads/images", { authToken: token, method: "POST", body });
}
export async function getPrivateImage(token: string, id: string): Promise<Blob> {
  const response = await fetch(apiUrl(`/uploads/images/${encodeURIComponent(id)}`), { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" });
  if (!response.ok) {
    if (response.status === 401) clearSessionIfToken(token);
    throw new ApiError("Unable to load this image", response.status, response.status);
  }
  return response.blob();
}

export function listUploadedImages(token: string, page = 1) { return apiRequest<PageResult<UploadedImage>>( `/uploads/images?${queryString({ page, limit: 20 })}`, { authToken: token }); }
