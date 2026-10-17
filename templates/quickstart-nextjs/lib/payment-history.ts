import type { InvoiceItem, PurchaseItem } from "./api";

type PaymentPage<T> = { items: T[]; page: number; limit: number; total: number };

/** Refresh the loaded window, so returning to a page does not collapse its history. */
export async function reloadLoadedPaymentPages<T>(load: (page: number) => Promise<PaymentPage<T>>, pages: number): Promise<PaymentPage<T>> {
  const first = await load(1);
  const lastPage = Math.min(Math.max(1, pages), Math.max(1, Math.ceil(first.total / first.limit)));
  const remaining = await Promise.all(Array.from({ length: lastPage - 1 }, (_, index) => load(index + 2)));
  return { ...first, page: lastPage, items: [first, ...remaining].flatMap(result => result.items) };
}

export type PaymentRecord = {
  id: string;
  created_at: string;
  purchase?: PurchaseItem;
  invoice?: InvoiceItem;
};

/** Join only the actual provider invoice ID, never amount or date heuristics. */
export function paymentRecords(purchases: readonly PurchaseItem[], invoices: readonly InvoiceItem[]): PaymentRecord[] {
  const records = new Map<string, PaymentRecord>();
  for (const purchase of purchases) records.set(purchase.id, { id: purchase.id, created_at: purchase.created_at, purchase });
  for (const invoice of invoices) {
    const id = `stripe:${invoice.id}`;
    const previous = records.get(id);
    records.set(id, { ...previous, id, created_at: previous?.created_at || invoice.created_at, invoice });
  }
  return [...records.values()].sort((left, right) =>
    (Date.parse(right.created_at) || 0) - (Date.parse(left.created_at) || 0) || right.id.localeCompare(left.id));
}

export function invoiceDownloadURL(invoice?: InvoiceItem): string | null {
  const value = invoice?.pdf_url || invoice?.pdfurl;
  if (!value) return null;
  try {
    const url = new URL(value);
    return url.protocol === "https:" && !url.username && !url.password ? url.href : null;
  } catch { return null; }
}
