import assert from "node:assert/strict";
import test from "node:test";
import { paymentRecords, invoiceDownloadURL } from "../lib/payment-history";
import type { InvoiceItem, PurchaseItem } from "../lib/api";

const purchase = (id: string, created_at = "2026-10-16T00:00:00Z"): PurchaseItem => ({ id, created_at, product_type: "subscription", status: "paid", amount: 1900, currency: "usd", processed: true });
const invoice = (id: string, created_at = "2026-10-16T00:00:00Z"): InvoiceItem => ({ id, created_at, status: "paid", amount_usd: 19, period: "2026-10", pdf_url: "https://invoice.stripe.com/i/fixture-not-a-real-invoice" });

test("one payment row keeps its corresponding invoice download", () => {
  const result = paymentRecords([purchase("stripe:in_one")], [invoice("in_one")]);
  assert.equal(result.length, 1);
  assert.equal(result[0].purchase?.amount, 1900);
  assert.equal(result[0].invoice?.id, "in_one");
  assert.equal(invoiceDownloadURL(result[0].invoice), "https://invoice.stripe.com/i/fixture-not-a-real-invoice");
});

test("invoice-only history and purchases without invoices remain visible", () => {
  const result = paymentRecords([purchase("stripe:cs_credits")], [invoice("in_historical")]);
  assert.equal(result.length, 2);
  assert.ok(result.some(row => row.invoice && !row.purchase));
  assert.ok(result.some(row => row.purchase && !row.invoice));
  assert.equal(invoiceDownloadURL(undefined), null);
});

test("equal amounts and dates never associate unrelated payments or providers", () => {
  const result = paymentRecords([purchase("stripe:in_other"), purchase("another:in_same")], [invoice("in_same")]);
  assert.equal(result.length, 3);
  assert.equal(result.find(row => row.id === "another:in_same")?.invoice, undefined);
});

test("loading another batch deduplicates by ID and keeps newest-first stable order", () => {
  const result = paymentRecords([purchase("stripe:in_old", "2026-09-16T00:00:00Z"), purchase("stripe:in_new"), { ...purchase("stripe:in_new"), status: "unpaid" }], [invoice("in_old"), invoice("in_new"), invoice("in_new")]);
  assert.deepEqual(result.map(row => row.id), ["stripe:in_new", "stripe:in_old"]);
  assert.equal(result[0].purchase?.status, "unpaid");
  assert.deepEqual(paymentRecords([], []), []);
});

test("invoice links accept HTTPS downloads and reject unsafe URL schemes", () => {
  for (const pdf_url of ["javascript:alert(1)", "data:text/html,test", "http://example.test/invoice", "https://user:password@example.test/invoice", "/invoice", ""]) {
    assert.equal(invoiceDownloadURL({ ...invoice("in_test"), pdf_url }), null);
  }
  assert.equal(invoiceDownloadURL({ ...invoice("in_test"), pdf_url: undefined, pdfurl: "https://example.test/invoice.pdf" }), "https://example.test/invoice.pdf");
});
